package baremetal

import (
	"io"
	"os"

	"k8s.io/kubernetes/pkg/cloudprovider"
	"k8s.io/kubernetes/pkg/controller"
)

const (
	providerName = "baremetal"
)

// BareMetalProvider implents LoadBalancer
type BareMetalProvider struct {
	ip string
}

// Initialize passes a Kubernetes clientBuilder interface to the cloud provider
func (b *BareMetalProvider) Initialize(clientBuilder controller.ControllerClientBuilder) {}

// ProviderName returns the cloud provider ID.
func (b *BareMetalProvider) ProviderName() string {
	return providerName
}

// ScrubDNS filters DNS settings for pods.
func (b *BareMetalProvider) ScrubDNS(nameservers, searches []string) (nsOut, srchOut []string) {
	return nameservers, searches
}

// LoadBalancer returns an implementation of LoadBalancer
func (b *BareMetalProvider) LoadBalancer() (cloudprovider.LoadBalancer, bool) {
	return NewBareMetalLoadBalancer(b.ip), true
}

// Zones not supported
func (b *BareMetalProvider) Zones() (cloudprovider.Zones, bool) {
	return nil, false
}

// Instances not supported
func (b *BareMetalProvider) Instances() (cloudprovider.Instances, bool) {
	return nil, false
}

// Clusters not supported
func (b *BareMetalProvider) Clusters() (cloudprovider.Clusters, bool) {
	return nil, false
}

// Routes not supported
func (b *BareMetalProvider) Routes() (cloudprovider.Routes, bool) {
	return nil, false
}

func init() {
	cloudprovider.RegisterCloudProvider(providerName, func(config io.Reader) (cloudprovider.Interface, error) {
		ip := os.Getenv("LB_IP")
		return &BareMetalProvider{ip: ip}, nil
	})
}
