package services

import (
	"errors"
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"

	"github.com/jinzhu/gorm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm/converters"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/getsentry/sentry-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

//go:generate moq -out clusterservice_moq.go . ClusterService
type ClusterService interface {
	Create(cluster *api.Cluster) (*clustersmgmtv1.Cluster, *apiErrors.ServiceError)
	GetClusterDNS(clusterID string) (string, *apiErrors.ServiceError)
	ListByStatus(state api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError)
	UpdateStatus(cluster api.Cluster, status api.ClusterStatus) error
	FindCluster(criteria FindClusterCriteria) (*api.Cluster, *apiErrors.ServiceError)
	// FindClusterByID returns the cluster corresponding to the provided clusterID.
	// If the cluster has not been found nil is returned. If there has been an issue
	// finding the cluster an error is set
	FindClusterByID(clusterID string) (*api.Cluster, *apiErrors.ServiceError)
	ScaleUpComputeNodes(clusterID string, increment int) (*clustersmgmtv1.Cluster, *apiErrors.ServiceError)
	ScaleDownComputeNodes(clusterID string, decrement int) (*clustersmgmtv1.Cluster, *apiErrors.ServiceError)
	SetComputeNodes(clusterID string, numNodes int) (*clustersmgmtv1.Cluster, *apiErrors.ServiceError)
	ListGroupByProviderAndRegion(providers []string, regions []string, status []string) ([]*ResGroupCPRegion, *apiErrors.ServiceError)
	RegisterClusterJob(clusterRequest *api.Cluster) *apiErrors.ServiceError
	AddIdentityProviderID(clusterId string, identityProviderId string) *apiErrors.ServiceError
	DeleteByClusterID(clusterID string) *apiErrors.ServiceError
	// FindNonEmptyClusterById returns a cluster if it present and it is not empty.
	// Cluster emptiness is determined by checking whether the cluster contains Kafkas that have been provisioned, are being provisioned on it, or are being deprovisioned from it i.e kafka that are not in failure state.
	FindNonEmptyClusterById(clusterId string) (*api.Cluster, *apiErrors.ServiceError)
}

type clusterService struct {
	connectionFactory *db.ConnectionFactory
	ocmClient         ocm.Client
	awsConfig         *config.AWSConfig
	clusterBuilder    ocm.ClusterBuilder
}

// NewClusterService creates a new client for the OSD Cluster Service
func NewClusterService(connectionFactory *db.ConnectionFactory, ocmClient ocm.Client, awsConfig *config.AWSConfig, osdClusterConfig *config.OSDClusterConfig) ClusterService {
	return &clusterService{
		connectionFactory: connectionFactory,
		ocmClient:         ocmClient,
		awsConfig:         awsConfig,
		clusterBuilder:    ocm.NewClusterBuilder(awsConfig, osdClusterConfig),
	}
}

// RegisterClusterJob registers a new job in the cluster table
func (c clusterService) RegisterClusterJob(clusterRequest *api.Cluster) *apiErrors.ServiceError {
	dbConn := c.connectionFactory.New()
	clusterRequest.Status = api.ClusterAccepted
	if err := dbConn.Save(clusterRequest).Error; err != nil {
		return apiErrors.GeneralError("failed to create cluster job: %v", err)
	}
	return nil
}

// Create creates a new OSD cluster
//
// Returns the newly created cluster object
func (c clusterService) Create(cluster *api.Cluster) (*clustersmgmtv1.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	// Build a new OSD cluster object
	newCluster, err := c.clusterBuilder.NewOCMClusterFromCluster(cluster)
	if err != nil {
		return nil, apiErrors.New(apiErrors.ErrorGeneral, err.Error())
	}

	// Send POST request to /api/clusters_mgmt/v1/clusters to create a new OSD cluster
	createdCluster, err := c.ocmClient.CreateCluster(newCluster)
	if err != nil {
		sentry.CaptureException(err)
		return createdCluster, apiErrors.New(apiErrors.ErrorGeneral, err.Error())
	}

	// convert the cluster to the cluster type this service understands before saving
	convertedCluster := converters.ConvertCluster(createdCluster)

	// if the passed in cluster object has an ID, it means we need to overwrite it
	// because it was a cluster provisioning request (cluster_accepted status)
	if cluster.ID != "" {
		convertedCluster.ID = cluster.ID
	}

	if err := dbConn.Save(convertedCluster).Error; err != nil {
		return &clustersmgmtv1.Cluster{}, apiErrors.New(apiErrors.ErrorGeneral, err.Error())
	}

	return createdCluster, nil
}

// GetClusterDNS gets an OSD clusters DNS from OCM cluster service by ID
//
// Returns the DNS name
func (c clusterService) GetClusterDNS(clusterID string) (string, *apiErrors.ServiceError) {
	clusterDNS, err := c.ocmClient.GetClusterDNS(clusterID)
	if err != nil {
		return "", apiErrors.New(apiErrors.ErrorGeneral, err.Error())
	}
	return clusterDNS, nil
}

func (c clusterService) ListByStatus(status api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError) {
	if status.String() == "" {
		return nil, apiErrors.Validation("status is undefined")
	}
	dbConn := c.connectionFactory.New()

	var clusters []api.Cluster

	if err := dbConn.Model(&api.Cluster{}).Where("status = ?", status).Scan(&clusters).Error; err != nil {
		return nil, apiErrors.GeneralError("failed to query by status: %s", err.Error())
	}

	return clusters, nil
}

func (c clusterService) UpdateStatus(cluster api.Cluster, status api.ClusterStatus) error {

	if status.String() == "" {
		return apiErrors.Validation("status is undefined")
	}
	if cluster.ID == "" && cluster.ClusterID == "" {
		return apiErrors.Validation("id is undefined")
	}

	if status == api.ClusterReady || status == api.ClusterFailed {
		metrics.IncreaseClusterTotalOperationsCountMetric(constants.ClusterOperationCreate)
	}

	dbConn := c.connectionFactory.New()

	if cluster.ID != "" {
		if err := dbConn.Model(&api.Cluster{Meta: api.Meta{ID: cluster.ID}}).Update("status", status).Error; err != nil {
			return apiErrors.GeneralError("failed to update status: %s", err.Error())
		}
	} else {
		if err := dbConn.Model(&api.Cluster{ClusterID: cluster.ClusterID}).Update("status", status).Error; err != nil {
			return apiErrors.GeneralError("failed to update status: %s", err.Error())
		}
	}

	if status == api.ClusterReady {
		metrics.IncreaseClusterSuccessOperationsCountMetric(constants.ClusterOperationCreate)
	}

	return nil
}

type ResGroupCPRegion struct {
	Provider string
	Region   string
	Count    int
}

// ListGroupByProviderAndRegion retrieves existing OSD cluster with specified status in all providers and regions
func (c clusterService) ListGroupByProviderAndRegion(providers []string, regions []string, status []string) ([]*ResGroupCPRegion, *apiErrors.ServiceError) {
	if len(providers) == 0 || len(regions) == 0 || len(status) == 0 {
		return nil, apiErrors.Validation("provider, region and status must not be empty")
	}
	dbConn := c.connectionFactory.New()
	var grpResult []*ResGroupCPRegion

	//only one record returns for each region if they exist
	if err := dbConn.Table("clusters").
		Select("cloud_provider as Provider, region as Region, count(1) as Count").
		Where("cloud_provider in (?)", providers).
		Where("region in (?)", regions).
		Where("status in (?) ", status).
		Group("cloud_provider, region").Scan(&grpResult).Error; err != nil {
		return nil, apiErrors.GeneralError("failed to query by cluster info.: %s", err.Error())
	}

	return grpResult, nil
}

type FindClusterCriteria struct {
	Provider string
	Region   string
	MultiAZ  bool
	Status   api.ClusterStatus
}

func (c clusterService) FindCluster(criteria FindClusterCriteria) (*api.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	var cluster api.Cluster

	clusterDetails := &api.Cluster{
		CloudProvider: criteria.Provider,
		Region:        criteria.Region,
		MultiAZ:       criteria.MultiAZ,
		Status:        criteria.Status,
	}

	if err := dbConn.Where(clusterDetails).First(&cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apiErrors.GeneralError("failed to find cluster with criteria: %s", err.Error())
	}

	return &cluster, nil
}

func (c clusterService) FindClusterByID(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
	if clusterID == "" {
		return nil, apiErrors.Validation("clusterID is undefined")
	}
	dbConn := c.connectionFactory.New()

	var cluster *api.Cluster = &api.Cluster{}

	clusterDetails := &api.Cluster{
		ClusterID: clusterID,
	}

	if err := dbConn.Where(clusterDetails).First(cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apiErrors.GeneralError("failed to find cluster with id: %s %s", clusterID, err.Error())
	}

	return cluster, nil
}

// ScaleUpComputeNodes adds three additional compute nodes to cluster specified by clusterID
func (c clusterService) ScaleUpComputeNodes(clusterID string, increment int) (*clustersmgmtv1.Cluster, *apiErrors.ServiceError) {
	if clusterID == "" {
		return nil, apiErrors.Validation("clusterID is undefined")
	}

	// scale up compute nodes
	cluster, err := c.ocmClient.ScaleUpComputeNodes(clusterID, increment)
	if err != nil {
		return nil, apiErrors.New(apiErrors.ErrorGeneral, err.Error())
	}
	return cluster, nil
}

// ScaleDownComputeNodes removes three compute nodes to cluster specified by clusterID
func (c clusterService) ScaleDownComputeNodes(clusterID string, decrement int) (*clustersmgmtv1.Cluster, *apiErrors.ServiceError) {
	if clusterID == "" {
		return nil, apiErrors.Validation("clusterID is undefined")
	}

	// scale up compute nodes
	cluster, err := c.ocmClient.ScaleDownComputeNodes(clusterID, decrement)
	if err != nil {
		return nil, apiErrors.New(apiErrors.ErrorGeneral, err.Error())
	}
	return cluster, nil
}

func (c clusterService) SetComputeNodes(clusterID string, numNodes int) (*clustersmgmtv1.Cluster, *apiErrors.ServiceError) {
	if clusterID == "" {
		return nil, apiErrors.Validation("clusterID is undefined")
	}

	// set number of compute nodes
	cluster, err := c.ocmClient.SetComputeNodes(clusterID, numNodes)
	if err != nil {
		return nil, apiErrors.New(apiErrors.ErrorGeneral, err.Error())
	}
	return cluster, nil
}

func (c clusterService) AddIdentityProviderID(id string, identityProviderId string) *apiErrors.ServiceError {
	dbConn := c.connectionFactory.New()
	if err := dbConn.Model(&api.Cluster{Meta: api.Meta{ID: id}}).Update("identity_provider_id", identityProviderId).Error; err != nil {
		return apiErrors.GeneralError("failed to update identity_provider_id for cluster %s: %s", id, err.Error())
	}

	return nil
}

func (c clusterService) DeleteByClusterID(clusterID string) *apiErrors.ServiceError {
	dbConn := c.connectionFactory.New()
	if err := dbConn.Delete(&api.Cluster{}, api.Cluster{ClusterID: clusterID}).Error; err != nil {
		return apiErrors.GeneralError("Unable to delete cluster with cluster_id %s: %s", clusterID, err)
	}

	return nil
}

func (c clusterService) FindNonEmptyClusterById(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	var cluster *api.Cluster = &api.Cluster{}

	clusterDetails := &api.Cluster{
		ClusterID: clusterID,
	}

	//subQuery := dbConn.Select("cluster_id").Where("status != '?' AND cluster_id != ''", constants.KafkaRequestStatusFailed).Table("kafka_requests")
	subQuery := `SELECT "kafka_requests"."cluster_id" FROM kafka_requests WHERE "kafka_requests"."status" != 'failed' AND "kafka_requests"."deleted_at" IS NULL AND "kafka_requests"."cluster_id" != ''`
	if err := dbConn.Where(clusterDetails).Where(fmt.Sprintf("cluster_id IN (%s)", subQuery)).First(cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apiErrors.GeneralError("failed to find cluster with id: %s %s", clusterID, err.Error())
	}

	return cluster, nil
}
