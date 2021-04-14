/*
Copyright 2021 Deutsche Telekom Technik GmbH.
*/

package controllers

import (
	"context"
	"errors"
	"net"

	"github.com/go-logr/logr"
	"gitlab.devops.telekom.de/schiff/engine/schiff-operator.git/pkg/ipam"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha3"
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
const finalizer = "ipam.schiff.telekom.de/DeallocateMachineIP"
const networkNameAnnotation = "ipam.schiff.telekom.de/NetworkName"
const subnetAnnotation = "ipam.schiff.telekom.de/Subnet"
const infobloxNetworkViewAnnotation = "ipam.schiff.telekom.de/InfobloxNetworkView"

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vspheremachines,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vspheremachines/status,verbs=get
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=vspheremachines/finalizers,verbs=update

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
	subnetAnno, ok := vSphereMachine.Annotations[subnetAnnotation]
	if !ok {
		log.V(2).Info("missing subnet annotation")
		return ctrl.Result{}, nil
	}
	_, subnet, err := net.ParseCIDR(subnetAnno)
	if err != nil {
		log.Error(err, "failed to parse subnet CIDR", "subnet", subnetAnno)
		return ctrl.Result{}, err
	}
	ibNetworkView, ok := vSphereMachine.Annotations[infobloxNetworkViewAnnotation]
	if !ok {
		log.V(2).Info("missing network zone annotation")
		return ctrl.Result{}, nil
	}

	// Deallocate the IP if the Machine is marked for deletion
	hasFinalizer := controllerutil.ContainsFinalizer(&vSphereMachine, finalizer)
	if vSphereMachine.DeletionTimestamp != nil {
		if hasFinalizer {
			log.Info("machine deleted, releasing ip")
			err := r.IPAM.ReleaseIP(vSphereMachine.Name, ibNetworkView, subnet)
			if err != nil {
				log.Error(err, "failed to release ip address")
				return ctrl.Result{}, err
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

	networkName, ok := vSphereMachine.Annotations[networkNameAnnotation]
	if !ok {
		log.V(2).Info("missing network name annotation")
		return ctrl.Result{}, nil
	}

	dev, devIdx := getDeviceByNetworkName(vSphereMachine.Spec.Network.Devices, networkName)
	if devIdx < 0 {
		err := errors.New("device with annotated network name not found")
		log.WithValues("network name", networkName).Error(err, "could not set manual IP")
		return ctrl.Result{}, err
	}

	if dev.DHCP4 || dev.DHCP6 {
		err := errors.New("dhcp enabled on device")
		log.Error(err, "could set manual IP")
		return ctrl.Result{}, err
	}

	desiredIP, err := r.IPAM.GetOrAllocateIP(vSphereMachine.Name, ibNetworkView, subnet)
	if err != nil {
		log.Error(err, "failed to retrieve desired IP")
		return ctrl.Result{}, err
	}
	log.V(4).WithValues("ipAddress", desiredIP).Info("fetched ip address from ipam")

	exists := false
	for _, ip := range dev.IPAddrs {
		if net.ParseIP(ip).Equal(desiredIP) {
			exists = true
			break
		}
	}

	changed := false

	if !exists {
		log.WithValues("ipAddress", desiredIP.String()).Info("adding allocated ip address to machine")
		vSphereMachine.Spec.Network.Devices[devIdx].IPAddrs = append(dev.IPAddrs, desiredIP.String())
		changed = true
	}

	if !hasFinalizer {
		controllerutil.AddFinalizer(&vSphereMachine, finalizer)
		changed = true
	}

	if changed {
		err := r.Client.Update(ctx, &vSphereMachine)
		if err != nil {
			log.Error(err, "failed to update VSphereMachine")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func getDeviceByNetworkName(devices []v1alpha3.NetworkDeviceSpec, name string) (dev v1alpha3.NetworkDeviceSpec, index int) {
	for i, d := range devices {
		if d.NetworkName == name {
			return d, i
		}
	}
	return v1alpha3.NetworkDeviceSpec{}, -1
}

// SetupWithManager sets up the controller with the Manager.
func (r *VSphereMachineIPAMReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&v1alpha3.VSphereMachine{}).
		Complete(r)
}
