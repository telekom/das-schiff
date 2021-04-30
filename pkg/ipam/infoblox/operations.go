package infoblox

import (
	"net"

	ibclient "github.com/infobloxopen/infoblox-go-client"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

type Manager struct{}

// Initializes a connection to Infoblox
func (m *Manager) getIBConnector() (*ibclient.Connector, error) {
	hostConfig := ibclient.HostConfig{
		Host:     viper.GetString("ipam.infoblox.host"),
		Version:  viper.GetString("ipam.infoblox.wapi_version"),
		Port:     viper.GetString("ipam.infoblox.port"),
		Username: viper.GetString("ipam.infoblox.username"),
		Password: viper.GetString("ipam.infoblox.password"),
	}

	transportConfig := ibclient.NewTransportConfig("false", 15, 10)
	requestBuilder := &ibclient.WapiRequestBuilder{}
	requestor := &ibclient.WapiHttpRequestor{}
	conn, err := ibclient.NewConnector(hostConfig, transportConfig, requestBuilder, requestor)
	if err != nil {
		log.Error(err, "Could not establish a connection to Infoblox Client")
		return nil, err
	}
	return conn, nil
}

// GetorAllocateIP retrieves a reserved IP address from the subnet, if no IP has been reserved, it reserves
// the next available IP address in the subnet.

func (m *Manager) GetOrAllocateIP(deviceName, networkView string, subnet *net.IPNet) (net.IP, error) {
	conn, err := m.getIBConnector()
	if err != nil {
		log.Error(err, "Cannot initiate connection")
	}
	defer conn.Logout()
	objMgr := ibclient.NewObjectManager(conn, "myclient", "")
	objMgr.OmitCloudAttrs = true // Needs to be set for on-prem version of Infoblox
	ea := make(ibclient.EA)
	fixedAddress, err := objMgr.GetFixedAddress(networkView, subnet.String(), "", "")
	if err != nil {
		log.Error(err, "Could not get assigned IP address for cluster")
	}
	if fixedAddress != nil {
		log.Info("IP Address already assigned to cluster")
		return net.ParseIP(fixedAddress.IPAddress), err
	} else {
		log.Info("No IP allocated to cluster, allocating IP")
		// AllocateIP assigns first available IP to  the cluster.
		allocatedIP, err := objMgr.AllocateIP(networkView, subnet.String(), "", "", deviceName, ea)
		if err != nil {
			log.Error(err, "Could not allocate IP for cluster")
		}
		log.Info("IP address allocated successfully to cluster")
		return net.ParseIP(allocatedIP.IPAddress), err
	}
}

// ReleaseIP releases a single IP address within a subnet that's assigned to a cluster.
func (m *Manager) ReleaseIP(deviceName, networkView string, subnet *net.IPNet) error {
	conn, err := m.getIBConnector()
	if err != nil {
		return err
	}
	defer conn.Logout()
	objMgr := ibclient.NewObjectManager(conn, "myclient", "")
	objMgr.OmitCloudAttrs = true // Needs to be set for on-prem version of Infoblox
	_, err = objMgr.ReleaseIP(networkView, string(subnet.IP)+"/"+string(subnet.Mask), "", "")
	if err != nil {
		log.Error("Could not release IP for cluster")
		return err
	}
	log.Info("IP address released for cluster")
	return err
}
