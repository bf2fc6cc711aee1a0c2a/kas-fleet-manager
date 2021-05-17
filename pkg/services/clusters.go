package services

import (
	"errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/golang/glog"

	"gorm.io/gorm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm/converters"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

//go:generate moq -out clusterservice_moq.go . ClusterService
type ClusterService interface {
	Create(cluster *api.Cluster) (*clustersmgmtv1.Cluster, *apiErrors.ServiceError)
	Update(cluster *api.Cluster) *apiErrors.ServiceError
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
	AddIdentityProviderID(clusterID string, identityProviderId string) *apiErrors.ServiceError
	DeleteByClusterID(clusterID string) *apiErrors.ServiceError
	// FindNonEmptyClusterById returns a cluster if it present and it is not empty.
	// Cluster emptiness is determined by checking whether the cluster contains Kafkas that have been provisioned, are being provisioned on it, or are being deprovisioned from it i.e kafka that are not in failure state.
	FindNonEmptyClusterById(clusterID string) (*api.Cluster, *apiErrors.ServiceError)
	// ListAllClusterIds returns all the valid cluster ids in array
	ListAllClusterIds() ([]api.Cluster, *apiErrors.ServiceError)
	// FindAllClusters return all the valid clusters in array
	FindAllClusters(criteria FindClusterCriteria) ([]*api.Cluster, *apiErrors.ServiceError)
	// FindKafkaInstanceCount returns the kafka instance counts associated with the list of clusters. If the list is empty, it will list all clusterIds that have Kafka instances assigned.
	FindKafkaInstanceCount(clusterIDs []string) ([]ResKafkaInstanceCount, *apiErrors.ServiceError)
	// UpdateMultiClusterStatus updates a list of clusters' status to a status
	UpdateMultiClusterStatus(clusterIds []string, status api.ClusterStatus) *apiErrors.ServiceError
	// CountByStatus returns the count of clusters for each given status in the database
	CountByStatus([]api.ClusterStatus) ([]ClusterStatusCount, error)
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
	if err := dbConn.Save(clusterRequest).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to register cluster job")
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
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "error building cluster")
	}

	// Send POST request to /api/clusters_mgmt/v1/clusters to create a new OSD cluster
	createdCluster, err := c.ocmClient.CreateCluster(newCluster)
	if err != nil {
		return createdCluster, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "error creating a Dataplane cluster")
	}

	// convert the cluster to the cluster type this service understands before saving
	convertedCluster := converters.ConvertCluster(createdCluster)

	// if the passed in cluster object has an ID, it means we need to overwrite it
	// because it was a cluster provisioning request (cluster_accepted status)
	if cluster.ID != "" {
		convertedCluster.ID = cluster.ID
	}

	if err := dbConn.Save(convertedCluster).Error; err != nil {
		return &clustersmgmtv1.Cluster{}, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update created cluster")
	}

	return createdCluster, nil
}

func (k *clusterService) Update(cluster *api.Cluster) *apiErrors.ServiceError {
	dbConn := k.connectionFactory.New()

	if err := dbConn.Model(cluster).Updates(cluster).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update cluster")
	}
	return nil
}

// GetClusterDNS gets an OSD clusters DNS from OCM cluster service by ID
//
// Returns the DNS name
func (c clusterService) GetClusterDNS(clusterID string) (string, *apiErrors.ServiceError) {
	cluster, serviceErr := c.FindClusterByID(clusterID)
	if serviceErr != nil {
		return "", serviceErr
	}

	if cluster != nil && cluster.ClusterDNS != "" {
		return cluster.ClusterDNS, nil
	}

	// If the clusterDNS is not present in the database, retrieve it from OCM
	clusterDNS, err := c.ocmClient.GetClusterDNS(clusterID)
	if err != nil {
		return "", apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get cluster DNS from OCM")
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
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to query by status")
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

	var query, arg string

	if cluster.ID != "" {
		query, arg = "id = ?", cluster.ID
	} else {
		query, arg = "cluster_id = ?", cluster.ClusterID
	}

	if err := dbConn.Model(&api.Cluster{}).Where(query, arg).Update("status", status).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update cluster status")
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
	if err := dbConn.Model(&api.Cluster{}).
		Select("cloud_provider as Provider, region as Region, count(1) as Count").
		Where("cloud_provider in (?)", providers).
		Where("region in (?)", regions).
		Where("status in (?) ", status).
		Group("cloud_provider, region").Scan(&grpResult).Error; err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to list by cloud provider, regions and status")
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

	// we order them by "created_at" field instead of the default "id" field.
	// They are mostly the same as the library we use (ksuid) does take the generation timestamp into consideration,
	// However, it only down to the level of seconds. This means that if a few records are created at almost the same time,
	// the order is not guaranteed. So use the `created_at` column will provider better consistency.
	if err := dbConn.Where(clusterDetails).First(&cluster).Order("created_at asc").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to find cluster with criteria")
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
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to find cluster with id: %s", clusterID)
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
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update identity_provider_id for cluster %s", id)
	}

	return nil
}

func (c clusterService) DeleteByClusterID(clusterID string) *apiErrors.ServiceError {
	dbConn := c.connectionFactory.New()
	metrics.IncreaseClusterTotalOperationsCountMetric(constants.ClusterOperationDelete)

	if err := dbConn.Delete(&api.Cluster{}, api.Cluster{ClusterID: clusterID}).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "Unable to delete cluster with cluster_id %s", clusterID)
	}

	glog.Infof("Cluster %s deleted successful", clusterID)
	metrics.IncreaseClusterSuccessOperationsCountMetric(constants.ClusterOperationDelete)
	return nil
}

