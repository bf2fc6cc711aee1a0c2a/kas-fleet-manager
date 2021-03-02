package ocm

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	sdkClient "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

// Specify the parameters for an addon
// Value will be string here and OCM will convert it to the right type
type AddonParameter struct {
	Id    string
	Value string
}

//go:generate moq -out client_moq.go . Client
type Client interface {
	CreateCluster(cluster *clustersmgmtv1.Cluster) (*clustersmgmtv1.Cluster, error)
	GetClusterIngresses(clusterID string) (*clustersmgmtv1.IngressesListResponse, error)
	GetCluster(clusterID string) (*clustersmgmtv1.Cluster, error)
	GetClusterStatus(id string) (*clustersmgmtv1.ClusterStatus, error)
	GetCloudProviders() (*clustersmgmtv1.CloudProviderList, error)
	GetRegions(provider *clustersmgmtv1.CloudProvider) (*clustersmgmtv1.CloudRegionList, error)
	GetAddon(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error)
	CreateAddonWithParams(clusterId string, addonId string, parameters []AddonParameter) (*clustersmgmtv1.AddOnInstallation, error)
	CreateAddon(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error)
	GetClusterDNS(clusterID string) (string, error)
	CreateSyncSet(clusterID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error)
	UpdateSyncSet(clusterID string, syncSetID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error)
	GetSyncSet(clusterID string, syncSetID string) (*clustersmgmtv1.Syncset, error)
	DeleteSyncSet(clusterID string, syncsetID string) (int, error)
	ScaleUpComputeNodes(clusterID string, increment int) (*clustersmgmtv1.Cluster, error)
	ScaleDownComputeNodes(clusterID string, decrement int) (*clustersmgmtv1.Cluster, error)
	SetComputeNodes(clusterID string, numNodes int) (*clustersmgmtv1.Cluster, error)
}

var _ Client = &client{}

type client struct {
	ocmClient *sdkClient.Connection
}

func NewClient(ocmClient *sdkClient.Connection) Client {
	return &client{
		ocmClient: ocmClient,
	}
}

func (c *client) CreateCluster(cluster *clustersmgmtv1.Cluster) (*clustersmgmtv1.Cluster, error) {
	clusterResource := c.ocmClient.ClustersMgmt().V1().Clusters()
	response, err := clusterResource.Add().Body(cluster).Send()
	if err != nil {
		return &clustersmgmtv1.Cluster{}, errors.New(errors.ErrorGeneral, err.Error())
	}
	createdCluster := response.Body()

	return createdCluster, nil
}

// GetClusterIngresses sends a GET request to ocm to retrieve the ingresses of an OSD cluster
func (c *client) GetClusterIngresses(clusterID string) (*clustersmgmtv1.IngressesListResponse, error) {
	clusterIngresses := c.ocmClient.ClustersMgmt().V1().Clusters().Cluster(clusterID).Ingresses()
	ingressList, err := clusterIngresses.List().Send()
	if err != nil {
		return nil, err
	}

	return ingressList, nil
}

func (c client) GetCluster(clusterID string) (*clustersmgmtv1.Cluster, error) {
	resp, err := c.ocmClient.ClustersMgmt().V1().Clusters().Cluster(clusterID).Get().Send()
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c client) GetClusterStatus(id string) (*clustersmgmtv1.ClusterStatus, error) {
	resp, err := c.ocmClient.ClustersMgmt().V1().Clusters().Cluster(id).Status().Get().Send()
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c *client) GetCloudProviders() (*clustersmgmtv1.CloudProviderList, error) {
	providersCollection := c.ocmClient.ClustersMgmt().V1().CloudProviders()
	providersResponse, err := providersCollection.List().Send()
	if err != nil {
		return nil, err
	}
	cloudProviderList := providersResponse.Items()
	return cloudProviderList, nil
}

func (c *client) GetRegions(provider *clustersmgmtv1.CloudProvider) (*clustersmgmtv1.CloudRegionList, error) {
	regionsCollection := c.ocmClient.ClustersMgmt().V1().CloudProviders().CloudProvider(provider.ID()).Regions()
	regionsResponse, err := regionsCollection.List().Send()
	if err != nil {
		return nil, err
	}

	regionList := regionsResponse.Items()
	return regionList, nil
}

func (c client) CreateAddonWithParams(clusterId string, addonId string, params []AddonParameter) (*clustersmgmtv1.AddOnInstallation, error) {
	addon := clustersmgmtv1.NewAddOn().ID(addonId)
	addonParameters := newAddonParameterListBuilder(params)
	addonInstallationBuilder := clustersmgmtv1.NewAddOnInstallation().Addon(addon)
	if addonParameters != nil {
		addonInstallationBuilder = addonInstallationBuilder.Parameters(addonParameters)
	}
	addonInstallation, err := addonInstallationBuilder.Build()
	if err != nil {
		return nil, err
	}
	resp, err := c.ocmClient.ClustersMgmt().V1().Clusters().Cluster(clusterId).Addons().Add().Body(addonInstallation).Send()
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c client) CreateAddon(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
	return c.CreateAddonWithParams(clusterId, addonId, []AddonParameter{})
}

func (c client) GetAddon(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
	resp, err := c.ocmClient.ClustersMgmt().V1().Clusters().Cluster(clusterId).Addons().List().Send()
	if err != nil {
		return nil, err
	}

	addon := &clustersmgmtv1.AddOnInstallation{}
	resp.Items().Each(func(addOnInstallation *clustersmgmtv1.AddOnInstallation) bool {
		if addOnInstallation.ID() == addonId {
			addon = addOnInstallation
			return false
		}
		return true
	})

	return addon, nil
}

