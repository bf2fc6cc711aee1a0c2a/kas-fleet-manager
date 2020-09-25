package ocm

import (
	sdkClient "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

//go:generate moq -out client_moq.go . Client
type Client interface {
	CreateCluster(cluster *v1.Cluster) (*v1.Cluster, error)
	GetClusterIngresses(clusterID string) (*v1.IngressesListResponse, error)
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

func (c *client) CreateCluster(cluster *clustersmgmtv1.Cluster) (*v1.Cluster, error) {
	clusterResource := c.ocmClient.ClustersMgmt().V1().Clusters()
	response, err := clusterResource.Add().Body(cluster).Send()
	if err != nil {
		return &clustersmgmtv1.Cluster{}, errors.New(errors.ErrorGeneral, err.Error())
	}
	createdCluster := response.Body()

	return createdCluster, nil
}

// GetClusterIngresses sends a GET request to ocm to retrieve the ingresses of an OSD cluster
func (c *client) GetClusterIngresses(clusterID string) (*v1.IngressesListResponse, error) {
	clusterIngresses := c.ocmClient.ClustersMgmt().V1().Clusters().Cluster(clusterID).Ingresses()
	ingressList, err := clusterIngresses.List().Send()
	if err != nil {
		return nil, err
	}

	return ingressList, nil
}
