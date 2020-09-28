package ocm

import (
	sdkClient "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

//go:generate moq -out client_moq.go . Client
type Client interface {
	CreateCluster(cluster *clustersmgmtv1.Cluster) (*clustersmgmtv1.Cluster, error)
	GetClusterIngresses(clusterID string) (*clustersmgmtv1.IngressesListResponse, error)
	GetClusterStatus(id string) (*clustersmgmtv1.ClusterStatus, error)
	GetManagedKafkaAddon(id string) (*clustersmgmtv1.AddOnInstallation, error)
	CreateManagedKafkaAddon(id string) (*clustersmgmtv1.AddOnInstallation, error)
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

func (c client) GetClusterStatus(id string) (*clustersmgmtv1.ClusterStatus, error) {
	resp, err := c.ocmClient.ClustersMgmt().V1().Clusters().Cluster(id).Status().Get().Send()
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c client) CreateManagedKafkaAddon(id string) (*clustersmgmtv1.AddOnInstallation, error) {
	addon := clustersmgmtv1.NewAddOn().ID(api.ManagedKafkaAddonID)
	addonInstallation, err := clustersmgmtv1.NewAddOnInstallation().Addon(addon).Build()
	if err != nil {
		return nil, err
	}

	resp, err := c.ocmClient.ClustersMgmt().V1().Clusters().Cluster(id).Addons().Add().Body(addonInstallation).Send()
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c client) GetManagedKafkaAddon(id string) (*clustersmgmtv1.AddOnInstallation, error) {
	resp, err := c.ocmClient.ClustersMgmt().V1().Clusters().Cluster(id).Addons().List().Send()
	if err != nil {
		return nil, err
	}

	managedKafkaAddon := &clustersmgmtv1.AddOnInstallation{}
	resp.Items().Each(func(addOnInstallation *clustersmgmtv1.AddOnInstallation) bool {
		if addOnInstallation.ID() == api.ManagedKafkaAddonID {
			managedKafkaAddon = addOnInstallation
			return false
		}
		return true
	})

	return managedKafkaAddon, nil
}
