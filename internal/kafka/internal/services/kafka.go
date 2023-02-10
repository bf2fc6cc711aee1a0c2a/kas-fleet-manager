package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"sync"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/utils/arrays"
	"gorm.io/gorm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/authorization"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/queryparser"

	"strconv"
	"time"

	"github.com/golang/glog"

	managedkafka "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/aws"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
)

var kafkaDeletionStatuses = []string{constants.KafkaRequestStatusDeleting.String(), constants.KafkaRequestStatusDeprovision.String()}
var kafkaManagedCRStatuses = []string{
	constants.KafkaRequestStatusProvisioning.String(),
	constants.KafkaRequestStatusDeprovision.String(),
	constants.KafkaRequestStatusReady.String(),
	constants.KafkaRequestStatusFailed.String(),
	constants.KafkaRequestStatusSuspended.String(),
	constants.KafkaRequestStatusSuspending.String(),
	constants.KafkaRequestStatusResuming.String(),
}

type KafkaRoutesAction string

func (a KafkaRoutesAction) String() string {
	return string(a)
}

const (
	KafkaRoutesActionCreate KafkaRoutesAction = "CREATE"
	KafkaRoutesActionDelete KafkaRoutesAction = "DELETE"
)

const CanaryServiceAccountPrefix = "canary"

type CNameRecordStatus struct {
	Id     *string
	Status *string
}