func (c *client) GetClusterDNS(clusterID string) (string, error) {
	if clusterID == "" {
		return "", errors.New(errors.ErrorGeneral, "ClusterID cannot be empty")
	}
	ingresses, err := c.GetClusterIngresses(clusterID)
	if err != nil {
		return "", errors.New(errors.ErrorGeneral, err.Error())
	}

	var clusterDNS string
	ingresses.Items().Each(func(ingress *clustersmgmtv1.Ingress) bool {
		if ingress.Default() {
			clusterDNS = ingress.DNSName()
			return false
		}
		return true
	})
	return clusterDNS, nil
}

func (c client) CreateSyncSet(clusterID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
	clustersResource := c.ocmClient.ClustersMgmt().V1().Clusters()
	response, syncsetErr := clustersResource.Cluster(clusterID).
		ExternalConfiguration().
		Syncsets().
		Add().
		Body(syncset).
		Send()
	var err error
	if syncsetErr != nil {
		err = errors.NewErrorFromHTTPStatusCode(response.Status(), "ocm client failed to create syncset: %s", syncsetErr)
	}
	return response.Body(), err
}

func (c client) UpdateSyncSet(clusterID string, syncSetID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
	clustersResource := c.ocmClient.ClustersMgmt().V1().Clusters()
	response, syncsetErr := clustersResource.Cluster(clusterID).
		ExternalConfiguration().
		Syncsets().
		Syncset(syncSetID).
		Update().
		Body(syncset).
		Send()

	var err error
	if syncsetErr != nil {
		err = errors.NewErrorFromHTTPStatusCode(response.Status(), "ocm client failed to update syncset '%s': %s", syncSetID, syncsetErr)
	}
	return response.Body(), err
}

func (c client) GetSyncSet(clusterID string, syncSetID string) (*clustersmgmtv1.Syncset, error) {
	clustersResource := c.ocmClient.ClustersMgmt().V1().Clusters()
	response, syncsetErr := clustersResource.Cluster(clusterID).
		ExternalConfiguration().
		Syncsets().
		Syncset(syncSetID).
		Get().
		Send()

	var err error
	if syncsetErr != nil {
		err = errors.NewErrorFromHTTPStatusCode(response.Status(), "ocm client failed to get syncset '%s': %s", syncSetID, syncsetErr)
	}
	return response.Body(), err
}

// Status returns the response status code.
func (c client) DeleteSyncSet(clusterID string, syncsetID string) (int, error) {
	clustersResource := c.ocmClient.ClustersMgmt().V1().Clusters()
	response, syncsetErr := clustersResource.Cluster(clusterID).
		ExternalConfiguration().
		Syncsets().
		Syncset(syncsetID).
		Delete().
		Send()
	return response.Status(), syncsetErr
}

// ScaleUpComputeNodes scales up compute nodes by increment value
func (c client) ScaleUpComputeNodes(clusterID string, increment int) (*clustersmgmtv1.Cluster, error) {
	return c.scaleComputeNodes(clusterID, increment)
}

// ScaleDownComputeNodes scales down compute nodes by decrement value
func (c client) ScaleDownComputeNodes(clusterID string, decrement int) (*clustersmgmtv1.Cluster, error) {
	return c.scaleComputeNodes(clusterID, -decrement)
}

// scaleComputeNodes scales the Compute nodes up or down by the value of `numNodes`
func (c client) scaleComputeNodes(clusterID string, numNodes int) (*clustersmgmtv1.Cluster, error) {
	clusterClient := c.ocmClient.ClustersMgmt().V1().Clusters().Cluster(clusterID)

	cluster, err := clusterClient.Get().Send()
	if err != nil {
		return nil, err
	}

	// get current number of compute nodes
	currentNumOfNodes := cluster.Body().Nodes().Compute()

	// create a cluster object with updated number of compute nodes
	// NOTE - there is no need to handle whether the number of nodes is valid, as this is handled by OCM
	patch, err := clustersmgmtv1.NewCluster().Nodes(clustersmgmtv1.NewClusterNodes().Compute(currentNumOfNodes + numNodes)).
		Build()
	if err != nil {
		return nil, err
	}

	// patch cluster with updated number of compute nodes
	resp, err := clusterClient.Update().Body(patch).Send()
	if err != nil {
		return nil, err
	}

	return resp.Body(), nil
}

func (c client) SetComputeNodes(clusterID string, numNodes int) (*clustersmgmtv1.Cluster, error) {
	clusterClient := c.ocmClient.ClustersMgmt().V1().Clusters().Cluster(clusterID)

	patch, err := clustersmgmtv1.NewCluster().Nodes(clustersmgmtv1.NewClusterNodes().Compute(numNodes)).
		Build()
	if err != nil {
		return nil, err
	}

	// patch cluster with updated number of compute nodes
	resp, err := clusterClient.Update().Body(patch).Send()
	if err != nil {
		return nil, err
	}

	return resp.Body(), nil
}

func newAddonParameterListBuilder(params []AddonParameter) *clustersmgmtv1.AddOnInstallationParameterListBuilder {
	if len(params) > 0 {
		var items []*clustersmgmtv1.AddOnInstallationParameterBuilder
		for _, p := range params {
			pb := clustersmgmtv1.NewAddOnInstallationParameter().ID(p.Id).Value(p.Value)
			items = append(items, pb)
		}
		return clustersmgmtv1.NewAddOnInstallationParameterList().Items(items...)
	}
	return nil
}
