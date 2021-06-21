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

package mock

import (
	"net"

	"gitlab.devops.telekom.de/schiff/engine/schiff-operator.git/pkg/ipam"
)

type Manager struct {
	Callback func(t, identifier, networkView string, subnet *net.IPNet)
}

var _ ipam.Manager = &Manager{}

// func (m *Manager) GetClusterSubnet(clusterIdentifier string, spec ipam.NetworkSpec) (ipam.Subnet, error) {
// 	return Subnet{cid: clusterIdentifier}, nil
// }

func (m *Manager) GetOrAllocateIP(identifier, networkView string, subnet *net.IPNet) (net.IP, error) {
	if m.Callback != nil {
		m.Callback("GetOrAllocate", identifier, networkView, subnet)
	}
	ip := subnet.IP
	ip[3] += 1
	return ip, nil
}

func (m *Manager) ReleaseAllIPs(identifier, networkView string) error {
	if m.Callback != nil {
		m.Callback("ReleaseIP", identifier, networkView, nil)
	}
	return nil
}
