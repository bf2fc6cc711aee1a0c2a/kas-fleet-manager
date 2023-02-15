package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	kafkaTypes "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	apiErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

var kafkaStatusesThatNoLongerConsumeResourcesInTheDataPlane = []string{constants.KafkaRequestStatusDeleting.String()}

//go:generate moq -out clusterservice_moq.go . ClusterService
type ClusterService interface {
	Create(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError)
	GetClusterDNS(clusterID string) (string, *apiErrors.ServiceError)
	GetExternalID(clusterID string) (string, *apiErrors.ServiceError)
	// ListEnterpriseClustersOfAnOrganization returns a list of enterprise clusters (ClusterID, AccessKafkasViaPrivateNetwork, Cloud Provider, Region, MultiAZ and Status fields only) which belong to organization obtained from the context
	ListEnterpriseClustersOfAnOrganization(ctx context.Context) ([]*api.Cluster, *apiErrors.ServiceError)
	ListByStatus(state api.ClusterStatus) ([]api.Cluster, *apiErrors.ServiceError)
	UpdateStatus(cluster api.Cluster, status api.ClusterStatus) error
	// Update updates a Cluster. Only fields whose value is different than the
	// zero-value of their corresponding type will be updated
	Update(cluster api.Cluster) *apiErrors.ServiceError
	FindCluster(criteria FindClusterCriteria) (*api.Cluster, error)
	// FindClusterByID returns the cluster corresponding to the provided clusterID.
	// If the cluster has not been found nil is returned. If there has been an issue
	// finding the cluster an error is set
	FindClusterByID(clusterID string) (*api.Cluster, *apiErrors.ServiceError)
	GetClientID(clusterID string) (string, error)
	ListGroupByProviderAndRegion(providers []string, regions []string, status []string) ([]*ResGroupCPRegion, *apiErrors.ServiceError)
	RegisterClusterJob(clusterRequest *api.Cluster) *apiErrors.ServiceError
	DeregisterClusterJob(clusterID string) *apiErrors.ServiceError
	// DeleteByClusterID will delete the cluster from the database
	DeleteByClusterID(clusterID string) *apiErrors.ServiceError
	// HardDeleteByClusterID hard deletes a cluster from the database, no delete flag is set
	HardDeleteByClusterID(clusterID string) *apiErrors.ServiceError
	// FindNonEmptyClusterByID returns a cluster if it present and it is not empty.
	// Cluster emptiness is determined by checking whether the cluster contains Kafkas that have been provisioned, are being provisioned on it,
	// or are being deprovisioned from it i.e kafka that are not in deleting state.
	// NOTE. Kafka in "failed" are included as well since it is not a terminal status at the moment.
	FindNonEmptyClusterByID(clusterID string) (*api.Cluster, *apiErrors.ServiceError)
	// ListNonEnterpriseClusterIDs returns all the valid cluster ids in array (except enterprise clusters)
	ListNonEnterpriseClusterIDs() ([]api.Cluster, *apiErrors.ServiceError)
	// FindAllClusters return all the valid clusters in array
	FindAllClusters(criteria FindClusterCriteria) ([]*api.Cluster, error)
	// FindKafkaInstanceCount returns the kafka instance counts associated with the list of clusters. If the list is empty, it will list all clusterIDs that have Kafka instances assigned.
	// Kafkas that are in deleting state won't be included in the count as they no longer consume resources in the data plane cluster.
	FindKafkaInstanceCount(clusterIDs []string) ([]ResKafkaInstanceCount, error)
	// UpdateMultiClusterStatus updates a list of clusters' status to a status
	UpdateMultiClusterStatus(clusterIDs []string, status api.ClusterStatus) *apiErrors.ServiceError
	// CountByStatus returns the count of clusters for each given status in the database
	CountByStatus([]api.ClusterStatus) ([]ClusterStatusCount, *apiErrors.ServiceError)
	CheckClusterStatus(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError)
	// Delete will delete the cluster from the provider
	Delete(cluster *api.Cluster) (bool, *apiErrors.ServiceError)
	ConfigureAndSaveIdentityProvider(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError)
	ApplyResources(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError
	RemoveResources(cluster *api.Cluster, syncSetName string) *apiErrors.ServiceError
	// Install the strimzi operator in a given cluster
	InstallStrimzi(cluster *api.Cluster) (bool, *apiErrors.ServiceError)
	// Install the cluster logging operator for a given cluster
	InstallClusterLogging(cluster *api.Cluster, params []types.Parameter) (bool, *apiErrors.ServiceError)
	CheckStrimziVersionReady(cluster *api.Cluster, strimziVersion string) (bool, error)
	IsStrimziKafkaVersionAvailableInCluster(cluster *api.Cluster, strimziVersion string, kafkaVersion string, ibpVersion string) (bool, error)
	// FindStreamingUnitCountByClusterAndInstanceType returns kafka streaming unit counts per region, cloud provider, cluster id and instance type.
	// Data Plane clusters that are in 'failed' state are not included in the response.
	// Kafkas that are in deleting state won't be included in the count as they no longer consume resources in the data plane cluster.
	FindStreamingUnitCountByClusterAndInstanceType() (KafkaStreamingUnitCountPerClusterList, error)

	// Computes the consumed streaming unit coount per instance of a given cluster.
	// If an instance type if not contained in the returned object, it can be considered that the consumed capacity for that instance type is 0
	ComputeConsumedStreamingUnitCountPerInstanceType(clusterID string) (StreamingUnitCountPerInstanceType, error)
}

type StreamingUnitCountPerInstanceType map[kafkaTypes.KafkaInstanceType]int64

var _ ClusterService = &clusterService{}

type clusterService struct {
	connectionFactory *db.ConnectionFactory
	providerFactory   clusters.ProviderFactory
	kafkaConfig       *config.KafkaConfig
}

// NewClusterService creates a new client for the OSD Cluster Service
func NewClusterService(connectionFactory *db.ConnectionFactory, providerFactory clusters.ProviderFactory, kafkaConfig *config.KafkaConfig) ClusterService {
	return &clusterService{
		connectionFactory: connectionFactory,
		providerFactory:   providerFactory,
		kafkaConfig:       kafkaConfig,
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

// DeregisterClusterJob deregister an existing cluster if it exists
func (c clusterService) DeregisterClusterJob(clusterID string) *apiErrors.ServiceError {
	err := c.UpdateStatus(api.Cluster{ClusterID: clusterID}, api.ClusterDeprovisioning)
	if err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to deregister cluster")
	}
	return nil
}

// ListEnterpriseClustersOfAnOrganization returns a list of clusters (ClusterID, AccessKafkasViaPrivateNetwork, CloudProvider, Region, MultiAZ and Status fields only) which belong to organization obtained from the context
func (c clusterService) ListEnterpriseClustersOfAnOrganization(ctx context.Context) ([]*api.Cluster, *apiErrors.ServiceError) {
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorUnauthenticated, err, "user not authenticated")
	}

	user, _ := claims.GetUsername()
	if user == "" {
		return nil, apiErrors.Unauthenticated("user not authenticated")
	}

	orgId, _ := claims.GetOrgId()

	dbConn := c.connectionFactory.New().
		Model(&api.Cluster{}).Select("cluster_id, status, access_kafkas_via_private_network, cloud_provider, region, multi_az")

	var clusters []*api.Cluster

	if err := dbConn.Where("organization_id = ? AND cluster_type = ?", orgId, api.EnterpriseDataPlaneClusterType.String()).Scan(&clusters).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return []*api.Cluster{}, nil
		}
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to list clusters")
	}

	return clusters, nil
}

