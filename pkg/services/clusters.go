package services

import (
	"errors"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/metrics"

	"github.com/jinzhu/gorm"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm/converters"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	ocmErrors "gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"

	"github.com/getsentry/sentry-go"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	apiErrors "gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

//go:generate moq -out clusterservice_moq.go . ClusterService
type ClusterService interface {
	Create(cluster *api.Cluster) (*clustersmgmtv1.Cluster, *ocmErrors.ServiceError)
	GetClusterDNS(clusterID string) (string, *ocmErrors.ServiceError)
	ListByStatus(state api.ClusterStatus) ([]api.Cluster, *ocmErrors.ServiceError)
	UpdateStatus(cluster api.Cluster, status api.ClusterStatus) error
	FindCluster(criteria FindClusterCriteria) (*api.Cluster, *ocmErrors.ServiceError)
	FindClusterByID(clusterID string) (api.Cluster, *ocmErrors.ServiceError)
	ScaleUpComputeNodes(clusterID string) (*clustersmgmtv1.Cluster, *ocmErrors.ServiceError)
	ScaleDownComputeNodes(clusterID string) (*clustersmgmtv1.Cluster, *ocmErrors.ServiceError)
	ListGroupByProviderAndRegion(providers []string, regions []string, status []string) ([]*ResGroupCPRegion, *ocmErrors.ServiceError)
	RegisterClusterJob(clusterRequest *api.Cluster) *apiErrors.ServiceError
}

type clusterService struct {
	connectionFactory *db.ConnectionFactory
	ocmClient         ocm.Client
	awsConfig         *config.AWSConfig
	clusterBuilder    ocm.ClusterBuilder
}

// NewClusterService creates a new client for the OSD Cluster Service
func NewClusterService(connectionFactory *db.ConnectionFactory, ocmClient ocm.Client, awsConfig *config.AWSConfig) ClusterService {
	return &clusterService{
		connectionFactory: connectionFactory,
		ocmClient:         ocmClient,
		awsConfig:         awsConfig,
		clusterBuilder:    ocm.NewClusterBuilder(awsConfig),
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
func (c clusterService) Create(cluster *api.Cluster) (*clustersmgmtv1.Cluster, *ocmErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	// Build a new OSD cluster object
	newCluster, err := c.clusterBuilder.NewOCMClusterFromCluster(cluster)
	if err != nil {
		return nil, ocmErrors.New(ocmErrors.ErrorGeneral, err.Error())
	}

	// Send POST request to /api/clusters_mgmt/v1/clusters to create a new OSD cluster
	createdCluster, err := c.ocmClient.CreateCluster(newCluster)
	if err != nil {
		sentry.CaptureException(err)
		return createdCluster, ocmErrors.New(ocmErrors.ErrorGeneral, err.Error())
	}

	// convert the cluster to the cluster type this service understands before saving
	convertedCluster := converters.ConvertCluster(createdCluster)

	// if the passed in cluster object has an ID, it means we need to overwrite it
	// because it was a cluster provisioning request (cluster_accepted status)
	if cluster.ID != "" {
		convertedCluster.ID = cluster.ID
	}

	if err := dbConn.Save(convertedCluster).Error; err != nil {
		return &clustersmgmtv1.Cluster{}, ocmErrors.New(ocmErrors.ErrorGeneral, err.Error())
	}

	return createdCluster, nil
}

// GetClusterDNS gets an OSD clusters DNS from OCM cluster service by ID
//
// Returns the DNS name
func (c clusterService) GetClusterDNS(clusterID string) (string, *ocmErrors.ServiceError) {
	clusterDNS, err := c.ocmClient.GetClusterDNS(clusterID)
	if err != nil {
		return "", ocmErrors.New(ocmErrors.ErrorGeneral, err.Error())
	}
	return clusterDNS, nil
}

func (c clusterService) ListByStatus(status api.ClusterStatus) ([]api.Cluster, *ocmErrors.ServiceError) {
	if status.String() == "" {
		return nil, ocmErrors.Validation("status is undefined")
	}
	dbConn := c.connectionFactory.New()

	var clusters []api.Cluster

	if err := dbConn.Model(&api.Cluster{}).Where("status = ?", status).Scan(&clusters).Error; err != nil {
		return nil, ocmErrors.GeneralError("failed to query by status: %s", err.Error())
	}

	return clusters, nil
}

func (c clusterService) UpdateStatus(cluster api.Cluster, status api.ClusterStatus) error {

	if status.String() == "" {
		return ocmErrors.Validation("status is undefined")
	}
	if cluster.ID == "" && cluster.ClusterID == "" {
		return ocmErrors.Validation("id is undefined")
	}

	if status == api.ClusterReady || status == api.ClusterFailed {
		metrics.IncreaseClusterTotalOperationsCountMetric(constants.ClusterOperationCreate)
	}

	dbConn := c.connectionFactory.New()

	if cluster.ID != "" {
		if err := dbConn.Model(&api.Cluster{Meta: api.Meta{ID: cluster.ID}}).Update("status", status).Error; err != nil {
			return ocmErrors.GeneralError("failed to update status: %s", err.Error())
		}
	} else {
		if err := dbConn.Model(&api.Cluster{ClusterID: cluster.ClusterID}).Update("status", status).Error; err != nil {
			return ocmErrors.GeneralError("failed to update status: %s", err.Error())
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
func (c clusterService) ListGroupByProviderAndRegion(providers []string, regions []string, status []string) ([]*ResGroupCPRegion, *ocmErrors.ServiceError) {
	if len(providers) == 0 || len(regions) == 0 || len(status) == 0 {
		return nil, ocmErrors.Validation("provider, region and status must not be empty")
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
		return nil, ocmErrors.GeneralError("failed to query by cluster info.: %s", err.Error())
	}

	return grpResult, nil
}

type FindClusterCriteria struct {
	Provider string
	Region   string
	MultiAZ  bool
	Status   api.ClusterStatus
}

func (c clusterService) FindCluster(criteria FindClusterCriteria) (*api.Cluster, *ocmErrors.ServiceError) {
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
		return nil, ocmErrors.GeneralError("failed to find cluster with criteria: %s", err.Error())
	}

	return &cluster, nil
}

func (c clusterService) FindClusterByID(clusterID string) (api.Cluster, *ocmErrors.ServiceError) {
	if clusterID == "" {
		return api.Cluster{}, ocmErrors.Validation("clusterID is undefined")
	}
	dbConn := c.connectionFactory.New()

	var cluster api.Cluster

	clusterDetails := &api.Cluster{
		ClusterID: clusterID,
	}

	if err := dbConn.Where(clusterDetails).First(&cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return api.Cluster{}, nil
		}
		return api.Cluster{}, ocmErrors.GeneralError("failed to find cluster with id: %s %s", clusterID, err.Error())
	}

	return cluster, nil
}

// ScaleUpComputeNodes adds three additional compute nodes to cluster specified by clusterID
func (c clusterService) ScaleUpComputeNodes(clusterID string) (*clustersmgmtv1.Cluster, *ocmErrors.ServiceError) {
	if clusterID == "" {
		return nil, ocmErrors.Validation("clusterID is undefined")
	}

	// scale up compute nodes
	cluster, err := c.ocmClient.ScaleUpComputeNodes(clusterID)
	if err != nil {
		return nil, ocmErrors.New(ocmErrors.ErrorGeneral, err.Error())
	}
	return cluster, nil
}

// ScaleDownComputeNodes removes three compute nodes to cluster specified by clusterID
func (c clusterService) ScaleDownComputeNodes(clusterID string) (*clustersmgmtv1.Cluster, *ocmErrors.ServiceError) {
	if clusterID == "" {
		return nil, ocmErrors.Validation("clusterID is undefined")
	}

	// scale up compute nodes
	cluster, err := c.ocmClient.ScaleDownComputeNodes(clusterID)
	if err != nil {
		return nil, ocmErrors.New(ocmErrors.ErrorGeneral, err.Error())
	}
	return cluster, nil
}
