package controllers

import (
	"context"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha3"
	capiv1alpha3 "sigs.k8s.io/cluster-api/api/v1alpha3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("VSphereMachine IPAM controller", func() {
	const (
		Namespace   = "default"
		MachineName = "test-machine"
		ClusterName = "test-cluster"
		Template    = "test-template"
		NetworkName = "testNetwork"
		NetworkView = "testview"
		DNSZone     = "test.ipam.telekom.de"
	)
	var (
		TestSubnet = net.IPNet{
			IP:   net.IPv4(10, 0, 0, 0),
			Mask: net.IPv4Mask(255, 255, 255, 0),
		}
		NamespacedName = types.NamespacedName{Namespace: Namespace, Name: MachineName}
		getMeta        = func(uid types.UID) v1.ObjectMeta {
			return v1.ObjectMeta{
				Name:      MachineName,
				Namespace: Namespace,
				Labels: map[string]string{
					clusterNameLabel: ClusterName,
				},
				Annotations: map[string]string{
					annotationPrefix + networkNameParam:         NetworkName,
					annotationPrefix + infobloxNetworkViewParam: NetworkView,
					annotationPrefix + subnetParam:              TestSubnet.String(),
					annotationPrefix + dnsZoneParam:             DNSZone,
					clusterNameLabel:                            ClusterName,
				},
				OwnerReferences: []v1.OwnerReference{{Name: MachineName, Kind: "Machine", APIVersion: "v1alpha3", UID: uid}},
			}
		}
	)

	BeforeEach(func() {
		ipamManager.Callback = nil
	})

	AfterEach(func() {
		vSphereMachine := &v1alpha3.VSphereMachine{}
		err := k8sClient.Get(context.Background(), NamespacedName, vSphereMachine)
		if err == nil || vSphereMachine.Name != "" {
			vSphereMachine.Finalizers = []string{}
			Expect(k8sClient.Update(context.Background(), vSphereMachine)).To(Succeed())
			Expect(k8sClient.Delete(context.Background(), &v1alpha3.VSphereMachine{ObjectMeta: getMeta("")})).To(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), NamespacedName, &v1alpha3.VSphereMachine{})
				return err != nil
			}, timeout, interval).Should(BeTrue())
		}
		Expect(k8sClient.Delete(context.Background(), &capiv1alpha3.Machine{ObjectMeta: v1.ObjectMeta{Name: MachineName, Namespace: Namespace}})).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), NamespacedName, &capiv1alpha3.Machine{})
			return err != nil
		}, timeout, interval).Should(BeTrue())
	})

	Context("it finds a machine with ipam annotations", func() {
		It("handles its full lifecycle", func() {
			ctx := context.Background()
			allocated := false
			released := false
			ipamManager.Callback = func(t, id, networkView string, subnet *net.IPNet) {
				ctrl.Log.Info("ipam callback", "deviceName", id, "networkView", networkView, "subnet", subnet.String())
				if id != MachineName+"."+DNSZone || networkView != NetworkView || subnet.String() != TestSubnet.String() {
					return
				}
				if t == "GetOrAllocate" {
					allocated = true
				}
				if t == "ReleaseIP" {
					released = true
				}
			}
			machine := &capiv1alpha3.Machine{
				ObjectMeta: v1.ObjectMeta{
					Name:      MachineName,
					Namespace: Namespace,
				},
				Spec: capiv1alpha3.MachineSpec{
					ClusterName: ClusterName,
				},
			}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed())
			vSphereMachine := &v1alpha3.VSphereMachine{
				ObjectMeta: getMeta(machine.UID),
				Spec: v1alpha3.VSphereMachineSpec{
					VirtualMachineCloneSpec: v1alpha3.VirtualMachineCloneSpec{
						Network:  v1alpha3.NetworkSpec{Devices: []v1alpha3.NetworkDeviceSpec{{NetworkName: NetworkName}}},
						Template: Template,
					},
				},
			}
			Expect(k8sClient.Create(ctx, vSphereMachine)).To(Succeed())

			By("allocating an IP after creation")
			createdMachine := &v1alpha3.VSphereMachine{}
			// wait for creation
			Eventually(func() bool {
				err := k8sClient.Get(ctx, NamespacedName, createdMachine)
				if err != nil {
					return false
				}
				return checkNetworkDevices(createdMachine.Spec.Network.Devices)
			}, timeout, interval).Should(BeTrue())
			Expect(createdMachine.Finalizers).To(ContainElement(finalizer))
			Expect(allocated).To(BeTrue(), "should allocate the ip in ipam")

			By("releasing the IP on deletion")
			Expect(k8sClient.Delete(context.Background(), &v1alpha3.VSphereMachine{ObjectMeta: getMeta("")})).To(Succeed())
			// wait for deletion
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), NamespacedName, &v1alpha3.VSphereMachine{})
				if err != nil || false {
					return true
				}
				return false
			}, timeout, interval).Should(BeTrue())
			Expect(released).To(BeTrue(), "should release the ip in ipam")
		})

		It("doesn't assign an IP address when DHCP4 is enabled", func() {
			ctx := context.Background()
			called := false
			ipamManager.Callback = func(t, _, _ string, _ *net.IPNet) {
				called = true
			}
			machine := &capiv1alpha3.Machine{
				ObjectMeta: v1.ObjectMeta{
					Name:      MachineName,
					Namespace: Namespace,
				},
				Spec: capiv1alpha3.MachineSpec{
					ClusterName: ClusterName,
				},
			}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed())
			vSphereMachine := &v1alpha3.VSphereMachine{
				ObjectMeta: getMeta(machine.UID),
				Spec: v1alpha3.VSphereMachineSpec{
					VirtualMachineCloneSpec: v1alpha3.VirtualMachineCloneSpec{
						Network:  v1alpha3.NetworkSpec{Devices: []v1alpha3.NetworkDeviceSpec{{DHCP4: true}}},
						Template: Template,
					},
				},
			}
			Expect(k8sClient.Create(ctx, vSphereMachine)).To(Succeed())
			createdMachine := &v1alpha3.VSphereMachine{}
			waitForObject(ctx, NamespacedName, createdMachine)
			Consistently(func() (int, error) {
				err := k8sClient.Get(ctx, NamespacedName, createdMachine)
				if err != nil {
					return -1, err
				}
				return len(createdMachine.Spec.Network.Devices[0].IPAddrs), nil
			}, duration, interval).Should(Equal(0))
			Expect(createdMachine.Finalizers).NotTo(ContainElement(finalizer))
			Expect(called).To(BeFalse(), "should not call ipam")
		})

		It("fetches the annotations of the owning Machine if it doesn't have them", func() {
			ctx := context.Background()
			allocated := false
			ipamManager.Callback = func(t, id, networkView string, subnet *net.IPNet) {
				ctrl.Log.Info("ipam callback", "deviceName", id, "networkView", networkView, "subnet", subnet.String())
				if id != MachineName+"."+DNSZone || networkView != NetworkView || subnet.String() != TestSubnet.String() {
					return
				}
				if t == "GetOrAllocate" {
					allocated = true
				}
			}

			machine := &capiv1alpha3.Machine{
				ObjectMeta: v1.ObjectMeta{
					Name:      MachineName,
					Namespace: Namespace,
					Annotations: map[string]string{
						annotationPrefix + networkNameParam:         NetworkName,
						annotationPrefix + infobloxNetworkViewParam: NetworkView,
						annotationPrefix + subnetParam:              TestSubnet.String(),
						annotationPrefix + dnsZoneParam:             DNSZone,
						clusterNameLabel:                            ClusterName,
					},
				},
				Spec: capiv1alpha3.MachineSpec{
					ClusterName: ClusterName,
				},
			}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed())
			// ctrl.Log.Info("machine created", "uid", machine.ObjectMeta.UID)
			vmachine := &v1alpha3.VSphereMachine{
				ObjectMeta: v1.ObjectMeta{
					Name:      MachineName,
					Namespace: Namespace,
					Labels: map[string]string{
						clusterNameLabel: ClusterName,
					},
					OwnerReferences: []v1.OwnerReference{
						{APIVersion: "v1alpha3", Kind: "Machine", Name: machine.ObjectMeta.Name, UID: machine.ObjectMeta.UID},
					},
				},
				Spec: v1alpha3.VSphereMachineSpec{
					VirtualMachineCloneSpec: v1alpha3.VirtualMachineCloneSpec{
						Template: Template,
						Network:  v1alpha3.NetworkSpec{Devices: []v1alpha3.NetworkDeviceSpec{{NetworkName: NetworkName}}},
					},
				},
			}
			Expect(k8sClient.Create(ctx, vmachine)).To(Succeed())
			createdMachine := &v1alpha3.VSphereMachine{}
			// wait for creation
			Eventually(func() bool {
				err := k8sClient.Get(ctx, NamespacedName, createdMachine)
				if err != nil {
					return false
				}
				return checkNetworkDevices(createdMachine.Spec.Network.Devices)
			}, timeout, interval).Should(BeTrue())
			Expect(allocated).To(BeTrue(), "should allocate the ip in ipam")
		})
	})
	// When("it finds a VSphereMachine with multiple ipam annotations", func() {
	// 	It("handles all specified interfaces", func() {
	// 		ctx := context.Background()
	// 		allocated := []string{}
	// 		released := []string{}
	// 		ipamManager.Callback = func(t, id, networkView string, subnet *net.IPNet) {
	// 			ctrl.Log.Info("ipam callback", "deviceName", id, "networkView", networkView, "subnet", subnet.String())
	// 			if id != MachineName || networkView != NetworkView {
	// 				return
	// 			}
	// 			if t == "GetOrAllocate" {
	// 				allocated = append(allocated, subnet.String())
	// 			}
	// 			if t == "ReleaseIP" {
	// 				released = append(released, subnet.String())
	// 			}
	// 		}
	// 		machine := &capiv1alpha3.Machine{
	// 			ObjectMeta: v1.ObjectMeta{
	// 				Name:      MachineName,
	// 				Namespace: Namespace,
	// 				Annotations: map[string]string{
	// 					annotationPrefix + networkNameParam:         NetworkName,
	// 					annotationPrefix + infobloxNetworkViewParam: NetworkView,
	// 					annotationPrefix + subnetParam:              TestSubnet.String(),
	// 					clusterNameLabel:                            ClusterName,
	// 					annotationPrefix + dnsZoneParam:             DNSZone,
	// 				},
	// 			},
	// 			Spec: capiv1alpha3.MachineSpec{
	// 				ClusterName: ClusterName,
	// 			},
	// 		}
	// 		Expect(k8sClient.Create(ctx, machine)).To(Succeed())
	// 		vSphereMachine := &v1alpha3.VSphereMachine{
	// 			ObjectMeta: v1.ObjectMeta{
	// 				Name:      MachineName,
	// 				Namespace: Namespace,
	// 				Labels: map[string]string{
	// 					clusterNameLabel: ClusterName,
	// 				},
	// 				Annotations: map[string]string{
	// 					"ipam.schiff.telekom.de/0-NetworkName":         NetworkName,
	// 					"ipam.schiff.telekom.de/0-InfobloxNetworkView": NetworkView,
	// 					"ipam.schiff.telekom.de/0-Subnet":              TestSubnet.String(),
	// 					"ipam.schiff.telekom.de/0-DNSZone":             DNSZone,
	// 					"ipam.schiff.telekom.de/1-NetworkName":         NetworkName + "2",
	// 					"ipam.schiff.telekom.de/1-InfobloxNetworkView": NetworkView,
	// 					"ipam.schiff.telekom.de/1-Subnet":              "10.1.0.0/24",
	// 					"ipam.schiff.telekom.de/1-DNSZone":             DNSZone,
	// 					clusterNameLabel:                               ClusterName,
	// 				},
	// 				OwnerReferences: []v1.OwnerReference{v1.OwnerReference{Name: "test-owner", Kind: "Machine", APIVersion: "v1alpha3", UID: machine.UID}},
	// 			},
	// 			Spec: v1alpha3.VSphereMachineSpec{
	// 				VirtualMachineCloneSpec: v1alpha3.VirtualMachineCloneSpec{
	// 					Network: v1alpha3.NetworkSpec{Devices: []v1alpha3.NetworkDeviceSpec{
	// 						{NetworkName: NetworkName},
	// 						{NetworkName: NetworkName + "2"},
	// 					}},
	// 					Template: Template,
	// 				},
	// 			},
	// 		}
	// 		Expect(k8sClient.Create(ctx, vSphereMachine)).To(Succeed())

	// 		By("allocating IPs for all interfaces")
	// 		createdMachine := &v1alpha3.VSphereMachine{}
	// 		// wait for creation
	// 		Eventually(func() bool {
	// 			err := k8sClient.Get(ctx, NamespacedName, createdMachine)
	// 			if err != nil {
	// 				return false
	// 			}
	// 			if len(createdMachine.Spec.Network.Devices) < 2 {
	// 				return false
	// 			}
	// 			for i, expectedIP := range []net.IP{net.IPv4(10, 0, 0, 1), net.IPv4(10, 1, 0, 1)} {
	// 				dev := createdMachine.Spec.Network.Devices[i]
	// 				if len(dev.IPAddrs) < 1 {
	// 					return false
	// 				}
	// 				ip, netw, err := net.ParseCIDR(dev.IPAddrs[0])
	// 				if err != nil || !ip.Equal(expectedIP) {
	// 					return false
	// 				}
	// 				if l, _ := netw.Mask.Size(); l != 24 {
	// 					return false
	// 				}
	// 			}

	// 			return true
	// 		}, timeout, interval).Should(BeTrue())
	// 		Expect(createdMachine.Finalizers).To(ContainElement(finalizer))
	// 		Expect(allocated).To(ContainElements([]string{TestSubnet.String(), "10.1.0.0/24"}), "should allocate the ip in ipam")

	// 		By("releasing the IPs on deletion")
	// 		Expect(k8sClient.Delete(context.Background(), &v1alpha3.VSphereMachine{ObjectMeta: getMeta("")})).To(Succeed())
	// 		// wait for deletion
	// 		Eventually(func() bool {
	// 			err := k8sClient.Get(context.Background(), NamespacedName, &v1alpha3.VSphereMachine{})
	// 			if err != nil || false {
	// 				return true
	// 			}
	// 			return false
	// 		}, timeout, interval).Should(BeTrue())
	// 		Expect(released).To(ContainElements([]string{TestSubnet.String(), "10.1.0.0/24"}), "should release the ip in ipam")

	// 	})
	// })
})

func waitForObject(ctx context.Context, key types.NamespacedName, obj client.Object) {
	Eventually(func() bool {
		err := k8sClient.Get(ctx, key, obj)
		return err == nil
	}, timeout, interval).Should(BeTrue())
}

func checkNetworkDevices(devices []v1alpha3.NetworkDeviceSpec) bool {
	if len(devices) < 1 {
		return false
	}
	dev := devices[0]
	if len(dev.IPAddrs) < 1 {
		return false
	}
	ip, netw, err := net.ParseCIDR(dev.IPAddrs[0])
	if err != nil || !ip.Equal(net.IPv4(10, 0, 0, 1)) {
		return false
	}
	if l, _ := netw.Mask.Size(); l != 24 {
		return false
	}
	return true
}
