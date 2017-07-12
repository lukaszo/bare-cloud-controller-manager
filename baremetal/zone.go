package baremetal

import "k8s.io/kubernetes/pkg/cloudprovider"

type BareMetalZones struct {
}

// GetZone is an implementation of Zones.GetZone
func (b *BareMetalZones) GetZone() (cloudprovider.Zone, error) {
	return cloudprovider.Zone{
		FailureDomain: "FailureDomain1",
		Region:        "Region1",
	}, nil
}