// Create Creates a new OpenShift/k8s cluster via the provider and save the details of the cluster in the database
// Returns the newly created cluster object
func (c clusterService) Create(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()
	r := &types.ClusterRequest{
		CloudProvider:  cluster.CloudProvider,
		Region:         cluster.Region,
		MultiAZ:        cluster.MultiAZ,
		AdditionalSpec: cluster.ProviderSpec,
	}
	provider, err := c.providerFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get provider implementation")
	}
	clusterSpec, err := provider.Create(r)
	if err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to create cluster")
	}

	cluster.ClusterID = clusterSpec.InternalID
	cluster.ExternalID = clusterSpec.ExternalID
	cluster.Status = clusterSpec.Status
	if clusterSpec.AdditionalInfo != nil {
		clusterInfo, err := json.Marshal(clusterSpec.AdditionalInfo)
		if err != nil {
			return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to marshal JSON value")
		}
		cluster.ClusterSpec = clusterInfo
	}

	if err := dbConn.Save(cluster).Error; err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to save data to db")
	}

	return cluster, nil
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

	p, err := c.providerFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return "", apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get provider implementation")
	}

	// If the clusterDNS is not present in the database, retrieve it from OCM
	clusterDNS, err := p.GetClusterDNS(buildClusterSpec(cluster))
	if err != nil {
		return "", apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get cluster DNS from OCM")
	}
	cluster.ClusterDNS = clusterDNS
	if err := c.Update(*cluster); err != nil {
		return "", apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update cluster DNS")
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

func (c clusterService) Update(cluster api.Cluster) *apiErrors.ServiceError {
	if cluster.ID == "" {
		return apiErrors.Validation("id is undefined")
	}

	// by specifying the Model with a non-empty primary key we ensure
	// only the record with that primary key is updated
	dbConn := c.connectionFactory.New().Model(cluster)

	if err := dbConn.Updates(cluster).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update cluster")
	}

	return nil
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

	if err := dbConn.Model(&api.Cluster{}).Where(query, arg).Updates(map[string]interface{}{"status": status}).Error; err != nil {
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
	Provider              string
	Region                string
	MultiAZ               bool
	Status                api.ClusterStatus
	SupportedInstanceType string
	ExternalID            string
}

func (c clusterService) FindCluster(criteria FindClusterCriteria) (*api.Cluster, error) {
	dbConn := c.connectionFactory.New()

	var cluster api.Cluster

	clusterDetails := &api.Cluster{
		CloudProvider: criteria.Provider,
		Region:        criteria.Region,
		MultiAZ:       criteria.MultiAZ,
		Status:        criteria.Status,
		ExternalID:    criteria.ExternalID,
	}

	// filter by supported instance type
	if criteria.SupportedInstanceType != "" {
		dbConn = dbConn.Where("supported_instance_type like ?", fmt.Sprintf("%%%s%%", criteria.SupportedInstanceType))
	}

	// we order them by "created_at" field instead of the default "id" field.
	// They are mostly the same as the library we use (xid) does take the generation timestamp into consideration,
	// However, it only down to the level of seconds. This means that if a few records are created at almost the same time,
	// the order is not guaranteed. So use the `created_at` column will provider better consistency.
	if err := dbConn.Where(clusterDetails).First(&cluster).Order("created_at asc").Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
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

func (c clusterService) GetClientID(clusterID string) (string, error) {
	if cluster, err := c.FindClusterByID(clusterID); err != nil {
		return "", err
	} else {
		if cluster == nil {
			return "", nil
		}
		return cluster.ClientID, nil
	}
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

func (c clusterService) HardDeleteByClusterID(clusterID string) *apiErrors.ServiceError {
	dbConn := c.connectionFactory.New()
	metrics.IncreaseClusterTotalOperationsCountMetric(constants.ClusterOperationHardDelete)

	if err := dbConn.Unscoped().Delete(&api.Cluster{}, api.Cluster{ClusterID: clusterID}).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "Unable to hard delete cluster with cluster_id %s", clusterID)
	}

	glog.Infof("Cluster %s hard deleted successful", clusterID)
	metrics.IncreaseClusterSuccessOperationsCountMetric(constants.ClusterOperationHardDelete)
	return nil
}

func (c clusterService) FindNonEmptyClusterByID(clusterID string) (*api.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	var cluster *api.Cluster = &api.Cluster{}

	clusterDetails := &api.Cluster{
		ClusterID: clusterID,
	}

	subQuery := dbConn.Select("cluster_id").Where("status != ? AND cluster_id = ?", constants.KafkaRequestStatusDeleting, clusterID).Model(dbapi.KafkaRequest{})
	if err := dbConn.Where(clusterDetails).Where("cluster_id IN (?)", subQuery).First(cluster).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to find cluster with id %s", clusterID)
	}

	return cluster, nil
}

