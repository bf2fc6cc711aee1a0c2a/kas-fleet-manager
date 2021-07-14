package services

import (
	"context"
	"fmt"
	"strings"
	"sync"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"

	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/queryparser"

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
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
)

var kafkaDeletionStatuses = []string{constants2.KafkaRequestStatusDeleting.String(), constants2.KafkaRequestStatusDeprovision.String()}
var kafkaManagedCRStatuses = []string{constants2.KafkaRequestStatusProvisioning.String(), constants2.KafkaRequestStatusDeprovision.String(), constants2.KafkaRequestStatusReady.String(), constants2.KafkaRequestStatusFailed.String()}

type KafkaRoutesAction string

const KafkaRoutesActionCreate KafkaRoutesAction = "CREATE"
const KafkaRoutesActionDelete KafkaRoutesAction = "DELETE"

//go:generate moq -out kafkaservice_moq.go . KafkaService
type KafkaService interface {
	HasAvailableCapacity() (bool, *errors.ServiceError)
	// PrepareKafkaRequest sets any required information (i.e. bootstrap server host, sso client id and secret)
	// to the Kafka Request record in the database. The kafka request will also be updated with an updated_at
	// timestamp and the corresponding cluster identifier.
	PrepareKafkaRequest(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError
	// Get method will retrieve the kafkaRequest instance that the give ctx has access to from the database.
	// This should be used when you want to make sure the result is filtered based on the request context.
	Get(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError)
	// GetById method will retrieve the KafkaRequest instance from the database without checking any permissions.
	// You should only use this if you are sure permission check is not required.
	GetById(id string) (*dbapi.KafkaRequest, *errors.ServiceError)
	// Delete cleans up all dependencies for a Kafka request and soft deletes the Kafka Request record from the database.
	// The Kafka Request in the database will be updated with a deleted_at timestamp.
	Delete(*dbapi.KafkaRequest) *errors.ServiceError
	List(ctx context.Context, listArgs *services.ListArguments) (dbapi.KafkaList, *api.PagingMeta, *errors.ServiceError)
	GetManagedKafkaByClusterID(clusterID string) ([]managedkafka.ManagedKafka, *errors.ServiceError)
	RegisterKafkaJob(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError
	ListByStatus(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError)
	// UpdateStatus change the status of the Kafka cluster
	// The returned boolean is to be used to know if the update has been tried or not. An update is not tried if the
	// original status is 'deprovision' (cluster in deprovision state can't be change state) or if the final status is the
	// same as the original status. The error will contain any error encountered when attempting to update or the reason
	// why no attempt has been done
	UpdateStatus(id string, status constants2.KafkaStatus) (bool, *errors.ServiceError)
	Update(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError
	ChangeKafkaCNAMErecords(kafkaRequest *dbapi.KafkaRequest, action KafkaRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError)
	RegisterKafkaDeprovisionJob(ctx context.Context, id string) *errors.ServiceError
	// DeprovisionKafkaForUsers registers all kafkas for deprovisioning given the list of owners
	DeprovisionKafkaForUsers(users []string) *errors.ServiceError
	DeprovisionExpiredKafkas(kafkaAgeInHours int) *errors.ServiceError
	CountByStatus(status []constants2.KafkaStatus) ([]KafkaStatusCount, error)
	ListKafkasWithRoutesNotCreated() ([]*dbapi.KafkaRequest, *errors.ServiceError)
}

var _ KafkaService = &kafkaService{}

type kafkaService struct {
	connectionFactory   *db.ConnectionFactory
	clusterService      ClusterService
	keycloakService     services.KeycloakService
	kafkaConfig         *config.KafkaConfig
	awsConfig           *config.AWSConfig
	quotaServiceFactory QuotaServiceFactory
	mu                  sync.Mutex
	awsClientFactory    aws.ClientFactory
}

func NewKafkaService(connectionFactory *db.ConnectionFactory, clusterService ClusterService, keycloakService services.KafkaKeycloakService, kafkaConfig *config.KafkaConfig, awsConfig *config.AWSConfig, quotaServiceFactory QuotaServiceFactory, awsClientFactory aws.ClientFactory) *kafkaService {
	return &kafkaService{
		connectionFactory:   connectionFactory,
		clusterService:      clusterService,
		keycloakService:     keycloakService,
		kafkaConfig:         kafkaConfig,
		awsConfig:           awsConfig,
		quotaServiceFactory: quotaServiceFactory,
		awsClientFactory:    awsClientFactory,
	}
}

func (k *kafkaService) HasAvailableCapacity() (bool, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var count int64

	if err := dbConn.Model(&dbapi.KafkaRequest{}).Count(&count).Error; err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, "failed to count kafka request")
	}

	glog.Infof("%d of %d kafka clusters currently instantiated", count, k.kafkaConfig.KafkaCapacity.MaxCapacity)
	return count < k.kafkaConfig.KafkaCapacity.MaxCapacity, nil
}

// RegisterKafkaJob registers a new job in the kafka table
func (k *kafkaService) RegisterKafkaJob(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
	k.mu.Lock()
	defer k.mu.Unlock()
	if hasCapacity, err := k.HasAvailableCapacity(); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to create kafka request")
	} else if !hasCapacity {
		logger.Logger.Warningf("Cluster capacity(%d) exhausted", k.kafkaConfig.KafkaCapacity.MaxCapacity)
		return errors.TooManyKafkaInstancesReached("cluster capacity exhausted")
	}
	//cluster id can't be nil. generating random temporary id.
	//reserve is false, checking whether a user can reserve a quota or not
	quotaService, factoryErr := k.quotaServiceFactory.GetQuotaService(api.QuotaType(k.kafkaConfig.Quota.Type))
	if factoryErr != nil {
		return errors.NewWithCause(errors.ErrorGeneral, factoryErr, "unabled to check quota")
	}
	err := quotaService.CheckQuota(kafkaRequest)
	if err != nil {
		return err
	}

	dbConn := k.connectionFactory.New()
	kafkaRequest.DesiredKafkaVersion = k.kafkaConfig.DefaultKafkaVersion
	kafkaRequest.Status = constants2.KafkaRequestStatusAccepted.String()

	// Persist the QuotaTyoe to be able to dynamically pick the right Quota service implementation even on restarts.
	// A typical usecase is when a kafka A is created, at the time of creation the quota-type was ams. At some point in the future
	// the API is restarted this time changing the --quota-type flag to allow-list, when kafka A is deleted at this point,
	// we want to use the correct quota to perform the deletion.
	kafkaRequest.QuotaType = k.kafkaConfig.Quota.Type
	if err := dbConn.Save(kafkaRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to create kafka request") //hide the db error to http caller
	}
	metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(constants2.KafkaRequestStatusAccepted, kafkaRequest.ID, kafkaRequest.ClusterID, time.Since(kafkaRequest.CreatedAt))
	return nil
}

