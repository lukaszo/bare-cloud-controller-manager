package baremetal

import (
	"fmt"

	"github.com/golang/glog"
	api "k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/cloudprovider"
)

type BareMetalLoadBalancer struct {
	balancers map[string]bool
}

func NewBareMetalLoadBalancer() *BareMetalLoadBalancer {
	return &BareMetalLoadBalancer{
		balancers: map[string]bool{},
	}
}

// GetLoadBalancer is an implementation of LoadBalancer.GetLoadBalancer
func (b *BareMetalLoadBalancer) GetLoadBalancer(clusterName string, service *api.Service) (status *api.LoadBalancerStatus, exists bool, retErr error) {
	name := cloudprovider.GetLoadBalancerName(service)
	glog.Infof("GetLoadBalancer [%s]", name)

	_, ok := b.balancers[name]
	if !ok {
		glog.Infof("Can't find lb by name [%s]", name)
		return &api.LoadBalancerStatus{}, false, nil
	}

	return lbStatus(), true, nil
}

// EnsureLoadBalancer is an implementation of LoadBalancer.EnsureLoadBalancer.
func (b *BareMetalLoadBalancer) EnsureLoadBalancer(clusterName string, service *api.Service, nodes []*api.Node) (*api.LoadBalancerStatus, error) {
	name := cloudprovider.GetLoadBalancerName(service)
	glog.Infof("GetLoadBalancer [%s]", name)
	b.balancers[name] = true
	return lbStatus(), nil
}

// UpdateLoadBalancer is an implementation of LoadBalancer.UpdateLoadBalancer.
func (b *BareMetalLoadBalancer) UpdateLoadBalancer(clusterName string, service *api.Service, nodes []*api.Node) error {
	name := cloudprovider.GetLoadBalancerName(service)
	glog.Infof("UpdateLoadBalancer [%s] [%s]", name, nodes)

	_, ok := b.balancers[name]
	if !ok {
		glog.Infof("Can't find lb by name [%s]", name)
		return fmt.Errorf("Couldn't find LB with name %s", name)
	}
	return nil
}

// EnsureLoadBalancerDeleted is an implementation of LoadBalancer.EnsureLoadBalancerDeleted.
func (b *BareMetalLoadBalancer) EnsureLoadBalancerDeleted(clusterName string, service *api.Service) error {
	name := cloudprovider.GetLoadBalancerName(service)
	glog.Infof("EnsureLoadBalancerDeleted [%s]", name)
	_, ok := b.balancers[name]
	if !ok {
		glog.Infof("Couldn't find LB %s to delete. Nothing to do.")
		return nil
	}
	delete(b.balancers, name)
	return nil
}

func lbStatus() *api.LoadBalancerStatus {
	return &api.LoadBalancerStatus{
		Ingress: []api.LoadBalancerIngress{api.LoadBalancerIngress{IP: "1.1.1.1"}},
	}
}