func (c clusterService) ListNonEnterpriseClusterIDs() ([]api.Cluster, *apiErrors.ServiceError) {
	dbConn := c.connectionFactory.New()

	var res []api.Cluster

	// we order them by "created_at" field instead of the default "id" field.
	// They are mostly the same as the library we use (xid) does take the generation timestamp into consideration,
	// However, it only down to the level of seconds. This means that if a few records are created at almost the same time,
	// the order is not guaranteed. So use the `created_at` column will provider better consistency.
	if err := dbConn.Model(&api.Cluster{}).
		Select("cluster_id").
		Where("cluster_id != '' ").
		Where("cluster_type != ? ", api.EnterpriseDataPlaneClusterType.String()). // don't include enterprise clusters
		Order("created_at asc ").
		Scan(&res).Error; err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to query by cluster info")
	}
	return res, nil
}

type ResKafkaInstanceCount struct {
	ClusterID string
	Count     int
}

func (c clusterService) GetExternalID(clusterID string) (string, *apiErrors.ServiceError) {
	cluster, err := c.FindClusterByID(clusterID)
	if err != nil {
		return "", err
	}
	if cluster == nil {
		return "", apiErrors.GeneralError("failed to get External ID for clusterID %s", clusterID)
	}
	return cluster.ExternalID, nil
}