func (k *kafkaService) PrepareKafkaRequest(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
	truncatedKafkaIdentifier := buildTruncateKafkaIdentifier(kafkaRequest)
	truncatedKafkaIdentifier, replaceErr := replaceHostSpecialChar(truncatedKafkaIdentifier)
	if replaceErr != nil {
		return errors.NewWithCause(errors.ErrorGeneral, replaceErr, "generated host is not valid")
	}

	clusterDNS, err := k.clusterService.GetClusterDNS(kafkaRequest.ClusterID)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "error retrieving cluster DNS")
	}

	clusterDNS = strings.Replace(clusterDNS, constants2.DefaultIngressDnsNamePrefix, constants2.ManagedKafkaIngressDnsNamePrefix, 1)
	kafkaRequest.BootstrapServerHost = fmt.Sprintf("%s.%s", truncatedKafkaIdentifier, clusterDNS)

	if k.kafkaConfig.EnableKafkaExternalCertificate {
		// If we enable KafkaTLS, the bootstrapServerHost should use the external domain name rather than the cluster domain
		kafkaRequest.BootstrapServerHost = fmt.Sprintf("%s.%s", truncatedKafkaIdentifier, k.kafkaConfig.KafkaDomainName)
	}

	if k.keycloakService.GetConfig().EnableAuthenticationOnKafka {
		kafkaRequest.SsoClientID = BuildKeycloakClientNameIdentifier(kafkaRequest.ID)
		kafkaRequest.SsoClientSecret, err = k.keycloakService.RegisterKafkaClientInSSO(kafkaRequest.SsoClientID, kafkaRequest.OrganisationId)
		if err != nil {
			return errors.FailedToCreateSSOClient("failed to create sso client %s:%v", kafkaRequest.SsoClientID, err)
		}
	}

	// Update the Kafka Request record in the database
	// Only updates the fields below
	updatedKafkaRequest := &dbapi.KafkaRequest{
		Meta: api.Meta{
			ID: kafkaRequest.ID,
		},
		BootstrapServerHost: kafkaRequest.BootstrapServerHost,
		SsoClientID:         kafkaRequest.SsoClientID,
		SsoClientSecret:     kafkaRequest.SsoClientSecret,
		PlacementId:         api.NewID(),
		Status:              constants2.KafkaRequestStatusProvisioning.String(),
	}
	if err := k.Update(updatedKafkaRequest); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update kafka request")
	}

	return nil
}

