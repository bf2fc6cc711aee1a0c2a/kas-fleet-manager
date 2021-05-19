package clusters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/go-errors/errors"
)

// IdentityProvider Information about an identity provider
type IdentityProvider struct {

}

// IdentityProviderList List of IdentityProvider
type IdentityProviderList struct {
	items []IdentityProvider
}

// ResourceSet A set of OpenShift/k8s resources
type ResourceSet struct {
	name string
	resources []interface{}
}

// ClusterRequest information about the cluster creation request
type ClusterRequest struct {
	// cloud provider requirement
	CloudProvider string
	// region of the cluster
	Region         string
	// if the cluster is multi-az
	MultiAZ        bool
	// AdditionalSpec Additional information that can be used by the cloud provider to use when creating a new OpenShift/k8s cluster
	AdditionalSpec api.JSON
}

// ClusterSpec Information about the openshift/k8s cluster
type ClusterSpec struct {
	// internal id of the cluster. Used when making requests to the provider
	InternalID             string
	// external id of the cluster. Some providers provide an additional id for external usage. If not provided, the InternalID will be used.
	ExternalID string
	// the status of the cluster
	Status api.ClusterStatus
	// additional information related to the cluster, can vary depending on the provider
	AdditionalInfo api.JSON
}

// ProviderFactory used to return an instance of Provider implementation
type ProviderFactory interface {
	GetProvider(providerType api.ClusterProviderType) (Provider, error)
}

type Provider interface {
	// Create using the information provided to request a new OpenShift/k8s cluster from the provider
	Create(request *ClusterRequest) (*ClusterSpec, error)
	// Get retrieve the details about the cluster
	Get(spec *ClusterSpec) (*ClusterSpec, error)
	// AddIdentityProvider add an identity provider to the cluster
	AddIdentityProvider(clusterSpec *ClusterSpec, identityProvider IdentityProvider) (*IdentityProvider, error)
	// ListIdentityProviders list all the identity providers for the cluster
	ListIdentityProviders(clusterSpec *ClusterSpec) (*IdentityProviderList, error)
	// ApplyResources apply openshift/k8s resources to the cluster
	ApplyResources(clusterSpec *ClusterSpec, resources ResourceSet) (*ResourceSet, *error)
	// DeleteResources delete openshift/k8s resources from the cluster
	DeleteResources(clusterSpec *ClusterSpec, resources ResourceSet) error
	// ScaleUp scale the cluster up with the number of additional nodes specified
	ScaleUp(clusterSpec *ClusterSpec, increment int) (*ClusterSpec, error)
	// ScaleDown scale the cluster down with the number of nodes specified
	ScaleDown(clusterSpec *ClusterSpec, decrement int) (*ClusterSpec, error)
	// SetComputeNodes set the number of desired compute nodes for the cluster
	SetComputeNodes(clusterSpec *ClusterSpec, numNodes int) (*ClusterSpec, error)
	// GetClusterDNS Get the dns of the cluster
	GetClusterDNS(clusterSpec *ClusterSpec) (string, error)
}

// DefaultProviderFactory the default implementation for ProviderFactory
type DefaultProviderFactory struct {
	ocmClient ocm.Client
	config config.ApplicationConfig

	ocmProvider *OCMProvider
	standaloneProvider *StandaloneProvider
}

func (d *DefaultProviderFactory) GetProvider(providerType api.ClusterProviderType) (Provider, error) {
	switch providerType {
	case api.ClusterProviderOCM:
		if d.ocmProvider == nil {
			cb := ocm.NewClusterBuilder(d.config.AWS, d.config.OSDClusterConfig)
			d.ocmProvider = newOCMProvider(d.ocmClient, cb)
		}
		return d.ocmProvider, nil
	case api.ClusterProviderStandalone:
		if d.standaloneProvider == nil {
			d.standaloneProvider = newStandaloneProvider()
		}
		return d.standaloneProvider, nil
	default:
		return nil, errors.Errorf("invalid provider type: %v", providerType)
	}
}