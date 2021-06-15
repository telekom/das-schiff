package ipam

import (
	"net"
)

// Manager is an interface to an IPAM backend.
type Manager interface {
	// GetOrAllocateIP assigns a single IP address within a subnet that's assigned to a cluster
	GetOrAllocateIP(deviceFQDN, networkView string, subnet *net.IPNet) (net.IP, error)

	// ReleaseAllIPs releases all IPs for a host with the provided deviceName
	ReleaseAllIPs(deviceFQDN, networkView string) error
}

// type Subnet interface {
// 	Identifier() string
// 	Network() *net.IPNet
// }

// type NetworkSpec struct{}
