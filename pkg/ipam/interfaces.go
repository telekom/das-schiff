package ipam

import (
	"net"
)

// Manager is an interface to an IPAM backend.
type Manager interface {
	// GetClusterSubnet(clusterIdentifier string, spec NetworkSpec) (Subnet, error)

	// GetOrAllocateIP assigns a single IP address within a subnet that's assigned to a cluster
	// The identifier is used as the device name in the IPAM and the clusterIdentifier is used
	// to identify the subnet.
	GetOrAllocateIP(deviceName, networkView string, subnet *net.IPNet) (net.IP, error)

	// ReleaseIP releases a single IP address within a subnet that's assigned to a cluster.
	// The identifier is the device name within the IPAM and the clusterIdentifier is used
	// to identify the subnet.
	ReleaseIP(deviceName, networkView string, subnet *net.IPNet) error
}

// type Subnet interface {
// 	Identifier() string
// 	Network() *net.IPNet
// }

// type NetworkSpec struct{}