func (c clusterService) FindKafkaInstanceCount(clusterIDs []string) ([]ResKafkaInstanceCount, error) {
	var kafkas []*dbapi.KafkaRequest

	query := c.connectionFactory.New().
		Model(&dbapi.KafkaRequest{}).
		Where("status not in (?)", kafkaStatusesThatNoLongerConsumeResourcesInTheDataPlane)

	if len(clusterIDs) > 0 {
		query = query.Where("cluster_id in (?)", clusterIDs)
	}

	query = query.Scan(&kafkas)

	if err := query.Error; err != nil {
		return nil, err
	}

	clusterIDCountMap := map[string]int{}

	var res []ResKafkaInstanceCount

	for _, k := range kafkas {
		kafkaInstanceSize, e := c.kafkaConfig.GetKafkaInstanceSize(k.InstanceType, k.SizeId)
		if e != nil {
			return nil, e
		}
		clusterIDCountMap[k.ClusterID] += kafkaInstanceSize.CapacityConsumed
	}

	// the query above won't return a count for a clusterId if that cluster doesn't have any Kafkas,
	// to keep things consistent and less confusing, we will identity these ids and set their count to 0
	if len(clusterIDs) > 0 {
		for _, clusterID := range clusterIDs {
			if _, ok := clusterIDCountMap[clusterID]; !ok {
				res = append(res, ResKafkaInstanceCount{ClusterID: clusterID, Count: 0})
			}
		}
	}

	for k, v := range clusterIDCountMap {
		res = append(res, ResKafkaInstanceCount{ClusterID: k, Count: v})
	}

	return res, nil
}

func (c clusterService) FindAllClusters(criteria FindClusterCriteria) ([]*api.Cluster, error) {
	dbConn := c.connectionFactory.New().
		Model(&api.Cluster{})

	var clusters []*api.Cluster

	clusterDetails := &api.Cluster{
		CloudProvider: criteria.Provider,
		Region:        criteria.Region,
		MultiAZ:       criteria.MultiAZ,
		Status:        criteria.Status,
		ExternalID:    criteria.ExternalID,
	}

	// filter by supported instance type
	if criteria.SupportedInstanceType != "" {
		dbConn.Where("supported_instance_type like ?", fmt.Sprintf("%%%s%%", criteria.SupportedInstanceType))
	}
	// we order them by "created_at" field instead of the default "id" field.
	// They are mostly the same as the library we use (xid) does take the generation timestamp into consideration,
	// However, it only down to the level of seconds. This means that if a few records are created at almost the same time,
	// the order is not guaranteed. So use the `created_at` column will provider better consistency.
	if err := dbConn.Where(clusterDetails).Order("created_at asc").Scan(&clusters).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return clusters, nil
}

