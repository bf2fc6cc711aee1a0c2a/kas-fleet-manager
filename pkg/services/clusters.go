package services

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm/converters"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"

	sdkClient "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

//go:generate moq -out clusterservice_moq.go . ClusterService
type ClusterService interface {
	Create(cluster *api.Cluster) (*clustersmgmtv1.Cluster, *errors.ServiceError)
	GetClusterDNS(clusterID string) (string, *errors.ServiceError)
}

type clusterService struct {
	connectionFactory *db.ConnectionFactory
	ocmClient         *sdkClient.Connection
	awsConfig         *config.AWSConfig
	clusterBuilder    ocm.ClusterBuilder
}

// NewClusterService creates a new client for the OSD Cluster Service
func NewClusterService(connectionFactory *db.ConnectionFactory, ocmClient *sdkClient.Connection, awsConfig *config.AWSConfig) ClusterService {
	return &clusterService{
		connectionFactory: connectionFactory,
		ocmClient:         ocmClient,
		awsConfig:         awsConfig,
		clusterBuilder:    ocm.NewClusterBuilder(awsConfig),
	}
}

// Create creates a new OSD cluster
//
// Returns the newly created cluster object
func (c clusterService) Create(cluster *api.Cluster) (*clustersmgmtv1.Cluster, *errors.ServiceError) {
	dbConn := c.connectionFactory.New()

	// Build a new OSD cluster object
	newCluster, err := c.clusterBuilder.NewOCMClusterFromCluster(cluster)
	if err != nil {
		return &clustersmgmtv1.Cluster{}, errors.New(errors.ErrorGeneral, err.Error())
	}

	// Send POST request to /api/clusters_mgmt/v1/clusters to create a new OSD cluster
	clusterResource := c.ocmClient.ClustersMgmt().V1().Clusters()
	response, err := clusterResource.Add().Body(newCluster).Send()
	if err != nil {
		return &clustersmgmtv1.Cluster{}, errors.New(errors.ErrorGeneral, err.Error())
	}

	// persist cluster information to database
	createdCluster := response.Body()

	// convert the cluster to the cluster type this service understands before saving
	if err := dbConn.Save(converters.ConvertCluster(createdCluster)).Error; err != nil {
		return &clustersmgmtv1.Cluster{}, errors.New(errors.ErrorGeneral, err.Error())
	}

	return createdCluster, nil
}

// GetClusterDNS gets an OSD clusters DNS from OCM cluster service by ID
//
// Returns the DNS name
func (c clusterService) GetClusterDNS(clusterID string) (string, *errors.ServiceError) {
	// Send GET request to /api/clusters_mgmt/v1/clusters/{clusterID} to retrieve an OSD cluster
	clusterIngresses := c.ocmClient.ClustersMgmt().V1().Clusters().Cluster(clusterID).Ingresses()
	response, err := clusterIngresses.List().Send()
	if err != nil {
		return "", errors.New(errors.ErrorGeneral, err.Error())
	}

	var clusterDNS string

	response.Items().Each(func(ingress *clustersmgmtv1.Ingress) bool {
		if ingress.Default() == true {
			clusterDNS = ingress.DNSName()
			return false
		}
		return true
	})

	return clusterDNS, nil
}
