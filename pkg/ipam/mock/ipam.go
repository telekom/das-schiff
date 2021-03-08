package mock

import (
	"net"

	"gitlab.devops.telekom.de/schiff/engine/schiff-operator.git/pkg/ipam"
)

type Manager struct {
	Callback func(t, identifier, clusterIdentifier string)
}
type Subnet struct {
	cid string
}

var _ ipam.Manager = &Manager{}

// func (m *Manager) GetClusterSubnet(clusterIdentifier string, spec ipam.NetworkSpec) (ipam.Subnet, error) {
// 	return Subnet{cid: clusterIdentifier}, nil
// }

func (m *Manager) GetOrAllocateIP(identifier, clusterIdentifier string) (net.IP, error) {
	if m.Callback != nil {
		m.Callback("GetOrAllocate", identifier, clusterIdentifier)
	}
	return net.IPv4zero, nil
}

func (m *Manager) ReleaseIP(identifier, clusterIdentifier string) error {
	if m.Callback != nil {
		m.Callback("ReleaseIP", identifier, clusterIdentifier)
	}
	return nil
}

func (s Subnet) Identifier() string {
	return s.cid
}

func (s Subnet) Network() *net.IPNet {
	return &net.IPNet{}
}