func (c clusterService) UpdateMultiClusterStatus(clusterIDs []string, status api.ClusterStatus) *apiErrors.ServiceError {
	if status.String() == "" {
		return apiErrors.Validation("status is undefined")
	}
	if len(clusterIDs) == 0 {
		return apiErrors.Validation("ids is empty")
	}

	dbConn := c.connectionFactory.New().
		Model(&api.Cluster{}).
		Where("cluster_id in (?)", clusterIDs)

	if status == api.ClusterDeprovisioning {
		dbConn = dbConn.Where("status != ?", api.ClusterCleanup.String())
	}

	if err := dbConn.Update("status", status).Error; err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update status: %s", clusterIDs)
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

func (c clusterService) CountByStatus(status []api.ClusterStatus) ([]ClusterStatusCount, *apiErrors.ServiceError) {
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

func (c clusterService) CheckClusterStatus(cluster *api.Cluster) (*api.Cluster, *apiErrors.ServiceError) {
	p, err := c.providerFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get provider implementation")
	}

	clusterSpec, err := p.CheckClusterStatus(buildClusterSpec(cluster))
	if err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to check cluster status")
	}

	cluster.Status = clusterSpec.Status
	cluster.StatusDetails = clusterSpec.StatusDetails
	cluster.ClusterSpec = clusterSpec.AdditionalInfo
	if clusterSpec.ExternalID != "" && cluster.ExternalID == "" {
		cluster.ExternalID = clusterSpec.ExternalID
	}
	if err := c.Update(*cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func (c clusterService) Delete(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
	p, err := c.providerFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return false, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get provider implementation")
	}
	if removed, err := p.Delete(buildClusterSpec(cluster)); err != nil {
		return false, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to delete the cluster from the provider")
	} else {
		return removed, nil
	}
}

func (c clusterService) ConfigureAndSaveIdentityProvider(cluster *api.Cluster, identityProviderInfo types.IdentityProviderInfo) (*api.Cluster, *apiErrors.ServiceError) {
	if cluster.IdentityProviderID != "" {
		return cluster, nil
	}
	p, err := c.providerFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get provider implementation")
	}
	providerInfo, err := p.AddIdentityProvider(buildClusterSpec(cluster), identityProviderInfo)
	if err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to add identity provider")
	}
	// need to review this if multiple identity providers are supported
	cluster.IdentityProviderID = providerInfo.OpenID.ID
	if err := c.Update(*cluster); err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to update cluster")
	}
	return cluster, nil
}

func (c clusterService) ApplyResources(cluster *api.Cluster, resources types.ResourceSet) *apiErrors.ServiceError {
	p, err := c.providerFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get provider implementation")
	}
	if _, err := p.ApplyResources(buildClusterSpec(cluster), resources); err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to apply resources %s", resources.Name)
	}
	return nil
}

func (c clusterService) RemoveResources(cluster *api.Cluster, syncSetName string) *apiErrors.ServiceError {
	p, err := c.providerFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get provider implementation")
	}
	if err := p.RemoveResources(buildClusterSpec(cluster), syncSetName); err != nil {
		return apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to remove resources %s", syncSetName)
	}
	return nil
}

func (c clusterService) InstallStrimzi(cluster *api.Cluster) (bool, *apiErrors.ServiceError) {
	p, err := c.providerFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return false, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get provider implementation")
	}
	if ready, err := p.InstallStrimzi(buildClusterSpec(cluster)); err != nil {
		return ready, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to install strimzi for cluster %s", cluster.ClusterID)
	} else {
		return ready, nil
	}
}

func (c clusterService) InstallClusterLogging(cluster *api.Cluster, params []types.Parameter) (bool, *apiErrors.ServiceError) {
	p, err := c.providerFactory.GetProvider(cluster.ProviderType)
	if err != nil {
		return false, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get provider implementation")
	}
	if ready, err := p.InstallClusterLogging(buildClusterSpec(cluster), params); err != nil {
		return ready, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to install cluster-logging for cluster %s", cluster.ClusterID)
	} else {
		return ready, nil
	}
}

func buildClusterSpec(cluster *api.Cluster) *types.ClusterSpec {
	return &types.ClusterSpec{
		InternalID:     cluster.ClusterID,
		ExternalID:     cluster.ExternalID,
		Status:         cluster.Status,
		MultiAZ:        cluster.MultiAZ,
		Region:         cluster.Region,
		CloudProvider:  cluster.CloudProvider,
		AdditionalInfo: cluster.ClusterSpec,
		StatusDetails:  cluster.StatusDetails,
	}
}

