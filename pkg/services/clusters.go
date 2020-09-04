package services

import (
	"context"
	"fmt"

	"github.com/rs/xid"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"

	sdkClient "github.com/openshift-online/ocm-sdk-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	clusterNamePrefix = "ms-"
)

type ClusterService interface {
	Create(ctx context.Context, cluster *api.Cluster) (*clustersmgmtv1.Cluster, *errors.ServiceError)
}

type clusterService struct {
	ocmClient *sdkClient.Connection
	awsConfig *config.AWSConfig
}

// NewClusterService creates a new client for the OSD Cluster Service
func NewClusterService(ocmClient *sdkClient.Connection, awsConfig *config.AWSConfig) *clusterService {
	return &clusterService{
		ocmClient: ocmClient,
		awsConfig: awsConfig,
	}
}

// Create creates a new OSD cluster
//
// Returns the newly created cluster object
func (c clusterService) Create(cluster *api.Cluster) (*clustersmgmtv1.Cluster, *errors.ServiceError) {
	// Build a new OSD cluster object
	newCluster, err := c.buildNewClusterObject(cluster)
	if err != nil {
		return &clustersmgmtv1.Cluster{}, errors.New(errors.ErrorGeneral, err.Error())
	}

	// Send POST request to /api/clusters_mgmt/v1/clusters to create a new OSD cluster
	clusterResource := c.ocmClient.ClustersMgmt().V1().Clusters()
	response, err := clusterResource.Add().Body(newCluster).Send()
	if err != nil {
		return &clustersmgmtv1.Cluster{}, errors.New(errors.ErrorGeneral, err.Error())
	}

	createdCluster := response.Body()
	// TODO: Store cluster info in DB

	return createdCluster, nil
}

// buildNewClusterObject creates a new Cluster object based on the cluster configuration passed
func (c clusterService) buildNewClusterObject(cluster *api.Cluster) (*clustersmgmtv1.Cluster, error) {
	clusterBuilder := clustersmgmtv1.NewCluster()
	clusterBuilder.Name(fmt.Sprintf("%s-%s", clusterNamePrefix, xid.New().String()))
	clusterBuilder.CloudProvider(clustersmgmtv1.NewCloudProvider().ID(cluster.CloudProvider))
	clusterBuilder.Region(clustersmgmtv1.NewCloudRegion().ID(cluster.Region))
	// clusterBuilder.MultiAZ(cluster.MultiAZ) // Currently disabled as we do not have quota for this type of cluster.

	// Setting BYOC to always be true for now as this is the only available cluster type within our quota.
	clusterBuilder.BYOC(true)
	clusterBuilder.Managed(true)

	// AWS config read from the secrets/aws.* files
	awsBuilder := clustersmgmtv1.NewAWS().AccountID(c.awsConfig.AccountID).AccessKeyID(c.awsConfig.AccessKey).SecretAccessKey(c.awsConfig.SecretAccessKey)
	clusterBuilder.AWS(awsBuilder)

	return clusterBuilder.Build()
}