func (k *kafkaService) ListByStatus(status ...constants2.KafkaStatus) ([]*dbapi.KafkaRequest, *errors.ServiceError) {
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

func (k *kafkaService) Get(ctx context.Context, id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorUnauthenticated, err, "user not authenticated")
	}

	user := auth.GetUsernameFromClaims(claims)
	if user == "" {
		return nil, errors.Unauthenticated("user not authenticated")
	}

	orgId := auth.GetOrgIdFromClaims(claims)
	filterByOrganisationId := auth.GetFilterByOrganisationFromContext(ctx)

	dbConn := k.connectionFactory.New().Where("id = ?", id)

	// filter by organisationId if a user is part of an organisation and is not allowed as a service account
	if filterByOrganisationId {
		dbConn = dbConn.Where("organisation_id = ?", orgId)
	} else {
		dbConn = dbConn.Where("owner = ?", user)
	}

	var kafkaRequest dbapi.KafkaRequest
	if err := dbConn.First(&kafkaRequest).Error; err != nil {
		return nil, services.HandleGetError("KafkaResource for user "+user, "id", id, err)
	}
	return &kafkaRequest, nil
}

func (k *kafkaService) GetById(id string) (*dbapi.KafkaRequest, *errors.ServiceError) {
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
	} else if auth.GetIsOrgAdminFromClaims(claims) {
		orgId := auth.GetOrgIdFromClaims(claims)
		dbConn = dbConn.Where("id = ?", id).Where("organisation_id = ?", orgId)
	} else {
		user := auth.GetUsernameFromClaims(claims)
		dbConn = dbConn.Where("id = ?", id).Where("owner = ? ", user)
	}

	var kafkaRequest dbapi.KafkaRequest
	if err := dbConn.First(&kafkaRequest).Error; err != nil {
		return services.HandleGetError("KafkaResource", "id", id, err)
	}
	metrics.IncreaseKafkaTotalOperationsCountMetric(constants2.KafkaOperationDeprovision)

	deprovisionStatus := constants2.KafkaRequestStatusDeprovision

	if executed, err := k.UpdateStatus(id, deprovisionStatus); executed {
		if err != nil {
			return services.HandleGetError("KafkaResource", "id", id, err)
		}
		metrics.IncreaseKafkaSuccessOperationsCountMetric(constants2.KafkaOperationDeprovision)
		metrics.UpdateKafkaRequestsStatusSinceCreatedMetric(deprovisionStatus, kafkaRequest.ID, kafkaRequest.ClusterID, time.Since(kafkaRequest.CreatedAt))
	}

	return nil
}