//go:generate moq -out kafkaservice_moq.go . KafkaService
type KafkaService interface {
	// PrepareKafkaRequest sets any required information (i.e. bootstrap server host, sso client id and secret)
	// to the Kafka Request record in the database. The kafka request will also be updated with an updated_at
	// timestamp and the corresponding cluster identifier.
	PrepareKafkaRequest(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError
	// Get method will retrieve the kafkaRequest instance that the give ctx has access to from the database.
	// This should be used when you want to make sure the result is filtered based on the request context.
	Get(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError)
	// GetByID method will retrieve the KafkaRequest instance from the database without checking any permissions.
	// You should only use this if you are sure permission check is not required.
	GetByID(id string) (*dbapi.KafkaRequest, *errors.ServiceError)
	// Delete cleans up all dependencies for a Kafka request and soft deletes the Kafka Request record from the database.
	// The Kafka Request in the database will be updated with a deleted_at timestamp.
	Delete(*dbapi.KafkaRequest) *errors.ServiceError
	List(ctx context.Context, listArgs *services.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError)
	// Lists all kafkas. As this returns all Kafka requests without need for authentication, this should only be used for internal purposes
	ListAll() (dbapi.KafkaList, *errors.ServiceError)
	ListKafkasToBePromoted() ([]*dbapi.KafkaRequest, *errors.ServiceError)
	GetManagedKafkaByClusterID(clusterID string) ([]managedkafka.ManagedKafka, *errors.ServiceError)
	// GenerateReservedManagedKafkasByClusterID returns a list of reserved managed
	// kafkas for a given clusterID. The number of generated reserved managed
	// kafkas in the cluster is the sum of the specified number of reserved
	// instances among all instance types supported by the cluster.
	// If the cluster is not in ready status the result is an empty list.
	// Generated kafka names have the following naming schema:
	// reserved-kafka-<instance_type>-<kafka_number> where kafka_number goes from
	// 1..<num_reserved_instances>_for_the_given_instance_type>
	// Each generated reserved kafka has a namespace equal to its name
	GenerateReservedManagedKafkasByClusterID(clusterID string) ([]managedkafka.ManagedKafka, *errors.ServiceError)
	RegisterKafkaJob(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError
	ListByStatus(status ...constants.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError)
	// UpdateStatus change the status of the Kafka cluster
	// The returned boolean is to be used to know if the update has been tried or not. An update is not tried if the
	// original status is 'deprovision' (cluster in deprovision state can't be change state) or if the final status is the
	// same as the original status. The error will contain any error encountered when attempting to update or the reason
	// why no attempt has been done
	UpdateStatus(id string, status constants.KafkaStatus) (bool, *errors.ServiceError)
	Update(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError
	// Updates() updates the given fields of a kafka. This takes in a map so that even zero-fields can be updated.
	// Use this only when you want to update the multiple columns that may contain zero-fields, otherwise use the `KafkaService.Update()` method.
	// See https://gorm.io/docs/update.html#Updates-multiple-columns for more info
	Updates(kafkaRequest *dbapi.KafkaRequest, values map[string]interface{}) *errors.ServiceError
	ChangeKafkaCNAMErecords(kafkaRequest *dbapi.KafkaRequest, action KafkaRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError)
	GetCNAMERecordStatus(kafkaRequest *dbapi.KafkaRequest) (*CNameRecordStatus, error)
	AssignInstanceType(owner string, organisationID string) (types.KafkaInstanceType, *errors.ServiceError)
	RegisterKafkaDeprovisionJob(ctx context.Context, id string) *errors.ServiceError
	// DeprovisionKafkaForUsers registers all kafkas for deprovisioning given the list of owners
	DeprovisionKafkaForUsers(users []string) *errors.ServiceError
	DeprovisionExpiredKafkas() *errors.ServiceError
	CountByStatus(status []constants.KafkaStatus) ([]KafkaStatusCount, error)
	ListKafkasWithRoutesNotCreated() ([]*dbapi.KafkaRequest, *errors.ServiceError)
	VerifyAndUpdateKafkaAdmin(ctx context.Context, kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError
	ListComponentVersions() ([]KafkaComponentVersions, error)
	HasAvailableCapacityInRegion(kafkaRequest *dbapi.KafkaRequest) (bool, *errors.ServiceError)
	// GetAvailableSizesInRegion returns a list of ids of the Kafka instance sizes that can still be created according to the specified criteria
	GetAvailableSizesInRegion(criteria *FindClusterCriteria) ([]string, *errors.ServiceError)
	ValidateBillingAccount(externalId string, instanceType types.KafkaInstanceType, kafkaBillingModelID string, billingCloudAccountId string, marketplace *string) *errors.ServiceError
	AssignBootstrapServerHost(kafkaRequest *dbapi.KafkaRequest) error
	// IsQuotaEntitlementActive checks if the user/organisation have an active entitlement to the quota
	// used by the given Kafka instance.
	//
	// It returns true if the user has an active quota entitlement and false if not.
	// It returns false and an error if it encounters any issues while trying to check the quota entitlement status.
	IsQuotaEntitlementActive(kafkaRequest *dbapi.KafkaRequest) (bool, error)

	// MGDSTRM-10012 temporarily add method to reconcile updating the zero-value
	// of ExpiredAt. Remove this method when functionality has been rolled out
	// to stage and prod
	UpdateZeroValueOfKafkaRequestsExpiredAt() error
}

var _ KafkaService = &kafkaService{}

type kafkaService struct {
	connectionFactory        *db.ConnectionFactory
	clusterService           ClusterService
	keycloakService          sso.KeycloakService
	kafkaConfig              *config.KafkaConfig
	awsConfig                *config.AWSConfig
	quotaServiceFactory      QuotaServiceFactory
	mu                       sync.Mutex
	awsClientFactory         aws.ClientFactory
	authService              authorization.Authorization
	dataplaneClusterConfig   *config.DataplaneClusterConfig
	providerConfig           *config.ProviderConfig
	clusterPlacementStrategy ClusterPlacementStrategy
}

func NewKafkaService(connectionFactory *db.ConnectionFactory, clusterService ClusterService, keycloakService sso.KafkaKeycloakService, kafkaConfig *config.KafkaConfig, dataplaneClusterConfig *config.DataplaneClusterConfig, awsConfig *config.AWSConfig, quotaServiceFactory QuotaServiceFactory, awsClientFactory aws.ClientFactory, authorizationService authorization.Authorization, providerConfig *config.ProviderConfig, clusterPlacementStrategy ClusterPlacementStrategy) *kafkaService {
	return &kafkaService{
		connectionFactory:        connectionFactory,
		clusterService:           clusterService,
		keycloakService:          keycloakService,
		kafkaConfig:              kafkaConfig,
		awsConfig:                awsConfig,
		quotaServiceFactory:      quotaServiceFactory,
		awsClientFactory:         awsClientFactory,
		authService:              authorizationService,
		dataplaneClusterConfig:   dataplaneClusterConfig,
		providerConfig:           providerConfig,
		clusterPlacementStrategy: clusterPlacementStrategy,
	}
}

func (k *kafkaService) ValidateBillingAccount(externalId string, instanceType types.KafkaInstanceType, billingModelID string, billingCloudAccountId string, marketplace *string) *errors.ServiceError {
	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(k.kafkaConfig.Quota.Type))
	if factoryErr != nil {
		return errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unable to check quota during billing account validation")
	}

	return quotaService.ValidateBillingAccount(externalId, instanceType, billingModelID, billingCloudAccountId, marketplace)
}

func (k *kafkaService) HasAvailableCapacityInRegion(kafkaRequest *dbapi.KafkaRequest) (bool, *errors.ServiceError) {
	// get region limit for instance type
	regInstTypeLimit, e := k.providerConfig.GetInstanceLimit(kafkaRequest.Region, kafkaRequest.CloudProvider, kafkaRequest.InstanceType)
	if e != nil {
		return false, e
	}

	if regInstTypeLimit != nil && int64(*regInstTypeLimit) == 0 {
		return false, nil
	}

	// if auto scaling is enabled and no limit set - capacity is available in the region
	if k.dataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() && regInstTypeLimit == nil {
		return true, nil
	}
	// check capacity
	return k.capacityAvailableForRegionAndInstanceType(regInstTypeLimit, kafkaRequest)
}

func (k *kafkaService) capacityAvailableForRegionAndInstanceType(instTypeRegCapacity *int, kafkaRequest *dbapi.KafkaRequest) (bool, *errors.ServiceError) {
	errMessage := fmt.Sprintf("failed to check kafka capacity for region '%s' and instance type '%s'", kafkaRequest.Region, kafkaRequest.InstanceType)

	dbConn := k.connectionFactory.New()

	var count int64

	var kafkas []*dbapi.KafkaRequest

	if err := dbConn.Model(&dbapi.KafkaRequest{}).
		Where("region = ?", kafkaRequest.Region).
		Where("cloud_provider = ?", kafkaRequest.CloudProvider).
		Where("instance_type = ?", kafkaRequest.InstanceType).
		Scan(&kafkas).Error; err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, errMessage)
	}

	for _, kafka := range kafkas {
		kafkaInstanceSize, e := k.kafkaConfig.GetKafkaInstanceSize(kafka.InstanceType, kafka.SizeId)
		if e != nil {
			return false, errors.NewWithCause(errors.ErrorInstancePlanNotSupported, e, errMessage)
		}
		count += int64(kafkaInstanceSize.CapacityConsumed)
	}

	kafkaInstanceSize, e := k.kafkaConfig.GetKafkaInstanceSize(kafkaRequest.InstanceType, kafkaRequest.SizeId)
	if e != nil {
		return false, errors.NewWithCause(errors.ErrorInstancePlanNotSupported, e, errMessage)
	}

	count += int64(kafkaInstanceSize.CapacityConsumed)

	return instTypeRegCapacity == nil || count <= int64(*instTypeRegCapacity), nil
}

func (k *kafkaService) GetAvailableSizesInRegion(criteria *FindClusterCriteria) ([]string, *errors.ServiceError) {
	if criteria == nil {
		err := errors.GeneralError("unable to get available sizes in region: criteria was not specified")
		logger.Logger.Error(err)
		return nil, err
	}

	supportedInstanceTypes := k.kafkaConfig.SupportedInstanceTypes.Configuration
	instanceType, err := supportedInstanceTypes.GetKafkaInstanceTypeByID(criteria.SupportedInstanceType)
	if err != nil {
		err := errors.InstanceTypeNotSupported("unable to get available sizes in region: %s", err.Error())
		logger.Logger.Error(err)
		return nil, err
	}

	indexOfBiggestKafkaSizeAllowed := -1
	// The kafka size list configuration must always be ordered starting with the smallest unit.
	// The following finds the largest Kafka size that is still available in this region. Anything smaller than this
	// size will also be considered as available to create with the remaining capacity.
	for i := len(instanceType.Sizes) - 1; i >= 0; i-- {
		kafka := &dbapi.KafkaRequest{
			CloudProvider: criteria.Provider,
			Region:        criteria.Region,
			InstanceType:  criteria.SupportedInstanceType,
			MultiAZ:       criteria.MultiAZ,
			SizeId:        instanceType.Sizes[i].Id,
		}

		// Check against region limits
		hasCapacity, err := k.HasAvailableCapacityInRegion(kafka)
		if err != nil {
			logger.Logger.Error(err)
			return nil, err
		}

		if hasCapacity && k.dataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
			indexOfBiggestKafkaSizeAllowed = i
			break
		}
		if hasCapacity && !k.dataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() {
			// Check if there is an available cluster in the region that can fit this Kafka instance type and size
			cluster, err := k.clusterPlacementStrategy.FindCluster(kafka)
			if err != nil {
				logger.Logger.Error(err)
				return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to find data plane cluster for kafka with criteria '%v'", criteria)
			}
			if cluster != nil {
				indexOfBiggestKafkaSizeAllowed = i
				break
			}
		}
	}

	var availableSizes []string
	if indexOfBiggestKafkaSizeAllowed != -1 {
		for _, size := range instanceType.Sizes[0 : indexOfBiggestKafkaSizeAllowed+1] {
			availableSizes = append(availableSizes, size.Id)
		}
	}
	return availableSizes, nil
}

func (k *kafkaService) AssignInstanceType(owner string, organisationId string) (types.KafkaInstanceType, *errors.ServiceError) {
	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(k.kafkaConfig.Quota.Type))
	if factoryErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unable to check quota")
	}

	for _, instanceType := range k.kafkaConfig.SupportedInstanceTypes.Configuration.SupportedKafkaInstanceTypes {
		if instanceType.Id == types.DEVELOPER.String() {
			continue
		}
		for _, bm := range instanceType.SupportedBillingModels {
			hasQuota, err := quotaService.CheckIfQuotaIsDefinedForInstanceType(owner, organisationId, types.KafkaInstanceType(instanceType.Id), bm)
			if err != nil {
				return "", err
			}
			if hasQuota {
				return types.KafkaInstanceType(instanceType.Id), nil
			}
		}
	}

	return types.DEVELOPER, nil
}

