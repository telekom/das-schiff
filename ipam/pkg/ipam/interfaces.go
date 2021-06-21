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