func (k *kafkaService) DeprovisionKafkaForUsers(users []string) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(&dbapi.KafkaRequest{}).
		Where("owner IN (?)", users).
		Where("status NOT IN (?)", kafkaDeletionStatuses).
		Update("status", constants2.KafkaRequestStatusDeprovision)

	err := dbConn.Error
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Unable to deprovision kafka requests for users")
	}

	if dbConn.RowsAffected >= 1 {
		glog.Infof("%v kafkas are now deprovisioning for users %v", dbConn.RowsAffected, users)
		var counter int64 = 0
		for ; counter < dbConn.RowsAffected; counter++ {
			metrics.IncreaseKafkaTotalOperationsCountMetric(constants2.KafkaOperationDeprovision)
			metrics.IncreaseKafkaSuccessOperationsCountMetric(constants2.KafkaOperationDeprovision)
		}
	}

	return nil
}

func (k *kafkaService) DeprovisionExpiredKafkas(kafkaAgeInHours int) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(&dbapi.KafkaRequest{}).
		Where("created_at  <=  ?", time.Now().Add(-1*time.Duration(kafkaAgeInHours)*time.Hour)).
		Where("status NOT IN (?)", kafkaDeletionStatuses)

	if len(k.kafkaConfig.KafkaLifespan.LongLivedKafkas) > 0 {
		dbConn = dbConn.Where("id NOT IN (?)", k.kafkaConfig.KafkaLifespan.LongLivedKafkas)
	}

	db := dbConn.Update("status", constants2.KafkaRequestStatusDeprovision)
	err := db.Error
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "unable to deprovision expired kafkas")
	}

	if db.RowsAffected >= 1 {
		glog.Infof("%v kafka_request's lifespans are over %d hours and have had their status updated to deprovisioning", db.RowsAffected, kafkaAgeInHours)
		var counter int64 = 0
		for ; counter < db.RowsAffected; counter++ {
			metrics.IncreaseKafkaTotalOperationsCountMetric(constants2.KafkaOperationDeprovision)
			metrics.IncreaseKafkaSuccessOperationsCountMetric(constants2.KafkaOperationDeprovision)
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
			clientId := BuildKeycloakClientNameIdentifier(kafkaRequest.ID)
			keycloakErr := k.keycloakService.DeRegisterClientInSSO(clientId)
			if keycloakErr != nil {
				return errors.NewWithCause(errors.ErrorGeneral, keycloakErr, "error deleting sso client")
			}
		}

		if k.kafkaConfig.EnableKafkaExternalCertificate {
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

	metrics.IncreaseKafkaTotalOperationsCountMetric(constants2.KafkaOperationDelete)
	metrics.IncreaseKafkaSuccessOperationsCountMetric(constants2.KafkaOperationDelete)

	return nil
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
		user := auth.GetUsernameFromClaims(claims)
		if user == "" {
			return nil, nil, errors.Unauthenticated("user not authenticated")
		}

		orgId := auth.GetOrgIdFromClaims(claims)
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
			return kafkaRequestList, pagingMeta, errors.NewWithCause(errors.ErrorFailedToParseSearch, err, "Unable to list kafka requests: %s", err.Error())
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
		return kafkaRequestList, pagingMeta, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to list kafka requests")
	}

	return kafkaRequestList, pagingMeta, nil
}

func (k *kafkaService) GetManagedKafkaByClusterID(clusterID string) ([]managedkafka.ManagedKafka, *errors.ServiceError) {
	dbConn := k.connectionFactory.New().
		Where("cluster_id = ?", clusterID).
		Where("status IN (?)", kafkaManagedCRStatuses).
		Where("bootstrap_server_host != ''")

	if k.keycloakService.GetConfig().EnableAuthenticationOnKafka {
		dbConn = dbConn.
			Where("sso_client_id != ''").
			Where("sso_client_secret != ''")
	}

	var kafkaRequestList dbapi.KafkaList
	if err := dbConn.Find(&kafkaRequestList).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "unable to list kafka requests")
	}

	var res []managedkafka.ManagedKafka
	// convert kafka requests to managed kafka
	for _, kafkaRequest := range kafkaRequestList {
		ns, _ := BuildNamespaceName(kafkaRequest)
		mk := BuildManagedKafkaCR(kafkaRequest, k.kafkaConfig, k.keycloakService.GetConfig(), ns)
		res = append(res, *mk)
	}

	return res, nil
}