// reserveQuota - reserves quota for the given kafka request. If a RHOSAK quota has been assigned, it will try to reserve RHOSAK quota, otherwise it will try with RHOSAKTrial
func (k *kafkaService) reserveQuota(kafkaRequest *dbapi.KafkaRequest) (subscriptionId string, err *errors.ServiceError) {
	if kafkaRequest.InstanceType == types.DEVELOPER.String() {
		instType, err := k.kafkaConfig.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(kafkaRequest.InstanceType)

		if err != nil {
			return "", errors.NewWithCause(errors.ErrorGeneral, err, "unable to reserve quota")
		}

		if !k.kafkaConfig.Quota.AllowDeveloperInstance {
			return "", errors.NewWithCause(errors.ErrorForbidden, err, "kafka %s instances are not allowed", instType.DisplayName)
		}

		//N DEVELOPER instance is admitted. Let's check if the user already owns N instances
		dbConn := k.connectionFactory.New()
		var count int64
		if err := dbConn.Model(&dbapi.KafkaRequest{}).
			Where("instance_type = ?", types.DEVELOPER).
			Where("owner = ?", kafkaRequest.Owner).
			Where("organisation_id = ?", kafkaRequest.OrganisationId).
			Count(&count).
			Error; err != nil {
			return "", errors.NewWithCause(errors.ErrorGeneral, err, "failed to count kafka %s instances", instType.DisplayName)
		}

		maxAllowedDeveloperInstances := k.kafkaConfig.Quota.MaxAllowedDeveloperInstances

		if count >= int64(maxAllowedDeveloperInstances) {
			return "", errors.TooManyKafkaInstancesReached(fmt.Sprintf("only %d %s instance is allowed", maxAllowedDeveloperInstances, instType.DisplayName))
		}
	}

	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(k.kafkaConfig.Quota.Type))
	if factoryErr != nil {
		return "", errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unable to check quota")
	}
	subscriptionId, err = quotaService.ReserveQuota(kafkaRequest)
	return subscriptionId, err
}

// RegisterKafkaJob registers a new job in the kafka table.
// Before accepting the Kafka, the following checks are performed:
// That the user has quota to create the requested instance type. If not the Kafka registration is rejected.
// That the region limits have not been reached. If yes, then the Kafka registration is rejected.
// If region limits have not been reached and if the scaling mode is dynamic scaling, then the kafka registration is accepted.
// This means that kafka will be assigned to the data plane cluster when there is one available in the reconciliation step.
// If region limits have not been reached and if the scaling mode is manual, then we check if there is a cluster that has capacity left
// to accomodate this Kafka. If so, the registration of the kafka is accepted. Otherwise, it is rejected.
func (k *kafkaService) RegisterKafkaJob(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
	k.mu.Lock()
	defer k.mu.Unlock()
	// we need to pre-populate the ID to be able to reserve the quota
	kafkaRequest.ID = api.NewID()

	// The Instance Type determines the MultiAZ attribute. The previously value
	// set for the MultiAZ attribute in the request (if any) is ignored.
	// TODO improve this
	switch kafkaRequest.InstanceType {
	case types.STANDARD.String():
		kafkaRequest.MultiAZ = true
	case types.DEVELOPER.String():
		kafkaRequest.MultiAZ = false
	}

	// check for region capacity availability only if the kafka is not enterprise Kafka
	if !kafkaRequest.DesiredBillingModelIsEnterprise() {
		hasCapacity, err := k.HasAvailableCapacityInRegion(kafkaRequest)
		if err != nil {
			if err.Code == errors.ErrorGeneral {
				err = errors.NewWithCause(errors.ErrorGeneral, err, "unable to validate your request, please try again")
				logger.Logger.Errorf(err.Reason)
			}
			return err
		}
		if !hasCapacity {
			errorMsg := fmt.Sprintf("capacity exhausted in '%s' region for '%s' instance type", kafkaRequest.Region, kafkaRequest.InstanceType)
			logger.Logger.Warningf(errorMsg)
			return errors.TooManyKafkaInstancesReached(fmt.Sprintf("region %s cannot accept instance type: %s at this moment", kafkaRequest.Region, kafkaRequest.InstanceType))
		}
	}

	// assign the Kafka as soon as possible if we are not in dynamic scaling or that the Kafka is an enterprise Kafka
	if !k.dataplaneClusterConfig.IsDataPlaneAutoScalingEnabled() || kafkaRequest.DesiredBillingModelIsEnterprise() {
		cluster, err := k.findADataPlaneClusterToPlaceTheKafka(kafkaRequest)
		if err != nil {
			return err
		}

		kafkaRequest.ClusterID = cluster.ClusterID

		// assign cloud provider and region info of the cluster
		if kafkaRequest.DesiredBillingModelIsEnterprise() {
			kafkaRequest.CloudProvider = cluster.CloudProvider
			kafkaRequest.Region = cluster.Region
		}
	}

	subscriptionId, err := k.reserveQuota(kafkaRequest)

	if err != nil {
		return err
	}

	dbConn := k.connectionFactory.New()
	kafkaRequest.SubscriptionId = subscriptionId
	kafkaRequest.Status = constants.KafkaRequestStatusAccepted.String()

	// when creating new kafka - default storage size is assigned
	instanceType, instanceTypeErr := k.kafkaConfig.SupportedInstanceTypes.Configuration.GetKafkaInstanceTypeByID(kafkaRequest.InstanceType)
	if instanceTypeErr != nil {
		return errors.InstanceTypeNotSupported(instanceTypeErr.Error())
	}

	size, sizeErr := instanceType.GetKafkaInstanceSizeByID(kafkaRequest.SizeId)
	if sizeErr != nil {
		return errors.InstancePlanNotSupported(sizeErr.Error())
	}

	kafkaRequest.KafkaStorageSize = size.MaxDataRetentionSize.String()

	// We intentionally manually set CreatedAt and UpdatedAt instead of letting
	// gorm do it. The reason for that is that otherwise the ExpiresAt value
	// would be calculated from a start date potentially earlier than the CreatedAt
	// time a different value than those. An alternative would be performing
	// two different database updates but it would be less performant
	timeNow := dbConn.NowFunc()
	kafkaRequest.CreatedAt = timeNow
	kafkaRequest.UpdatedAt = timeNow
	if size.LifespanSeconds != nil {
		kafkaRequest.ExpiresAt = sql.NullTime{Time: timeNow.Add(time.Duration(*size.LifespanSeconds) * time.Second), Valid: true}
	}

	// Persist the QuotaTyoe to be able to dynamically pick the right Quota service implementation even on restarts.
	// A typical usecase is when a kafka A is created, at the time of creation the quota-type was ams. At some point in the future
	// the API is restarted this time changing the --quota-type flag to quota-management-list, when kafka A is deleted at this point,
	// we want to use the correct quota to perform the deletion.
	kafkaRequest.QuotaType = k.kafkaConfig.Quota.Type
	if err := dbConn.Create(kafkaRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to create kafka request") //hide the db error to http caller
	}

	metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants.KafkaRequestStatusAccepted, kafkaRequest.ID, kafkaRequest.ClusterID, time.Since(kafkaRequest.CreatedAt))

	return nil
}

