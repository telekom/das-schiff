// Copyright 2021 Deutsche Telekom AG
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	"context"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/go-logr/logr"
	"gitlab.devops.telekom.de/schiff/engine/schiff-operator.git/pkg/ipam"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// VSphereMachineIPAMReconciler reconciles a VSphereMachine object
type VSphereMachineIPAMReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
	IPAM   ipam.Manager
}

const clusterNameLabel = "cluster.x-k8s.io/cluster-name"
const finalizer = "ipam.schiff.telekom.de/ReleaseIPs"
const annotationPrefix = "ipam.schiff.telekom.de/"
const networkNameParam = "NetworkName"
const subnetParam = "Subnet"
const infobloxNetworkViewParam = "InfobloxNetworkView"
const dnsZoneParam = "DNSZone"

var annotationRegex = regexp.MustCompile(`ipam\.schiff\.telekom\.de\/(\d+-)Subnet`)

// var errMissingParam = errors.New("object is missing required parameters")
type errMissingParam error

type interfaceConfig struct {
	subnet              *net.IPNet
	networkName         string
	infobloxNetworkView string
	dnsZone             string
}

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vspheremachines,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vspheremachines/status,verbs=get
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vspheremachines/finalizers,verbs=update
// +kubebuilder:rbac:groups=cluster.x-k8s.io,resources=machines,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *VSphereMachineIPAMReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("vspheremachine", req.NamespacedName)

	log.V(4).Info("reconceiling VSphereMachine")

	var vSphereMachine v1alpha3.VSphereMachine
	if err := r.Client.Get(ctx, req.NamespacedName, &vSphereMachine); err != nil {
		// deleted objects somtimes still trigger reconciliation, we'll just ignore those.
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch VSphereMachine")
		return ctrl.Result{}, err
	}

	clusterName := vSphereMachine.Labels[clusterNameLabel]
	if clusterName == "" {
		err := errors.New("missing label " + clusterNameLabel)
		log.Error(err, "cannot get cluster name")
		return ctrl.Result{}, err
	}

	interfaces, err := r.getInterfacesFromAnnotations(ctx, vSphereMachine.ObjectMeta)
	if err != nil {
		if err, ok := err.(errMissingParam); ok {
			log.V(2).Info(err.Error())
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	machineName := getOwnerMachineName(vSphereMachine.ObjectMeta)
	if machineName == "" {
		return ctrl.Result{}, err
	}

	// Deallocate the IP if the Machine is marked for deletion
	hasFinalizer := controllerutil.ContainsFinalizer(&vSphereMachine, finalizer)
	if vSphereMachine.DeletionTimestamp != nil {
		if hasFinalizer {
			log.Info("machine deleted, releasing ip")
			errs := []error{}
			type ibhost struct {
				fqdn    string
				netview string
			}
			hosts := map[ibhost]struct{}{}
			for _, i := range interfaces {
				hosts[ibhost{fqdn: machineName + "." + i.dnsZone, netview: i.infobloxNetworkView}] = struct{}{}
			}
			for h := range hosts {
				err := r.IPAM.ReleaseAllIPs(h.fqdn, h.netview)
				// possible toDo: validate all IPs are actually free and no other host claims tem before releasing
				if err != nil && err.Error() != "no host record found" {
					log.Error(err, "failed to release ip address")
					errs = append(errs, err)
				}
			}
			if len(errs) > 0 {
				return ctrl.Result{}, errs[0]
			}

			controllerutil.RemoveFinalizer(&vSphereMachine, finalizer)

			err = r.Client.Update(ctx, &vSphereMachine)
			if err != nil {
				log.Error(err, "failed to update VSphereMachine")
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}

		log.V(4).Info("machine deleted, ignoring")
		return ctrl.Result{}, nil
	}

	//always set finalizers before perforin any creation action in IPAM
	if !hasFinalizer {
		controllerutil.AddFinalizer(&vSphereMachine, finalizer)
		err := r.Client.Update(ctx, &vSphereMachine)
		if err != nil {
			log.Error(err, "failed to add finalizer to VSphereMachine")
			return ctrl.Result{}, err
		}
	}

	changed := false
	for _, i := range interfaces {
		c, err := r.reconcileInterface(log, &vSphereMachine, machineName, i)
		if err != nil {
			return ctrl.Result{}, err
		}
		changed = changed || c
	}

	if changed {
		log.Info("Trying to update vSphereMachine")

		err := r.Client.Update(ctx, &vSphereMachine)
		if err != nil {
			log.Error(err, "failed to update VSphereMachine")
			return ctrl.Result{}, err
		}

	}

	return ctrl.Result{}, nil
}

func (r *VSphereMachineIPAMReconciler) reconcileInterface(log logr.Logger, vSphereMachine *v1alpha3.VSphereMachine, machineName string, i interfaceConfig) (
	changed bool, err error) {
	log = log.WithValues("subnet", i.subnet)
	dev, devIdx := getDeviceByNetworkName(vSphereMachine.Spec.Network.Devices, i.networkName)
	if devIdx < 0 {
		err := errors.New("device with annotated network name not found")
		log.WithValues("network name", i.networkName).Error(err, "could not set manual IP")
		return false, err
	}

	if dev.DHCP4 || dev.DHCP6 {
		err := errors.New("dhcp enabled on device")
		log.Error(err, "could not set manual IP")
		return false, err
	}

	desiredIP, err := r.IPAM.GetOrAllocateIP(machineName+"."+i.dnsZone, i.infobloxNetworkView, i.subnet)
	if err != nil && desiredIP != nil {
		log.Error(err, "failed to retrieve desired IP")
		return false, err
	}
	log.V(4).WithValues("ipAddress", desiredIP).Info("fetched ip address from ipam")

	exists := false
	for _, ip := range dev.IPAddrs {
		if mIP, mSubnet, err := net.ParseCIDR(ip); err == nil && mIP.Equal(desiredIP) && mSubnet.Mask.String() == i.subnet.Mask.String() {
			exists = true
			break
		}
	}

	if !exists {
		if desiredIP != nil {
			prefix, _ := i.subnet.Mask.Size()
			cidr := fmt.Sprintf("%v/%v", desiredIP.String(), prefix)
			if len(vSphereMachine.Spec.Network.Devices[devIdx].IPAddrs) != 0 {
				log.V(4).WithValues("ipAddresses", vSphereMachine.Spec.Network.Devices[devIdx].IPAddrs).Info("already existing ips are ")
			}
			log.WithValues("ipAddress", cidr).Info("adding allocated ip address to machine")
			vSphereMachine.Spec.Network.Devices[devIdx].IPAddrs = append(dev.IPAddrs, cidr)
			changed = true
		} else {
			err = errors.New("return form IPAM Warpper was invalid")
			return false, err
		}
	}
	return
}

func getDeviceByNetworkName(devices []v1alpha3.NetworkDeviceSpec, name string) (dev v1alpha3.NetworkDeviceSpec, index int) {
	for i, d := range devices {
		if d.NetworkName == name {
			return d, i
		}
	}
	return v1alpha3.NetworkDeviceSpec{}, -1
}

func getInterfaceFromAnnotations(annotations map[string]string, prefix string) (interfaceConfig, error) {
	i := interfaceConfig{}
	var err error
	subnetAnno, ok := annotations[annotationPrefix+prefix+subnetParam]
	if !ok {
		err = errMissingParam(errors.New("missing subnet annotation"))
		return interfaceConfig{}, err
	}
	_, i.subnet, err = net.ParseCIDR(subnetAnno)
	if err != nil {
		err = fmt.Errorf("failed to parse subnet CIDR: %v", err)
		return interfaceConfig{}, err
	}

	i.infobloxNetworkView, ok = annotations[annotationPrefix+prefix+infobloxNetworkViewParam]
	if !ok {
		err = errMissingParam(errors.New("missing network zone annotation"))
		return interfaceConfig{}, err
	}

	i.networkName, ok = annotations[annotationPrefix+prefix+networkNameParam]
	if !ok {
		err = errMissingParam(errors.New("missing network name annotation"))
		return interfaceConfig{}, err
	}

	i.dnsZone, ok = annotations[annotationPrefix+prefix+dnsZoneParam]
	if !ok {
		err = errMissingParam(errors.New("missing dns zone annotation"))
		return interfaceConfig{}, err
	}
	return i, nil
}

func (r *VSphereMachineIPAMReconciler) getInterfacesFromAnnotations(ctx context.Context, metadata v1.ObjectMeta) ([]interfaceConfig, error) {
	annotations := metadata.GetAnnotations()
	containsPrefix := false
	for k := range annotations {
		if strings.HasPrefix(k, annotationPrefix) {
			containsPrefix = true
			break
		}
	}
	if !containsPrefix {
		for _, ownerRef := range metadata.OwnerReferences {
			if ownerRef.Kind == "Machine" {
				machine := &capiv1alpha3.Machine{}
				err := r.Get(ctx, types.NamespacedName{Namespace: metadata.GetNamespace(), Name: ownerRef.Name}, machine)
				if err != nil {
					return nil, err
				}
				annotations = machine.GetAnnotations()
			}
		}
	}

	interfaces := []interfaceConfig{}
	for k := range annotations {
		res := annotationRegex.FindStringSubmatch(k)
		if len(res) > 1 {
			i, err := getInterfaceFromAnnotations(annotations, res[1])
			if err != nil {
				return nil, err
			}
			interfaces = append(interfaces, i)
		}
	}

	if len(interfaces) == 0 {
		i, err := getInterfaceFromAnnotations(annotations, "")
		if err != nil {
			return nil, err
		}
		interfaces = append(interfaces, i)
	}

	return interfaces, nil
}

func getOwnerMachineName(metadata v1.ObjectMeta) string {
	for _, ownerRef := range metadata.OwnerReferences {
		if ownerRef.Kind == "Machine" {
			return ownerRef.Name
		}
	}
	return ""
}

// SetupWithManager sets up the controller with the Manager.
func (r *VSphereMachineIPAMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&v1alpha3.VSphereMachine{}).
		Complete(r)
}