func (c clusterService) CheckStrimziVersionReady(cluster *api.Cluster, strimziVersion string) (bool, error) {
	readyStrimziVersions, err := cluster.GetAvailableAndReadyStrimziVersions()
	if err != nil {
		return false, err
	}
	for _, version := range readyStrimziVersions {
		if version.Version == strimziVersion {
			return true, nil
		}
	}
	return false, nil
}

func (c clusterService) IsStrimziKafkaVersionAvailableInCluster(cluster *api.Cluster, strimziVersion string, kafkaVersion string, ibpVersion string) (bool, error) {
	readyStrimziVersions, err := cluster.GetAvailableAndReadyStrimziVersions()
	if err != nil {
		return false, err
	}
	for _, version := range readyStrimziVersions {
		if version.Version == strimziVersion {
			kVvalid := arrays.AnyMatch(version.KafkaVersions, func(v api.KafkaVersion) bool { return v.Version == kafkaVersion })
			ibpVvalid := arrays.AnyMatch(version.KafkaIBPVersions, func(v api.KafkaIBPVersion) bool { return v.Version == ibpVersion })

			return kVvalid && ibpVvalid, nil
		}
	}
	return false, nil
}

type KafkaStreamingUnitCountPerCluster struct {
	Region        string
	InstanceType  string
	ID            string
	ClusterId     string
	Count         int32
	CloudProvider string
	MaxUnits      int32
	Status        string
	ClusterType   string
}

func (k KafkaStreamingUnitCountPerCluster) isSame(kafkaPerRegionFromDB *KafkaPerClusterCount) bool {
	// A KafkaPerClusterCount instance that doesn't have a ClusterID set
	// means that it contains information about kafka instances that have not
	// been allocated to any cluster yet. In that case, it can never be the same
	// as any existing Cluster. Thus, in that case we always return false.
	if kafkaPerRegionFromDB.ClusterId == "" {
		return false
	}

	return k.CloudProvider == kafkaPerRegionFromDB.CloudProvider &&
		k.ClusterId == kafkaPerRegionFromDB.ClusterId &&
		k.InstanceType == kafkaPerRegionFromDB.InstanceType &&
		k.Region == kafkaPerRegionFromDB.Region
}

func (k KafkaStreamingUnitCountPerCluster) FreeStreamingUnits() int32 {
	return k.MaxUnits - k.Count
}

type KafkaStreamingUnitCountPerClusterList []KafkaStreamingUnitCountPerCluster

func (kafkaStreamingUnitCountPerClusterList KafkaStreamingUnitCountPerClusterList) GetStreamingUnitCountForClusterAndInstanceType(clusterId, instanceType string) int {
	for _, KafkaStreamingUnitCountPerCluster := range kafkaStreamingUnitCountPerClusterList {
		if KafkaStreamingUnitCountPerCluster.ClusterId == clusterId && KafkaStreamingUnitCountPerCluster.InstanceType == instanceType {
			return int(KafkaStreamingUnitCountPerCluster.Count)
		}
	}

	return 0
}

// KafkaPerClusterCount is a struct used to query the database using a "group by" clause
type KafkaPerClusterCount struct {
	Region        string
	InstanceType  string
	ClusterId     string
	Count         int32
	CloudProvider string
	SizeId        string
}

type ClusterSelection struct {
	CloudProvider         string
	ID                    string
	ClusterID             string
	Region                string
	SupportedInstanceType string
	DynamicCapacityInfo   api.JSON
	Status                string
	ClusterType           string
}