func (k *kafkaService) findADataPlaneClusterToPlaceTheKafka(kafkaRequest *dbapi.KafkaRequest) (*api.Cluster, *errors.ServiceError) {
	cluster, e := k.clusterPlacementStrategy.FindCluster(kafkaRequest)
	if e != nil || cluster == nil {
		msg := fmt.Sprintf("no available cluster found for Kafka instance type '%s' in region '%s'", kafkaRequest.InstanceType, kafkaRequest.Region)
		if e != nil {
			logger.Logger.Error(fmt.Errorf("%s:%w", msg, e))
		} else {
			logger.Logger.Infof(msg)
		}

		userMsg := fmt.Sprintf("region %s cannot accept instance type: %s at this moment", kafkaRequest.Region, kafkaRequest.InstanceType)
		if kafkaRequest.DesiredBillingModelIsEnterprise() {
			userMsg = fmt.Sprintf("cluster %q cannot accept instance type: %q at this moment", kafkaRequest.ClusterID, kafkaRequest.InstanceType)
		}
		return nil, errors.TooManyKafkaInstancesReached(userMsg)
	}

	return cluster, nil
}

func (k *kafkaService) PrepareKafkaRequest(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
	kafkaRequest.Namespace = fmt.Sprintf("kafka-%s", strings.ToLower(kafkaRequest.ID))

	err := k.AssignBootstrapServerHost(kafkaRequest)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "error assigning bootstrap server host to kafka %s", kafkaRequest.ID)
	}

	if k.keycloakService.GetConfig().EnableAuthenticationOnKafka {
		clientId := strings.ToLower(fmt.Sprintf("%s-%s", CanaryServiceAccountPrefix, kafkaRequest.ID))
		serviceAccountRequest := sso.CompleteServiceAccountRequest{
			Owner:          kafkaRequest.Owner,
			OwnerAccountId: kafkaRequest.OwnerAccountId,
			ClientId:       clientId,
			OrgId:          kafkaRequest.OrganisationId,
			Name:           fmt.Sprintf("canary-service-account-for-kafka %s", kafkaRequest.ID),
			Description:    fmt.Sprintf("canary service account for kafka %s", kafkaRequest.ID),
		}

		canaryServiceAccount, err := k.keycloakService.CreateServiceAccountInternal(serviceAccountRequest)

		if err != nil {
			return errors.FailedToCreateSSOClient("failed to  create canary service account %s:%v", kafkaRequest.ID, err)
		}

		kafkaRequest.CanaryServiceAccountClientID = canaryServiceAccount.ClientID
		kafkaRequest.CanaryServiceAccountClientSecret = canaryServiceAccount.ClientSecret
	}

	// Update the Kafka Request record in the database
	// Only updates the fields below
	updatedKafkaRequest := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID: kafkaRequest.ID,
		},
		BootstrapServerHost:              kafkaRequest.BootstrapServerHost,
		CanaryServiceAccountClientID:     kafkaRequest.CanaryServiceAccountClientID,
		CanaryServiceAccountClientSecret: kafkaRequest.CanaryServiceAccountClientSecret,
		PlacementId:                      api.NewID(),
		Status:                           constants.KafkaRequestStatusProvisioning.String(),
		Namespace:                        kafkaRequest.Namespace,
	}
	if err := k.Update(updatedKafkaRequest); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update kafka request")
	}
	return nil
}

func (k *kafkaService) ListByStatus(status ...constants.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
	if len(status) == 0 {
		return nil, errors.GeneralError("no status provided")
	}
	dbConn := k.connectionFactory.New()

	var kafkas []*dbapi.KafkaRequest

	if err := dbConn.Model(&dbapi.KafkaRequest{}).Where("status IN (?)", status).Scan(&kafkas).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list by status")
	}

	return kafkas, nil
}

func (k *kafkaService) ListKafkasToBePromoted() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()

	validPromotionStatuses := []string{
		constants.KafkaRequestStatusReady.String(),
		constants.KafkaRequestStatusSuspended.String(),
		constants.KafkaRequestStatusResuming.String(),
	}

	var kafkas []*dbapi.KafkaRequest

	if err := dbConn.Model(&dbapi.KafkaRequest{}).
		Where("actual_kafka_billing_model <> desired_kafka_billing_model").
		Where("promotion_status <> ?", dbapi.KafkaPromotionStatusFailed).
		Where("status in ?", validPromotionStatuses).
		Scan(&kafkas).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list kafkas to be promoted")
	}

	return kafkas, nil
}

func (k *kafkaService) Get(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}

	dbConn := k.connectionFactory.New().Where("id = ?", id)

	var user string
	if !auth.GetIsAdminFromContext(ctx) {
		user, _ = claims.GetUsername()
		if user == "" {
			return nil, errors.Unauthenticated("user not authenticated")
		}

		orgId, _ := claims.GetOrgId()
		filterByOrganisationId := auth.GetFilterByOrganisationFromContext(ctx)

		// filter by organisationId if a user is part of an organisation and is not allowed as a service account
		if filterByOrganisationId {
			dbConn = dbConn.Where("organisation_id = ?", orgId)
		} else {
			dbConn = dbConn.Where("owner = ?", user)
		}
	}

	var kafkaRequest dbapi.KafkaRequest
	if err := dbConn.First(&kafkaRequest).Error; err != nil {
		resourceTypeStr := "KafkaResource"
		if user != "" {
			resourceTypeStr = fmt.Sprintf("%s for user %s", resourceTypeStr, user)
		}
		return nil, services.HandleGetError(resourceTypeStr, "id", id, err)
	}
	return &kafkaRequest, nil
}

func (k *kafkaService) GetByID(id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	dbConn := k.connectionFactory.New()
	var kafkaRequest dbapi.KafkaRequest
	if err := dbConn.Where("id = ?", id).First(&kafkaRequest).Error; err != nil {
		return nil, services.HandleGetError("KafkaResource", "id", id, err)
	}
	return &kafkaRequest, nil
}

// RegisterKafkaDeprovisionJob registers a kafka deprovision job in the kafka table
func (k *kafkaService) RegisterKafkaDeprovisionJob(ctx context.Context, id string) *errors.ServiceError {
	if id == "" {
		return errors.Validation("id is undefined")
	}

	// filter kafka request by owner to only retrieve request of the current authenticated user
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}

	dbConn := k.connectionFactory.New()

	if auth.GetIsAdminFromContext(ctx) {
		dbConn = dbConn.Where("id = ?", id)
	} else if claims.IsOrgAdmin() {
		orgId, _ := claims.GetOrgId()
		dbConn = dbConn.Where("id = ?", id).Where("organisation_id = ?", orgId)
	} else {
		user, _ := claims.GetUsername()
		dbConn = dbConn.Where("id = ?", id).Where("owner = ? ", user)
	}

	var kafkaRequest dbapi.KafkaRequest
	if err := dbConn.First(&kafkaRequest).Error; err != nil {
		return services.HandleGetError("KafkaResource", "id", id, err)
	}
	metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationDeprovision)

	deprovisionStatus := constants.KafkaRequestStatusDeprovision

	if executed, err := k.UpdateStatus(id, deprovisionStatus); executed {
		if err != nil {
			return services.HandleGetError("KafkaResource", "id", id, err)
		}
		metrics.IncreaseKafkaSuccessOperationsCountMetric(constants.KafkaOperationDeprovision)
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(deprovisionStatus, kafkaRequest.ID, kafkaRequest.ClusterID, time.Since(kafkaRequest.CreatedAt))
	}

	return nil
}