func (k *kafkaService) Update(kafkaRequest *dbapi.KafkaRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New().
		Model(kafkaRequest).
		Where("status not IN (?)", kafkaDeletionStatuses) // ignore updates of kafka under deletion

	if err := dbConn.Updates(kafkaRequest).Error; err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update kafka")
	}

	return nil
}

func (k *kafkaService) UpdateStatus(id string, status constants2.KafkaStatus) (bool, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()

	if kafka, err := k.GetById(id); err != nil {
		return true, errors.NewWithCause(errors.ErrorGeneral, err, "failed to update status")
	} else {
		// only allow to change the status to "deleting" if the cluster is already in "deprovision" status
		if kafka.Status == constants2.KafkaRequestStatusDeprovision.String() && status != constants2.KafkaRequestStatusDeleting {
			return false, errors.GeneralError("failed to update status: cluster is deprovisioning")
		}

		if kafka.Status == status.String() {
			// no update needed
			return false, errors.GeneralError("failed to update status: the cluster %s is already in %s state", id, status.String())
		}
	}

	if err := dbConn.Model(&dbapi.KafkaRequest{Meta: api.Meta{ID: id}}).Update("status", status).Error; err != nil {
		return true, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to update kafka status")
	}

	return true, nil
}

func (k *kafkaService) ChangeKafkaCNAMErecords(kafkaRequest *dbapi.KafkaRequest, action KafkaRoutesAction) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
	routes, err := kafkaRequest.GetRoutes()
	if routes == nil || err != nil {
		glog.Infof("failed to parse routes data for kafka %s, routes := %v", kafkaRequest.ID, kafkaRequest.Routes)
		if clusterDNS, err := k.clusterService.GetClusterDNS(kafkaRequest.ClusterID); err != nil {
			return nil, err
		} else {
			routes = kafkaRequest.GetDefaultRoutes(clusterDNS, k.kafkaConfig.NumOfBrokers)
			glog.Infof("built default routes %v for kafka %s", kafkaRequest.ID, routes)
		}
	}
	domainRecordBatch := buildKafkaClusterCNAMESRecordBatch(routes, string(action))

	// Create AWS client with the region of this Kafka Cluster
	awsConfig := aws.Config{
		AccessKeyID:     k.awsConfig.Route53AccessKey,
		SecretAccessKey: k.awsConfig.Route53SecretAccessKey,
	}
	awsClient, err := k.awsClientFactory.NewClient(awsConfig, kafkaRequest.Region)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to create aws client")
	}

	changeRecordsOutput, err := awsClient.ChangeResourceRecordSets(k.kafkaConfig.KafkaDomainName, domainRecordBatch)
	if err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Unable to create domain record sets")
	}

	return changeRecordsOutput, nil
}

type KafkaStatusCount struct {
	Status constants2.KafkaStatus
	Count  int
}

