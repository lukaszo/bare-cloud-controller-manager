package baremetal

import (
	"fmt"
	"html/template"
	"io"
	"os"

	"github.com/golang/glog"
	api "k8s.io/kubernetes/pkg/api/v1"
	"k8s.io/kubernetes/pkg/cloudprovider"
)

type Endpoints []*Endpoint

type Frontend struct {
	Name     string
	Port     int32
	Backend  string
	ListenIp string
	// Protocol string TODO: do it via annotations like aws
}

type Backend struct {
	Name      string
	Endpoints Endpoints
	// Protocol string TODO
}

type Endpoint struct {
	Name string
	IP   string
	Port int32
}

type BareMetalLoadBalancer struct {
	frontends map[int32]*Frontend
	backends  map[string]*Backend
	template  string
	ip        string
}

func NewBareMetalLoadBalancer(ip string) *BareMetalLoadBalancer {
	return &BareMetalLoadBalancer{
		frontends: map[int32]*Frontend{},
		backends:  map[string]*Backend{},
		template:  "/haproxy_template.cfg",
		ip:        ip,
	}
}

// GetLoadBalancer is an implementation of LoadBalancer.GetLoadBalancer
func (b *BareMetalLoadBalancer) GetLoadBalancer(clusterName string, service *api.Service) (status *api.LoadBalancerStatus, exists bool, retErr error) {
	name := cloudprovider.GetLoadBalancerName(service)
	glog.Infof("GetLoadBalancer [%s]", name)

	ports := service.Spec.Ports
	for _, port := range ports {
		LBName := fmt.Sprintf("%s.%s.%s.%d", clusterName, service.Namespace, service.Name, port.Port)
		frontend, ok := b.frontends[port.Port]
		if !ok {
			glog.Infof("Can't find lb by name [%s]", LBName)
			return &api.LoadBalancerStatus{}, false, nil
		}
		if frontend.Name != LBName {
			glog.Infof("[%s] Port used by another service: %s", LBName, frontend.Name)
			return &api.LoadBalancerStatus{}, false, nil
		}
	}

	return b.lbStatus(), true, nil
}

// EnsureLoadBalancer is an implementation of LoadBalancer.EnsureLoadBalancer.
func (b *BareMetalLoadBalancer) EnsureLoadBalancer(clusterName string, service *api.Service, nodes []*api.Node) (*api.LoadBalancerStatus, error) {
	ports := service.Spec.Ports
	for _, port := range ports {
		LBName := fmt.Sprintf("%s.%s.%s.%d", clusterName, service.Namespace, service.Name, port.Port)
		glog.Infof("EnsureLoadBalancer [%s]", LBName)
		if port.NodePort == 0 {
			glog.Warningf("Ignoring port without NodePort: %s", port)
			continue
		}
		frontend, ok := b.frontends[port.Port]
		if ok {
			if frontend.Name != LBName {
				return &api.LoadBalancerStatus{}, fmt.Errorf("Service port conflict. Port %d already reserved by %s", port.Port, frontend.Name)
			}
		} else {

			err := b.createLB(LBName, port, nodes)
			if err != nil {
				return &api.LoadBalancerStatus{}, fmt.Errorf("Error when creating LoadBalancer: %s", err)
			}
		}

	}

	err := b.write()
	if err != nil {
		return &api.LoadBalancerStatus{}, fmt.Errorf("Error when creating LoadBalancer: %s", err)
	}

	return b.lbStatus(), nil
}

// UpdateLoadBalancer is an implementation of LoadBalancer.UpdateLoadBalancer.
func (b *BareMetalLoadBalancer) UpdateLoadBalancer(clusterName string, service *api.Service, nodes []*api.Node) error {
	ports := service.Spec.Ports
	for _, port := range ports {
		LBName := fmt.Sprintf("%s.%s.%s.%d", clusterName, service.Namespace, service.Name, port.Port)
		glog.Infof("EnsureLoadBalancerUpdated[%s]", LBName)
		if port.NodePort == 0 {
			glog.Warningf("Ignoring port without NodePort: %s", port)
			continue
		}
		frontend, ok := b.frontends[port.Port]
		if ok {
			if frontend.Name == LBName {
				// just delete and recreate
				delete(b.frontends, port.Port)
				_, ok := b.backends[LBName]
				if ok {
					delete(b.backends, LBName)
				}
				err := b.createLB(LBName, port, nodes)
				if err != nil {
					return fmt.Errorf("Couldn't create LB: %s", err)
				}
			}
		}
		if !ok {
			glog.Infof("Can't find lb by name [%s]", LBName)
			return fmt.Errorf("Couldn't find LB with name %s", LBName)
		}
	}
	err := b.write()

	return err
}

// EnsureLoadBalancerDeleted is an implementation of LoadBalancer.EnsureLoadBalancerDeleted.
func (b *BareMetalLoadBalancer) EnsureLoadBalancerDeleted(clusterName string, service *api.Service) error {
	ports := service.Spec.Ports
	for _, port := range ports {
		LBName := fmt.Sprintf("%s.%s.%s.%d", clusterName, service.Namespace, service.Name, port.Port)
		glog.Infof("EnsureLoadBalancerDeleted [%s]", LBName)
		if port.NodePort == 0 {
			glog.Warningf("Ignoring port without NodePort: %s", port)
			continue
		}
		frontend, ok := b.frontends[port.Port]
		if ok {
			if frontend.Name == LBName {
				delete(b.frontends, port.Port)
				_, ok := b.backends[LBName]
				if ok {
					delete(b.backends, LBName)
				}
			}
		}
	}
	err := b.write()

	return err
}

func (b *BareMetalLoadBalancer) createLB(LBName string, port api.ServicePort, nodes []*api.Node) error {
	frontend := &Frontend{LBName, port.Port, LBName, b.ip}
	var endpoints Endpoints
	for i, node := range nodes {
		address, err := nodeAddress(node)
		if err != nil {
			glog.Infof("Can't get node address: %s", err)
			return fmt.Errorf("Can't get node address: %s", err)
		}
		endpoint := Endpoint{fmt.Sprintf("%s%d", LBName, i), address, port.NodePort}
		endpoints = append(endpoints, &endpoint)
	}
	backend := Backend{LBName, endpoints}
	b.backends[LBName] = &backend
	b.frontends[port.Port] = frontend
	return nil
}

func (b *BareMetalLoadBalancer) lbStatus() *api.LoadBalancerStatus {
	return &api.LoadBalancerStatus{
		Ingress: []api.LoadBalancerIngress{api.LoadBalancerIngress{IP: b.ip}},
	}
}

func nodeAddress(node *api.Node) (string, error) {
	addressTypes := []api.NodeAddressType{
		api.NodeExternalIP,
		api.NodeInternalIP,
	}
	for _, addressType := range addressTypes {
		for _, address := range node.Status.Addresses {
			if address.Type == addressType {
				return address.Address, nil
			}
		}
	}

	return "", fmt.Errorf("Address for node %s not found", node.Name)
}

func (b *BareMetalLoadBalancer) write() (err error) {
	var w io.Writer
	w, err = os.Create("/haproxy.cfg")
	if err != nil {
		return err
	}
	var t *template.Template
	t, err = template.ParseFiles(b.template)
	if err != nil {
		return err
	}
	conf := make(map[string]interface{})
	conf["frontends"] = b.frontends
	conf["backends"] = b.backends
	err = t.Execute(w, conf)
	return err
}