func (k *kafkaService) DeprovisionKafkaForUsers(users []string) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(&dbapi.KafkaRequest{}).
		Where("owner IN (?)", users).
		Where("status NOT IN (?)", kafkaDeletionStatuses).
		Update("status", constants.KafkaRequestStatusDeprovision)

	err := dbConn.Error
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "unable to deprovision kafka requests for users")
	}

	if dbConn.RowsAffected >= 1 {
		glog.Infof("%v kafkas are now deprovisioning for users %v", dbConn.RowsAffected, users)
		var counter int64 = 0
		for ; counter < dbConn.RowsAffected; counter++ {
			metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationDeprovision)
			metrics.IncreaseKafkaSuccessOperationsCountMetric(constants.KafkaOperationDeprovision)
		}
	}

	return nil
}

func (k *kafkaService) kafkaWithExpiresAtShouldBeDeprovisioned(kafkaRequest *dbapi.KafkaRequest, currentTime time.Time) bool {
	glog.V(10).Infof("Evaluating expiration time of kafka request '%s' with instance type '%s', size ID '%s' and status '%s'", kafkaRequest.ID, kafkaRequest.InstanceType, kafkaRequest.SizeId, kafkaRequest.Status)
	if currentTime.After(kafkaRequest.ExpiresAt.Time) {
		glog.V(10).Infof("Kafka ID '%s' has expired", kafkaRequest.ID)
		return true
	}

	glog.V(10).Infof("Kafka ID '%s' has not expired", kafkaRequest.ID)

	return false
}

func (k *kafkaService) DeprovisionExpiredKafkas() *errors.ServiceError {
	dbConn := k.connectionFactory.New().Model(&dbapi.KafkaRequest{}).Session(&gorm.Session{})

	var existingKafkaRequests []dbapi.KafkaRequest
	db := dbConn.Where("status NOT IN (?)", kafkaDeletionStatuses).
		Where("expires_at IS NOT NULL").
		Scan(&existingKafkaRequests)
	err := db.Error
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "unable to deprovision expired kafkas")
	}

	var kafkasToDeprovisionIDs []string
	timeNow := time.Now()

	for idx := range existingKafkaRequests {
		existingKafkaRequest := &existingKafkaRequests[idx]
		shouldBeDeprovisioned := k.kafkaWithExpiresAtShouldBeDeprovisioned(existingKafkaRequest, timeNow)
		if shouldBeDeprovisioned {
			kafkasToDeprovisionIDs = append(kafkasToDeprovisionIDs, existingKafkaRequest.ID)
		}
	}

	if len(kafkasToDeprovisionIDs) > 0 {
		glog.V(10).Infof("Kafka IDs to mark with status %s: %+v", constants.KafkaRequestStatusDeprovision, kafkasToDeprovisionIDs)
		db = dbConn.Where("id IN (?)", kafkasToDeprovisionIDs).
			Updates(map[string]interface{}{"status": constants.KafkaRequestStatusDeprovision})
		err = db.Error
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "unable to deprovision expired kafkas")
		}
		if db.RowsAffected >= 1 {
			glog.Infof("%v expired kafka_request's have had their status updated to deprovisioning", db.RowsAffected)
			var counter int64 = 0
			for ; counter < db.RowsAffected; counter++ {
				metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationDeprovision)
				metrics.IncreaseKafkaSuccessOperationsCountMetric(constants.KafkaOperationDeprovision)
			}
		}
	}

	return nil
}

func (k *kafkaService) Delete(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	// if the we don't have the clusterID we can only delete the row from the database
	if kafkaRequest.ClusterID != "" {
		// delete the kafka client in mas sso
		if k.keycloakService.GetConfig().EnableAuthenticationOnKafka {
			if kafkaRequest.CanaryServiceAccountClientID != "" {
				keycloakErr := k.keycloakService.DeleteServiceAccountInternal(kafkaRequest.CanaryServiceAccountClientID)

				if keycloakErr != nil {
					// Log the info for not found and proceed - not an error if service account is not found
					if keycloakErr.Code == errors.ErrorServiceAccountNotFound {
						glog.V(10).Infof("Service account with ID '%s' not found. Skipping deletion", kafkaRequest.CanaryServiceAccountClientID)
					} else {
						return errors.NewWithCause(errors.ErrorGeneral, keycloakErr, "error deleting canary service account")
					}
				}
			}
		}

		routes, err := kafkaRequest.GetRoutes()
		if err != nil {
			return errors.NewWithCause(errors.ErrorGeneral, err, "failed to get routes")
		}
		// Only delete the routes when they are set
		if routes != nil && k.kafkaConfig.EnableKafkaCNAMERegistration {
			_, err := k.ChangeKafkaCNAMErecords(kafkaRequest, KafkaRoutesActionDelete)
			if err != nil {
				return err
			}
		}
	}

	// soft delete the kafka request
	if err := dbConn.Delete(kafkaRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "unable to delete kafka request with id %s", kafkaRequest.ID)
	}

	metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationDelete)
	metrics.IncreaseKafkaSuccessOperationsCountMetric(constants.KafkaOperationDelete)

	return nil
}

// Lists all kafkas. As this returns all Kafka requests without need for authentication, this should only be used for internal purposes
func (k *kafkaService) ListAll() (dbapi.KafkaList, *errors.ServiceError) {
	var kafkaRequestList dbapi.KafkaList
	dbConn := k.connectionFactory.New()
	if err := dbConn.Find(&kafkaRequestList).Error; err != nil {
		return kafkaRequestList, errors.NewWithCause(errors.ErrorGeneral, err, "unable to list kafka requests")
	}

	return kafkaRequestList, nil
}

// List returns all Kafka requests belonging to a user.
func (k *kafkaService) List(ctx context.Context, listArgs *services.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError) {
	var kafkaRequestList dbapi.KafkaList
	dbConn := k.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, nil, errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}

	if !auth.GetIsAdminFromContext(ctx) {
		user, _ := claims.GetUsername()
		if user == "" {
			return nil, nil, errors.Unauthenticated("user not authenticated")
		}

		orgId, _ := claims.GetOrgId()
		filterByOrganisationId := auth.GetFilterByOrganisationFromContext(ctx)

		// filter by organisationId if a user is part of an organisation and is not allowed as a service account
		if filterByOrganisationId {
			// filter kafka requests by organisation_id since the user is allowed to see all kafka requests of my id
			dbConn = dbConn.Where("organisation_id = ?", orgId)
		} else {
			// filter kafka requests by owner as we are dealing with service accounts which may not have an org id
			dbConn = dbConn.Where("owner = ?", user)
		}
	}

	// Apply search query
	if len(listArgs.Search) > 0 {
		searchDbQuery, err := coreServices.NewQueryParser().Parse(listArgs.Search)
		if err != nil {
			return kafkaRequestList, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "unable to list kafka requests: %s", err.Error())
		}
		dbConn = dbConn.Where(searchDbQuery.Query, searchDbQuery.Values...)
	}

	if len(listArgs.OrderBy) == 0 {
		// default orderBy name
		dbConn = dbConn.Order("name")
	}

	// Set the order by arguments if any
	for _, orderByArg := range listArgs.OrderBy {
		dbConn = dbConn.Order(orderByArg)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	total := int64(pagingMeta.Total)
	dbConn.Model(&kafkaRequestList).Count(&total)
	pagingMeta.Total = int(total)
	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// execute query
	if err := dbConn.Find(&kafkaRequestList).Error; err != nil {
		return kafkaRequestList, pagingMeta, errors.NewWithCause(errors.ErrorGeneral, err, "unable to list kafka requests")
	}

	return kafkaRequestList, pagingMeta, nil
}