func (k *kafkaService) CountByStatus(status []constants2.KafkaStatus) ([]KafkaStatusCount, error) {
	dbConn := k.connectionFactory.New()
	var results []KafkaStatusCount
	if err := dbConn.Model(&dbapi.KafkaRequest{}).Select("status as Status, count(1) as Count").Where("status in (?)", status).Group("status").Scan(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "Failed to count kafkas")
	}

	// if there is no count returned for a status from the above query because there is no kafkas in such a status,
	// we should return the count for these as well to avoid any confusion
	if len(status) > 0 {
		countersMap := map[constants2.KafkaStatus]int{}
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

func (k *kafkaService) ListKafkasWithRoutesNotCreated() ([]*dbapi.KafkaRequest, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var results []*dbapi.KafkaRequest
	if err := dbConn.Where("routes IS NOT NULL").Where("routes_created = ?", "no").Find(&results).Error; err != nil {
		return nil, errors.NewWithCause(errors.ErrorGeneral, err, "failed to list kafka requests")
	}
	return results, nil
}

func BuildManagedKafkaCR(kafkaRequest *dbapi.KafkaRequest, kafkaConfig *config.KafkaConfig, keycloakConfig *keycloak.KeycloakConfig, namespace string) *managedkafka.ManagedKafka {
	managedKafkaCR := &managedkafka.ManagedKafka{
		Id: kafkaRequest.ID,
		TypeMeta: metav1.TypeMeta{
			Kind:       "ManagedKafka",
			APIVersion: "managedkafka.bf2.org/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaRequest.Name,
			Namespace: namespace,
			Annotations: map[string]string{
				"bf2.org/id":          kafkaRequest.ID,
				"bf2.org/placementId": kafkaRequest.PlacementId,
			},
		},
		Spec: managedkafka.ManagedKafkaSpec{
			Capacity: managedkafka.Capacity{
				IngressEgressThroughputPerSec: kafkaConfig.KafkaCapacity.IngressEgressThroughputPerSec,
				TotalMaxConnections:           kafkaConfig.KafkaCapacity.TotalMaxConnections,
				MaxDataRetentionSize:          kafkaConfig.KafkaCapacity.MaxDataRetentionSize,
				MaxPartitions:                 kafkaConfig.KafkaCapacity.MaxPartitions,
				MaxDataRetentionPeriod:        kafkaConfig.KafkaCapacity.MaxDataRetentionPeriod,
				MaxConnectionAttemptsPerSec:   kafkaConfig.KafkaCapacity.MaxConnectionAttemptsPerSec,
			},
			Endpoint: managedkafka.EndpointSpec{
				BootstrapServerHost: kafkaRequest.BootstrapServerHost,
			},
			// These values must be changed as soon as we will have the real values
			Versions: managedkafka.VersionsSpec{
				Kafka: kafkaRequest.DesiredKafkaVersion,
				//TODO: we should remove the strimzi version as it should not be specified here
				Strimzi: "strimzi-cluster-operator.v0.23.0-0",
			},
			Deleted: kafkaRequest.Status == constants2.KafkaRequestStatusDeprovision.String(),
			Owners: []string{
				kafkaRequest.Owner,
			},
		},
		Status: managedkafka.ManagedKafkaStatus{},
	}

	if keycloakConfig.EnableAuthenticationOnKafka {
		managedKafkaCR.Spec.OAuth = managedkafka.OAuthSpec{
			ClientId:               kafkaRequest.SsoClientID,
			ClientSecret:           kafkaRequest.SsoClientSecret,
			TokenEndpointURI:       keycloakConfig.KafkaRealm.TokenEndpointURI,
			JwksEndpointURI:        keycloakConfig.KafkaRealm.JwksEndpointURI,
			ValidIssuerEndpointURI: keycloakConfig.KafkaRealm.ValidIssuerURI,
			UserNameClaim:          keycloakConfig.UserNameClaim,
			CustomClaimCheck:       BuildCustomClaimCheck(kafkaRequest),
		}

		if keycloakConfig.TLSTrustedCertificatesValue != "" {
			managedKafkaCR.Spec.OAuth.TlsTrustedCertificate = &keycloakConfig.TLSTrustedCertificatesValue
		}
	}

	if kafkaConfig.EnableKafkaExternalCertificate {
		managedKafkaCR.Spec.Endpoint.Tls = &managedkafka.TlsSpec{
			Cert: kafkaConfig.KafkaTLSCert,
			Key:  kafkaConfig.KafkaTLSKey,
		}
	}

	return managedKafkaCR
}

func buildKafkaClusterCNAMESRecordBatch(routes []dbapi.DataPlaneKafkaRoute, action string) *route53.ChangeBatch {
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

func buildResourceRecordChange(recordName string, clusterIngress string, action string) *route53.Change {
	recordType := "CNAME"
	recordTTL := int64(300)

	resourceRecordChange := &route53.Change{
		Action: &action,
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