func (c clusterService) FindNonEmptyClusterById(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	var cluster *api.Cluster = &api.Cluster{}

	clusterDetails := &api.Cluster{
		ClusterID: clusterID,
	}

	subQuery := dbConn.Select("cluster_id").Where("status != ? AND cluster_id = ?", constants.KafkaRequestStatusFailed, clusterID).Model(api.KafkaRequest{})
	if err := dbConn.Where(clusterDetails).Where("cluster_id IN (?)", subQuery).First(cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to find cluster with id %s", clusterID)
	}

	return cluster, nil
}

func (c clusterService) ListAllClusterIds() ([]api.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	var res []api.Cluster

	// we order them by "created_at" field instead of the default "id" field.
	// They are mostly the same as the library we use (ksuid) does take the generation timestamp into consideration,
	// However, it only down to the level of seconds. This means that if a few records are created at almost the same time,
	// the order is not guaranteed. So use the `created_at` column will provider better consistency.
	if err := dbConn.Model(&api.Cluster{}).
		Select("cluster_id").
		Where("cluster_id != '' ").
		Order("created_at asc ").
		Scan(&res).Error; err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to query by cluster info")
	}
	return res, nil
}

type ResKafkaInstanceCount struct {
	Clusterid string
	Count     int
}

func (c clusterService) FindKafkaInstanceCount(clusterIDs []string) ([]ResKafkaInstanceCount, *apiErrors.ServiceError) {
	var res []ResKafkaInstanceCount
	query := c.connectionFactory.New().
		Model(&api.KafkaRequest{}).
		Select("cluster_id as Clusterid, count(1) as Count")

	if len(clusterIDs) > 0 {
		query = query.Where("cluster_id in (?)", clusterIDs)
	}

	query = query.Group("cluster_id").Order("cluster_id asc").Scan(&res)

	if err := query.Error; err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to query by cluster info")
	}
	// the query above won't return a count for a clusterId if that cluster doesn't have any Kafkas,
	// to keep things consistent and less confusing, we will identity these ids and set their count to 0
	if len(clusterIDs) > 0 {
		countersMap := map[string]int{}
		for _, c := range res {
			countersMap[c.Clusterid] = c.Count
		}
		for _, clusterId := range clusterIDs {
			if _, ok := countersMap[clusterId]; !ok {
				res = append(res, ResKafkaInstanceCount{Clusterid: clusterId, Count: 0})
			}
		}
	}

	return res, nil
}

func (c clusterService) FindAllClusters(criteria FindClusterCriteria) ([]*api.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	var cluster []*api.Cluster

	clusterDetails := &api.Cluster{
		CloudProvider: criteria.Provider,
		Region:        criteria.Region,
		MultiAZ:       criteria.MultiAZ,
		Status:        criteria.Status,
	}

	// we order them by "created_at" field instead of the default "id" field.
	// They are mostly the same as the library we use (ksuid) does take the generation timestamp into consideration,
	// However, it only down to the level of seconds. This means that if a few records are created at almost the same time,
	// the order is not guaranteed. So use the `created_at` column will provider better consistency.
	if err := dbConn.Model(&api.Cluster{}).Where(clusterDetails).Order("created_at asc").Scan(&cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to find all clusters with criteria")
	}

	return cluster, nil
}

func (c clusterService) UpdateMultiClusterStatus(clusterIds []string, status api.ClusterStatus) *apiErrors.ServiceError {

	if status.String() == "" {
		return apiErrors.Validation("status is undefined")
	}
	if len(clusterIds) == 0 {
		return apiErrors.Validation("ids is empty")
	}

	dbConn := c.connectionFactory.New()

	if err := dbConn.Model(&api.Cluster{}).Where("cluster_id in (?)", clusterIds).
		Update("status", status).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update status: %s", clusterIds)
	}

	for rows := dbConn.RowsAffected; rows > 0; rows-- {
		if status == api.ClusterFailed {
			metrics.IncreaseClusterTotalOperationsCountMetric(constants.ClusterOperationCreate)
		}
		if status == api.ClusterReady {
			metrics.IncreaseClusterTotalOperationsCountMetric(constants.ClusterOperationCreate)
			metrics.IncreaseClusterSuccessOperationsCountMetric(constants.ClusterOperationCreate)
		}
	}

	return nil
}

type ClusterStatusCount struct {
	Status api.ClusterStatus
	Count  int
}

func (c clusterService) CountByStatus(status []api.ClusterStatus) ([]ClusterStatusCount, error) {
	dbConn := c.connectionFactory.New()
	var results []ClusterStatusCount
	if err := dbConn.Model(&api.Cluster{}).Select("status as Status, count(1) as Count").Where("status in (?)", status).Group("status").Scan(&results).Error; err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to count by status")
	}

	// if there is no count returned for a status from the above query because there is no clusters in such a status,
	// we should return the count for these as well to avoid any confusion
	if len(status) > 0 {
		countersMap := map[api.ClusterStatus]int{}
		for _, c := range results {
			countersMap[c.Status] = c.Count
		}
		for _, s := range status {
			if _, ok := countersMap[s]; !ok {
				results = append(results, ClusterStatusCount{Status: s, Count: 0})
			}
		}
	}

	return results, nil
}