func (k *kafkaService) GetManagedKafkaByClusterID(clusterID string) ([]managedkafka.ManagedKafka, *errors.ServiceError) {
	dbConn := k.connectionFactory.New().
		Where("cluster_id = ?", clusterID).
		Where("status IN (?)", kafkaManagedCRStatuses).
		Where("bootstrap_server_host != ''")

	var kafkaRequestList dbapi.KafkaList
	if err := dbConn.Find(&kafkaRequestList).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "unable to list kafka requests")
	}

	var res []managedkafka.ManagedKafka
	// convert kafka requests to managed kafka
	for _, kafkaRequest := range kafkaRequestList {
		mk, err := buildManagedKafkaCR(kafkaRequest, k.kafkaConfig, k.keycloakService)
		if err != nil {
			return nil, err
		}
		res = append(res, *mk)
	}

	return res, nil
}

func (k *kafkaService) GenerateReservedManagedKafkasByClusterID(clusterID string) ([]managedkafka.ManagedKafka, *errors.ServiceError) {
	reservedKafkas := []managedkafka.ManagedKafka{}
	cluster, svcErr := k.clusterService.FindClusterByID(clusterID)
	if svcErr != nil {
		return nil, svcErr
	}
	if cluster == nil {
		return nil, errors.GeneralError("failed to generate reserved managed kafkas for clusterID %s: clusterID not found", clusterID)
	}
	if cluster.Status != api.ClusterReady {
		logger.Logger.V(10).Infof("ClusterID '%s' is not ready. Its status is '%s'. Returning an empty list of reserved managed kafkas", clusterID, cluster.Status)
		return reservedKafkas, nil
	}

	latestStrimziVersion, err := cluster.GetLatestAvailableAndReadyStrimziVersion()
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to generate reserved managed kafkas for clusterID %s: error finding ready strimzi versions", clusterID)
	}
	if latestStrimziVersion == nil {
		return nil, errors.GeneralError("failed to generate reserved managed kafkas for clusterID %s: no ready strimzi versions found", clusterID)
	}

	supportedInstanceTypes := cluster.GetSupportedInstanceTypes()

	for _, supportedInstanceType := range supportedInstanceTypes {
		instanceTypeDynamicScalingConfig, ok := k.dataplaneClusterConfig.NodePrewarmingConfig.ForInstanceType(supportedInstanceType)
		if !ok {
			continue
		}
		numReservedInstances := instanceTypeDynamicScalingConfig.NumReservedInstances
		for i := 1; i <= numReservedInstances; i++ {
			generatedKafkaID := fmt.Sprintf("reserved-kafka-%s-%d", supportedInstanceType, i)
			res, err := k.buildReservedManagedKafkaCR(generatedKafkaID, supportedInstanceType, instanceTypeDynamicScalingConfig.BaseStreamingUnitSize, *latestStrimziVersion)
			if err != nil {
				return nil, err
			}

			reservedKafkas = append(reservedKafkas, *res)
		}

	}

	return reservedKafkas, nil
}

func (k *kafkaService) Update(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(kafkaRequest).
		Where("status not IN (?)", kafkaDeletionStatuses) // ignore updates of kafka under deletion

	if err := dbConn.Updates(kafkaRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update kafka")
	}

	return nil
}

func (k *kafkaService) Updates(kafkaRequest *dbapi.KafkaRequest, fields map[string]interface{}) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(kafkaRequest).
		Where("status not IN (?)", kafkaDeletionStatuses) // ignore updates of kafka under deletion

	if err := dbConn.Updates(fields).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update kafka")
	}

	return nil
}

func (k *kafkaService) VerifyAndUpdateKafkaAdmin(ctx context.Context, kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
	if !auth.GetIsAdminFromContext(ctx) {
		return errors.New(errors.ErrorUnauthenticated, "user not authenticated")
	}

	// only updated specified columns to avoid changing other columns e.g Status
	updatableFields := map[string]interface{}{
		"kafka_storage_size":        kafkaRequest.KafkaStorageSize,
		"desired_strimzi_version":   kafkaRequest.DesiredStrimziVersion,
		"desired_kafka_version":     kafkaRequest.DesiredKafkaVersion,
		"desired_kafka_ibp_version": kafkaRequest.DesiredKafkaIBPVersion,
		"status":                    kafkaRequest.Status,
	}

	dbConn := k.connectionFactory.New().
		Model(kafkaRequest)

	if err := dbConn.Updates(updatableFields).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update kafka")
	}

	return nil
}

func (k *kafkaService) UpdateStatus(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()

	if kafka, err := k.GetByID(id); err != nil {
		return true, errors.NewWithCause(errors.ErrorGeneral, err, "failed to update status")
	} else {
		// only allow to change the status to "deleting" if the cluster is already in "deprovision" status
		if kafka.Status == constants.KafkaRequestStatusDeprovision.String() && status != constants.KafkaRequestStatusDeleting {
			return false, errors.GeneralError("failed to update status: cluster is deprovisioning")
		}

		if kafka.Status == status.String() {
			// no update needed
			return false, errors.GeneralError("failed to update status: the cluster %s is already in %s state", id, status.String())
		}
	}

	if err := dbConn.Model(&dbapi.KafkaRequest{Meta: api.Meta{ID: id}}).Update("status", status).Error; err != nil {
		return true, errors.NewWithCause(errors.ErrorGeneral, err, "failed to update kafka status")
	}

	return true, nil
}

func (k *kafkaService) ChangeKafkaCNAMErecords(kafkaRequest *dbapi.KafkaRequest, action KafkaRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
	routes, err := kafkaRequest.GetRoutes()
	if routes == nil || err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to get routes")
	}

	domainRecordBatch := buildKafkaClusterCNAMESRecordBatch(routes, action)

	awsConfig := aws.Config{
		AccessKeyID:     k.awsConfig.Route53AccessKey,
		SecretAccessKey: k.awsConfig.Route53SecretAccessKey,
	}

	route53Region, err := k.getRoute53RegionFromKafkaRequest(kafkaRequest)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "error getting route 53 region from kafka request")
	}

	awsClient, err := k.awsClientFactory.NewClient(awsConfig, route53Region)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "unable to create aws client")
	}

	changeRecordsOutput, err := awsClient.ChangeResourceRecordSets(k.kafkaConfig.KafkaDomainName, domainRecordBatch)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "unable to create domain record sets")
	}

	return changeRecordsOutput, nil
}

func (k *kafkaService) GetCNAMERecordStatus(kafkaRequest *dbapi.KafkaRequest) (*CNameRecordStatus, error) {
	awsConfig := aws.Config{
		AccessKeyID:     k.awsConfig.Route53AccessKey,
		SecretAccessKey: k.awsConfig.Route53SecretAccessKey,
	}

	route53Region, err := k.getRoute53RegionFromKafkaRequest(kafkaRequest)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "error getting route 53 region from kafka request")
	}

	awsClient, err := k.awsClientFactory.NewClient(awsConfig, route53Region)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "unable to create aws client")
	}

	changeOutput, err := awsClient.GetChange(kafkaRequest.RoutesCreationId)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "unable to get status of Route53 change batch request with ID %q", kafkaRequest.RoutesCreationId)
	}

	return &CNameRecordStatus{
		Id:     changeOutput.ChangeInfo.Id,
		Status: changeOutput.ChangeInfo.Status,
	}, nil
}

type KafkaStatusCount struct {
	Status constants.KafkaStatus
	Count  int
}

