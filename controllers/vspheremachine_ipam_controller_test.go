package controllers

import (
	"context"
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api-provider-vsphere/api/v1alpha3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("VSphereMachine IPAM controller", func() {
	const (
		Namespace   = "default"
		MachineName = "test-machine"
		ClusterName = "test-cluster"
		Template    = "test-template"
		NetworkName = "testNetwork"
	)
	var (
		meta = v1.ObjectMeta{
			Name:      MachineName,
			Namespace: Namespace,
			Labels: map[string]string{
				clusterNameLabel: ClusterName,
			},
			Annotations: map[string]string{
				networkNameAnnotation: NetworkName,
				clusterNameLabel:      ClusterName,
			},
		}
		NamespacedName = types.NamespacedName{Namespace: Namespace, Name: MachineName}
	)

	BeforeEach(func() {
		ipamManager.Callback = nil
	})

	AfterEach(func() {
		machine := &v1alpha3.VSphereMachine{}
		err := k8sClient.Get(context.Background(), NamespacedName, machine)
		if err != nil || machine.Name == "" {
			return
		}
		machine.Finalizers = []string{}
		Expect(k8sClient.Update(context.Background(), machine)).To(Succeed())
		Expect(k8sClient.Delete(context.Background(), &v1alpha3.VSphereMachine{ObjectMeta: meta})).To(Succeed())
		Eventually(func() bool {
			err := k8sClient.Get(context.Background(), NamespacedName, &v1alpha3.VSphereMachine{})
			if err != nil {
				return true
			}
			return false
		}, timeout, interval).Should(BeTrue())
	})

	Context("when it finds a machine without an ip address", func() {
		It("handles its full lifecycle", func() {
			ctx := context.Background()
			allocated := false
			released := false
			ipamManager.Callback = func(t, id, cid string) {
				if id != MachineName || cid != ClusterName {
					return
				}
				if t == "GetOrAllocate" {
					allocated = true
				}
				if t == "ReleaseIP" {
					released = true
				}
			}
			machine := &v1alpha3.VSphereMachine{
				ObjectMeta: meta,
				Spec: v1alpha3.VSphereMachineSpec{
					VirtualMachineCloneSpec: v1alpha3.VirtualMachineCloneSpec{
						Network:  v1alpha3.NetworkSpec{Devices: []v1alpha3.NetworkDeviceSpec{{NetworkName: NetworkName}}},
						Template: Template,
					},
				},
			}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed())

			By("allocating an IP after creation")
			createdMachine := &v1alpha3.VSphereMachine{}
			// wait for creation
			Eventually(func() bool {
				err := k8sClient.Get(ctx, NamespacedName, createdMachine)
				if err != nil {
					return false
				}
				if len(createdMachine.Spec.Network.Devices) < 1 {
					return false
				}
				dev := createdMachine.Spec.Network.Devices[0]
				if len(dev.IPAddrs) < 1 || !net.ParseIP(dev.IPAddrs[0]).Equal(net.IPv4zero) {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
			Expect(createdMachine.Finalizers).To(ContainElement(finalizer))
			Expect(allocated).To(BeTrue(), "should allocate the ip in ipam")

			By("releasing the IP on deletion")
			Expect(k8sClient.Delete(context.Background(), &v1alpha3.VSphereMachine{ObjectMeta: meta})).To(Succeed())
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
			ipamManager.Callback = func(t, _, _ string) {
				called = true
			}
			machine := &v1alpha3.VSphereMachine{
				ObjectMeta: meta,
				Spec: v1alpha3.VSphereMachineSpec{
					VirtualMachineCloneSpec: v1alpha3.VirtualMachineCloneSpec{
						Network:  v1alpha3.NetworkSpec{Devices: []v1alpha3.NetworkDeviceSpec{{DHCP4: true}}},
						Template: Template,
					},
				},
			}
			Expect(k8sClient.Create(ctx, machine)).To(Succeed())
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
	})
})

func waitForObject(ctx context.Context, key types.NamespacedName, obj client.Object) {
	Eventually(func() bool {
		err := k8sClient.Get(ctx, key, obj)
		if err != nil {
			return false
		}
		return true
	}, timeout, interval).Should(BeTrue())
}