func (c *clusterService) FindStreamingUnitCountByClusterAndInstanceType() (KafkaStreamingUnitCountPerClusterList, error) {

	var clusters []*ClusterSelection
	dbConn := c.connectionFactory.New().
		Model(&api.Cluster{}).
		Where("status != ?", api.ClusterFailed)

	if err := dbConn.Scan(&clusters).Error; err != nil {
		return nil, errors.Wrap(err, "failed to list data plane clusters")
	}

	streamingUnitsCountPerCluster := KafkaStreamingUnitCountPerClusterList{}

	// pre-populate regions count with zero count values.
	// This is useful and it ensures that the count drops to 0 for each instance type and region when all the kafkas are removed
	for _, clusterSelection := range clusters {
		supportedInstanceTypes := strings.Split(clusterSelection.SupportedInstanceType, ",")

		c := &api.Cluster{DynamicCapacityInfo: clusterSelection.DynamicCapacityInfo}
		clusterDynamicCapacityInfo := c.RetrieveDynamicCapacityInfo()

		for _, supportedInstanceType := range supportedInstanceTypes {
			instanceType := strings.TrimSpace(supportedInstanceType)
			if instanceType == "" {
				continue
			}

			var maxUnits int32 = 0
			instanceCapacityInfo, ok := clusterDynamicCapacityInfo[instanceType]
			if ok {
				maxUnits = instanceCapacityInfo.MaxUnits
			}

			streamingUnitsCountPerCluster = append(streamingUnitsCountPerCluster, KafkaStreamingUnitCountPerCluster{
				CloudProvider: clusterSelection.CloudProvider,
				ID:            clusterSelection.ID,
				ClusterId:     clusterSelection.ClusterID,
				InstanceType:  instanceType,
				Region:        clusterSelection.Region,
				Count:         0,
				MaxUnits:      maxUnits,
				Status:        clusterSelection.Status,
				ClusterType:   clusterSelection.ClusterType,
			})
		}
	}

	dbConn = c.connectionFactory.New()
	var kafkasPerCluster []*KafkaPerClusterCount
	if err := dbConn.Model(&dbapi.KafkaRequest{}).
		Select("cloud_provider, region, count(1) as Count, size_id, cluster_id, instance_type").
		Group("size_id, cluster_id, cloud_provider, region, instance_type").
		Where("status not in (?)", kafkaStatusesThatNoLongerConsumeResourcesInTheDataPlane).
		Scan(&kafkasPerCluster).Error; err != nil {
		return nil, errors.Wrap(err, "failed to perform count query on kafkas table")
	}

	for _, kafkaCountPerCluster := range kafkasPerCluster {
		instSize, err := c.kafkaConfig.GetKafkaInstanceSize(kafkaCountPerCluster.InstanceType, kafkaCountPerCluster.SizeId)
		if err != nil {
			return nil, err
		}

		streamingUnitCount := int32(instSize.CapacityConsumed) * kafkaCountPerCluster.Count
		for i, streamingUnitCountPerRegion := range streamingUnitsCountPerCluster {
			if streamingUnitCountPerRegion.isSame(kafkaCountPerCluster) {
				streamingUnitsCountPerCluster[i].Count += streamingUnitCount
				break
			}
		}
	}

	return streamingUnitsCountPerCluster, nil
}

type ClusterSizeCountPerInstanceType struct {
	SizeId       string
	Count        int64
	InstanceType string
}

func (c *clusterService) ComputeConsumedStreamingUnitCountPerInstanceType(clusterID string) (StreamingUnitCountPerInstanceType, error) {
	dbConn := c.connectionFactory.New()
	var sizeCountsPerInstanceType []*ClusterSizeCountPerInstanceType
	if err := dbConn.Model(&dbapi.KafkaRequest{}).
		Select("size_id, instance_type, count(1) as Count").
		Group("size_id, instance_type").
		Where("cluster_id = ?", clusterID).
		Where("status not in (?)", kafkaStatusesThatNoLongerConsumeResourcesInTheDataPlane).
		Scan(&sizeCountsPerInstanceType).Error; err != nil {
		return nil, apiErrors.NewWithCause(apiErrors.ErrorGeneral, err, "failed to get count of sizes of a cluster")
	}

	streamingUnitCountPerInstanceType := StreamingUnitCountPerInstanceType{
		kafkaTypes.DEVELOPER: 0,
		kafkaTypes.STANDARD:  0,
	}

	for _, sizeCount := range sizeCountsPerInstanceType {
		instSize, err := c.kafkaConfig.GetKafkaInstanceSize(sizeCount.InstanceType, sizeCount.SizeId)
		if err != nil {
			return nil, err
		}

		streamingUnitCount := int64(instSize.CapacityConsumed) * sizeCount.Count
		instanceType := kafkaTypes.KafkaInstanceType(sizeCount.InstanceType)
		existingCount, ok := streamingUnitCountPerInstanceType[instanceType]

		if !ok {
			streamingUnitCountPerInstanceType[instanceType] = streamingUnitCount
		} else {
			streamingUnitCountPerInstanceType[instanceType] = existingCount + streamingUnitCount
		}
	}

	return streamingUnitCountPerInstanceType, nil
}