func (k *kafkaService) CountByStatus(status []constants.KafkaStatus) ([]KafkaStatusCount, error) {
	dbConn := k.connectionFactory.New()
	var results []KafkaStatusCount
	if err := dbConn.Model(&dbapi.KafkaRequest{}).Select("status as Status, count(1) as Count").Where("status in (?)", status).Group("status").Scan(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to count kafkas")
	}

	// if there is no count returned for a status from the above query because there is no kafkas in such a status,
	// we should return the count for these as well to avoid any confusion
	if len(status) > 0 {
		countersMap := map[constants.KafkaStatus]int{}
		for _, r := range results {
			countersMap[r.Status] = r.Count
		}
		for _, s := range status {
			if _, ok := countersMap[s]; !ok {
				results = append(results, KafkaStatusCount{Status: s, Count: 0})
			}
		}
	}

	return results, nil
}

type KafkaComponentVersions struct {
	ID                     string
	ClusterID              string
	DesiredStrimziVersion  string
	ActualStrimziVersion   string
	StrimziUpgrading       bool
	DesiredKafkaVersion    string
	ActualKafkaVersion     string
	KafkaUpgrading         bool
	DesiredKafkaIBPVersion string
	ActualKafkaIBPVersion  string
	KafkaIBPUpgrading      bool
}

func (k *kafkaService) ListComponentVersions() ([]KafkaComponentVersions, error) {
	dbConn := k.connectionFactory.New()
	var results []KafkaComponentVersions
	if err := dbConn.Model(&dbapi.KafkaRequest{}).Select("id", "cluster_id", "desired_strimzi_version", "actual_strimzi_version", "strimzi_upgrading", "desired_kafka_version", "actual_kafka_version", "kafka_upgrading", "desired_kafka_ibp_version", "actual_kafka_ibp_version", "kafka_ibp_upgrading").Scan(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list component versions")
	}
	return results, nil
}

func (k *kafkaService) ListKafkasWithRoutesNotCreated() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var results []*dbapi.KafkaRequest
	if err := dbConn.Where("routes IS NOT NULL").Where("routes_created = ?", "no").Find(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list kafka requests")
	}
	return results, nil
}

func buildManagedKafkaCR(kafkaRequest *dbapi.KafkaRequest, kafkaConfig *config.KafkaConfig, keycloakService sso.KeycloakService) (*managedkafka.ManagedKafka, *errors.ServiceError) {
	k, err := kafkaConfig.GetKafkaInstanceSize(kafkaRequest.InstanceType, kafkaRequest.SizeId)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "unable to list kafka request")
	}

	labels := map[string]string{
		"bf2.org/kafkaInstanceProfileQuotaConsumed":   strconv.Itoa(k.QuotaConsumed),
		"bf2.org/kafkaInstanceProfileType":            kafkaRequest.InstanceType,
		managedkafka.ManagedKafkaBf2SuspendedLabelKey: fmt.Sprintf("%t", arrays.Contains(constants.GetSuspendedStatuses(), kafkaRequest.Status)),
	}

	managedKafkaCR := &managedkafka.ManagedKafka{
		Id: kafkaRequest.ID,
		TypeMeta: metav1.TypeMeta{
			Kind:       "ManagedKafka",
			APIVersion: "managedkafka.bf2.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaRequest.Name,
			Namespace: kafkaRequest.Namespace,
			Annotations: map[string]string{
				"bf2.org/id":          kafkaRequest.ID,
				"bf2.org/placementId": kafkaRequest.PlacementId,
			},
			Labels: labels,
		},
		Spec: managedkafka.ManagedKafkaSpec{
			Capacity: managedkafka.Capacity{
				IngressPerSec:               k.IngressThroughputPerSec.String(),
				EgressPerSec:                k.EgressThroughputPerSec.String(),
				TotalMaxConnections:         k.TotalMaxConnections,
				MaxDataRetentionSize:        kafkaRequest.KafkaStorageSize,
				MaxPartitions:               k.MaxPartitions,
				MaxDataRetentionPeriod:      k.MaxDataRetentionPeriod,
				MaxConnectionAttemptsPerSec: k.MaxConnectionAttemptsPerSec,
			},
			Endpoint: managedkafka.EndpointSpec{
				BootstrapServerHost: kafkaRequest.BootstrapServerHost,
			},
			Versions: managedkafka.VersionsSpec{
				Kafka:    kafkaRequest.DesiredKafkaVersion,
				Strimzi:  kafkaRequest.DesiredStrimziVersion,
				KafkaIBP: kafkaRequest.DesiredKafkaIBPVersion,
			},
			Deleted: kafkaRequest.Status == constants.KafkaRequestStatusDeprovision.String(),
			Owners:  buildKafkaOwner(kafkaRequest, kafkaConfig),
		},
		Status: managedkafka.ManagedKafkaStatus{},
	}

	keycloakConfig := keycloakService.GetConfig()
	keycloakRealmConfig := keycloakService.GetRealmConfig()

	if keycloakConfig.EnableAuthenticationOnKafka {
		managedKafkaCR.Spec.OAuth = managedkafka.OAuthSpec{
			TokenEndpointURI:       keycloakRealmConfig.TokenEndpointURI,
			JwksEndpointURI:        keycloakRealmConfig.JwksEndpointURI,
			ValidIssuerEndpointURI: keycloakRealmConfig.ValidIssuerURI,
			UserNameClaim:          keycloakConfig.UserNameClaim,
			FallBackUserNameClaim:  keycloakConfig.FallBackUserNameClaim,
			CustomClaimCheck:       BuildCustomClaimCheck(kafkaRequest, keycloakConfig.SelectSSOProvider),
			MaximumSessionLifetime: 0,
		}

		if keycloakConfig.TLSTrustedCertificatesValue != "" {
			managedKafkaCR.Spec.OAuth.TlsTrustedCertificate = &keycloakConfig.TLSTrustedCertificatesValue
		}

		if kafkaRequest.ReauthenticationEnabled {
			managedKafkaCR.Spec.OAuth.MaximumSessionLifetime = 299000 // 4m59s
		}

		serviceAccounts := []managedkafka.ServiceAccount{}
		serviceAccounts = append(serviceAccounts, managedkafka.ServiceAccount{
			Name:      "canary",
			Principal: kafkaRequest.CanaryServiceAccountClientID,
			Password:  kafkaRequest.CanaryServiceAccountClientSecret,
		})
		managedKafkaCR.Spec.ServiceAccounts = serviceAccounts
	}

	if kafkaConfig.EnableKafkaExternalCertificate {
		managedKafkaCR.Spec.Endpoint.Tls = &managedkafka.TlsSpec{
			Cert: kafkaConfig.KafkaTLSCert,
			Key:  kafkaConfig.KafkaTLSKey,
		}
	}

	return managedKafkaCR, nil
}

// buildReservedManagedKafkaCR builds a Reserved Managed Kafka CR.
// The ID, K8s object ID, K8s namespace and PlacementID are all set to
// the provided kafkaID.
func (k *kafkaService) buildReservedManagedKafkaCR(kafkaID, instanceType, streamingBaseUnit string, desiredStrimziVersion api.StrimziVersion) (*managedkafka.ManagedKafka, *errors.ServiceError) {
	kafkaInstanceSize, err := k.kafkaConfig.GetKafkaInstanceSize(instanceType, streamingBaseUnit)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "unable to list kafka request")
	}
	labels := map[string]string{
		"bf2.org/kafkaInstanceProfileQuotaConsumed":    strconv.Itoa(kafkaInstanceSize.QuotaConsumed),
		"bf2.org/kafkaInstanceProfileType":             instanceType,
		managedkafka.ManagedKafkaBf2DeploymentLabelKey: managedkafka.ManagedKafkaBf2DeploymentLabelValueReserved,
	}

	desiredKafkaVersion := desiredStrimziVersion.GetLatestKafkaVersion()
	if desiredKafkaVersion == nil {
		return nil, errors.GeneralError("no available Kafka versions in Strimzi version %s", desiredStrimziVersion.Version)
	}
	desiredKafkaIBPVersion := desiredStrimziVersion.GetLatestKafkaIBPVersion()
	if desiredKafkaIBPVersion == nil {
		return nil, errors.GeneralError("no available Kafka IBP versions in Strimzi version %s", desiredStrimziVersion.Version)
	}

	managedKafkaCR := &managedkafka.ManagedKafka{
		Id: kafkaID,
		TypeMeta: metav1.TypeMeta{
			Kind:       "ManagedKafka",
			APIVersion: "managedkafka.bf2.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaID,
			Namespace: kafkaID,
			Annotations: map[string]string{
				"bf2.org/id":          kafkaID,
				"bf2.org/placementId": kafkaID,
			},
			Labels: labels,
		},
		Spec: managedkafka.ManagedKafkaSpec{
			Capacity: managedkafka.Capacity{
				IngressPerSec:               kafkaInstanceSize.IngressThroughputPerSec.String(),
				EgressPerSec:                kafkaInstanceSize.EgressThroughputPerSec.String(),
				TotalMaxConnections:         kafkaInstanceSize.TotalMaxConnections,
				MaxDataRetentionSize:        kafkaInstanceSize.MaxDataRetentionSize.String(),
				MaxPartitions:               kafkaInstanceSize.MaxPartitions,
				MaxDataRetentionPeriod:      kafkaInstanceSize.MaxDataRetentionPeriod,
				MaxConnectionAttemptsPerSec: kafkaInstanceSize.MaxConnectionAttemptsPerSec,
			},
			Endpoint: managedkafka.EndpointSpec{
				BootstrapServerHost: fmt.Sprintf("%s-dummyhost", kafkaID),
			},
			Versions: managedkafka.VersionsSpec{
				Kafka:    desiredKafkaVersion.Version,
				Strimzi:  desiredStrimziVersion.Version,
				KafkaIBP: desiredKafkaIBPVersion.Version,
			},
			Deleted: false,
			Owners:  []string{}, // TODO is this enough?
		},
		Status: managedkafka.ManagedKafkaStatus{},
	}
	return managedKafkaCR, nil
}

func buildKafkaOwner(kafkaRequest *dbapi.KafkaRequest, kafkaConfig *config.KafkaConfig) []string {
	if kafkaConfig.EnableKafkaOwnerConfig {
		return append([]string{kafkaRequest.Owner}, kafkaConfig.KafkaOwnerList...)
	}
	return []string{
		kafkaRequest.Owner,
	}
}

func buildKafkaClusterCNAMESRecordBatch(routes []dbapi.DataPlaneKafkaRoute, action KafkaRoutesAction) *route53.ChangeBatch {
	var changes []*route53.Change
	for _, r := range routes {
		c := buildResourceRecordChange(r.Domain, r.Router, action)
		changes = append(changes, c)
	}
	recordChangeBatch := &route53.ChangeBatch{
		Changes: changes,
	}

	return recordChangeBatch
}

func buildResourceRecordChange(recordName string, clusterIngress string, action KafkaRoutesAction) *route53.Change {
	recordType := "CNAME"
	recordTTL := int64(300)

	actionStr := action.String()
	resourceRecordChange := &route53.Change{
		Action: &actionStr,
		ResourceRecordSet: &route53.ResourceRecordSet{
			Name: &recordName,
			Type: &recordType,
			TTL:  &recordTTL,
			ResourceRecords: []*route53.ResourceRecord{
				{
					Value: &clusterIngress,
				},
			},
		},
	}

	return resourceRecordChange
}

func (k *kafkaService) AssignBootstrapServerHost(kafkaRequest *dbapi.KafkaRequest) error {
	truncatedKafkaIdentifier := buildTruncateKafkaIdentifier(kafkaRequest)
	truncatedKafkaIdentifier, replaceErr := replaceHostSpecialChar(truncatedKafkaIdentifier)
	if replaceErr != nil {
		return errors.NewWithCause(errors.ErrorGeneral, replaceErr, "generated host is not valid")
	}

	clusterDNS, err := k.clusterService.GetClusterDNS(kafkaRequest.ClusterID)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "error retrieving cluster DNS")
	}

	clusterDNS = strings.Replace(clusterDNS, constants.DefaultIngressDnsNamePrefix, constants.ManagedKafkaIngressDnsNamePrefix, 1)

	if k.kafkaConfig.EnableKafkaCNAMERegistration {
		// If we enable KafkaTLS, the bootstrapServerHost should use the external domain name rather than the cluster domain
		kafkaRequest.BootstrapServerHost = fmt.Sprintf("%s.%s", truncatedKafkaIdentifier, k.kafkaConfig.KafkaDomainName)
	} else {
		kafkaRequest.BootstrapServerHost = fmt.Sprintf("%s.%s", truncatedKafkaIdentifier, clusterDNS)
	}

	return nil
}

// getRoute53RegionFromKafkaRequest calculates the AWS region to be used for
// Route53 from the kafka request. It calculates its value using the
// cloud provider specified in the kafka request.
// Route53 is a global service which means that in most of the cases
// the region specified is only used to access a regional endpoint in AWS.
// There are some parts of the Route53 functionality that are regional.
// For what we perform which is create hosted zones and entries in them
// that is a global functionality.
// See: https://docs.aws.amazon.com/Route53/latest/DeveloperGuide/disaster-recovery-resiliency.html
// If at some point we end up needing Route53 regional functionalities this
// mechanism should be reevaluated
func (k *kafkaService) getRoute53RegionFromKafkaRequest(kafkaRequest *dbapi.KafkaRequest) (string, error) {
	switch kafkaRequest.CloudProvider {
	case cloudproviders.AWS.String():
		return aws.DefaultAWSRoute53Region, nil
	case cloudproviders.GCP.String():
		return aws.DefaultGCPRoute53Region, nil
	default:
		return "", errors.GeneralError("unknown cloud provider: %q", kafkaRequest.CloudProvider)
	}
}

func (k *kafkaService) UpdateZeroValueOfKafkaRequestsExpiredAt() error {
	dbConn := k.connectionFactory.New()
	zeroTime := time.Time{}
	nullTime := sql.NullTime{Time: time.Time{}, Valid: false}
	db := dbConn.Table("kafka_requests").Where("expires_at = ?", zeroTime).Where("deleted_at IS NULL").Update("expires_at", nullTime)
	glog.Infof("%v kafka_requests had expires_at with the zero-value of time.Time '%v' and have been updated to NULL", db.RowsAffected, zeroTime)

	return db.Error
}

func (k *kafkaService) IsQuotaEntitlementActive(kafkaRequest *dbapi.KafkaRequest) (bool, error) {
	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(k.kafkaConfig.Quota.Type))
	if factoryErr != nil {
		return false, factoryErr
	}

	return quotaService.IsQuotaEntitlementActive(kafkaRequest)
}
