package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/getsentry/sentry-go"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/auth"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/aws"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	constants "gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/db"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/metrics"
)

//go:generate moq -out kafkaservice_moq.go . KafkaService
type KafkaService interface {
	Create(kafkaRequest *api.KafkaRequest) *errors.ServiceError
	// Get method will retrieve the kafkaRequest instance that the give ctx has access to from the database.
	// This should be used when you want to make sure the result is filtered based on the request context.
	Get(ctx context.Context, id string) (*api.KafkaRequest, *errors.ServiceError)
	// GetById method will retrieve the KafkaRequest instance from the database without checking any permissions.
	// You should only use this if you are sure permission check is not required.
	GetById(id string) (*api.KafkaRequest, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	List(ctx context.Context, listArgs *ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError)
	RegisterKafkaJob(kafkaRequest *api.KafkaRequest) *errors.ServiceError
	ListByStatus(status constants.KafkaStatus) ([]*api.KafkaRequest, *errors.ServiceError)
	UpdateStatus(id string, status constants.KafkaStatus) *errors.ServiceError
	Update(kafkaRequest *api.KafkaRequest) *errors.ServiceError
	RegisterKafkaInSSO(ctx context.Context, kafkaRequest *api.KafkaRequest) *errors.ServiceError
	ChangeKafkaCNAMErecords(kafkaRequest *api.KafkaRequest, clusterDNS string, action string) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError)
}

var _ KafkaService = &kafkaService{}

type kafkaService struct {
	connectionFactory *db.ConnectionFactory
	syncsetService    SyncsetService
	clusterService    ClusterService
	keycloakService   KeycloakService
	kafkaConfig       *config.KafkaConfig
	awsConfig         *config.AWSConfig
}

func NewKafkaService(connectionFactory *db.ConnectionFactory, syncsetService SyncsetService, clusterService ClusterService, keycloakService KeycloakService, kafkaConfig *config.KafkaConfig, awsConfig *config.AWSConfig) *kafkaService {
	return &kafkaService{
		connectionFactory: connectionFactory,
		syncsetService:    syncsetService,
		clusterService:    clusterService,
		keycloakService:   keycloakService,
		kafkaConfig:       kafkaConfig,
		awsConfig:         awsConfig,
	}
}

// RegisterKafkaJob registers a new job in the kafka table
func (k *kafkaService) RegisterKafkaJob(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	kafkaRequest.Status = constants.KafkaRequestStatusAccepted.String()
	if err := dbConn.Save(kafkaRequest).Error; err != nil {
		return errors.GeneralError("failed to create kafka job: %v", err)
	}
	return nil
}

// Create will create a new kafka cr with the given configuration,
// and sync it via a syncset to an available cluster with capacity
// in the desired region for the desired cloud provider.
// The kafka object in the database will be updated with a updated_at
// timestamp and the corresponding cluster identifier.
func (k *kafkaService) Create(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
	truncatedKafkaIdentifier := buildTruncateKafkaIdentifier(kafkaRequest)
	truncatedKafkaIdentifier, replaceErr := replaceHostSpecialChar(truncatedKafkaIdentifier)
	if replaceErr != nil {
		sentry.CaptureException(replaceErr)
		return errors.GeneralError("generated host is not valid: %v", replaceErr)
	}

	clusterDNS, err := k.clusterService.GetClusterDNS(kafkaRequest.ClusterID)
	if err != nil || clusterDNS == "" {
		sentry.CaptureException(err)
		return errors.GeneralError("error retrieving cluster DNS: %v", err)
	}
	// We are manually creating a new IngressController on the data plane OSD clusters which the Kafka Clusters will use
	// Our ClusterDNS reference needs to match this
	if k.kafkaConfig.EnableDedicatedIngress {
		clusterDNS = strings.Replace(clusterDNS, "apps", "mk", 1)
	}
	kafkaRequest.BootstrapServerHost = fmt.Sprintf("%s.%s", truncatedKafkaIdentifier, clusterDNS)

	if k.kafkaConfig.EnableKafkaTLS {
		// If we enable KafkaTLS, the bootstrapServerHost should use the external domain name rather than the cluster domain
		kafkaRequest.BootstrapServerHost = fmt.Sprintf("%s.%s", truncatedKafkaIdentifier, k.kafkaConfig.KafkaDomainName)

		_, err = k.ChangeKafkaCNAMErecords(kafkaRequest, clusterDNS, "CREATE")
		if err != nil {
			return err
		}
	}

	var clientSecretValue string
	if k.keycloakService.GetConfig().EnableAuthenticationOnKafka {
		clientName := buildKeycloakClientNameIdentifier(kafkaRequest)
		clientSecretValue, err = k.keycloakService.GetSecretForRegisteredKafkaClient(clientName)
		if err != nil || clientSecretValue == "" {
			return errors.GeneralError("failed to create sso client: %v", err)
		}
	}
	// create the syncset builder
	syncsetBuilder, syncsetId, err := newKafkaSyncsetBuilder(kafkaRequest, k.kafkaConfig, k.keycloakService.GetConfig(), clientSecretValue)
	if err != nil {
		sentry.CaptureException(err)
		return errors.GeneralError("error creating kafka syncset builder: %v", err)
	}

	// create the syncset
	_, err = k.syncsetService.Create(syncsetBuilder, syncsetId, kafkaRequest.ClusterID)
	if err != nil {
		sentry.CaptureException(err)
		return errors.GeneralError("error creating syncset: %v", err)
	}

	// Update the Kafka Request record in the database
	// Only updates the fields below
	updatedKafkaRequest := &api.KafkaRequest{
		Meta: api.Meta{
			ID: kafkaRequest.ID,
		},
		BootstrapServerHost: kafkaRequest.BootstrapServerHost,
	}
	if err := k.Update(updatedKafkaRequest); err != nil {
		return errors.GeneralError("failed to update kafka request: %v", err)
	}

	return nil
}

func (k *kafkaService) RegisterKafkaInSSO(ctx context.Context, kafkaRequest *api.KafkaRequest) *errors.ServiceError {
	if k.keycloakService.GetConfig().EnableAuthenticationOnKafka {
		orgId := auth.GetOrgIdFromContext(ctx)
		// registering client in sso
		clientName := buildKeycloakClientNameIdentifier(kafkaRequest)
		keycloakSecret, err := k.keycloakService.RegisterKafkaClientInSSO(clientName, orgId)
		if err != nil || keycloakSecret == "" {
			return errors.GeneralError("failed to create sso client: %v", err)
		}
	}
	return nil
}

func (k *kafkaService) ListByStatus(status constants.KafkaStatus) ([]*api.KafkaRequest, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()

	var kafkas []*api.KafkaRequest

	if err := dbConn.Model(&api.KafkaRequest{}).Where("status = ?", status).Scan(&kafkas).Error; err != nil {
		return nil, errors.GeneralError(err.Error())
	}

	return kafkas, nil
}

func (k *kafkaService) Get(ctx context.Context, id string) (*api.KafkaRequest, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	user := auth.GetUsernameFromContext(ctx)
	if user == "" {
		return nil, errors.Unauthenticated("user not authenticated")
	}

	orgId := auth.GetOrgIdFromContext(ctx)
	dbConn := k.connectionFactory.New().Where("id = ?", id)
	var kafkaRequest api.KafkaRequest
	if orgId != "" {
		dbConn = dbConn.Where("organisation_id = ?", orgId)
	} else {
		dbConn = dbConn.Where("owner = ?", user)
	}
	if err := dbConn.First(&kafkaRequest).Error; err != nil {
		return nil, handleGetError("KafkaResource", "id", id, err)
	}
	return &kafkaRequest, nil
}

func (k *kafkaService) GetById(id string) (*api.KafkaRequest, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	dbConn := k.connectionFactory.New()
	var kafkaRequest api.KafkaRequest
	if err := dbConn.Where("id = ?", id).First(&kafkaRequest).Error; err != nil {
		return nil, handleGetError("KafkaResource", "id", id, err)
	}
	return &kafkaRequest, nil
}

// Delete deletes a kafka request and its corresponding syncset from
// the associated cluster it was deployed on. Deleting the syncset will
// delete all resources (Kafka CR, Project) associated with the syncset.
// The kafka object in the database will be updated with a deleted_at
// timestamp.
func (k *kafkaService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if id == "" {
		return errors.Validation("id is undefined")
	}

	// filter kafka request by owner to only retrieve request of the current authenticated user
	user := auth.GetUsernameFromContext(ctx)
	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Where("id = ?", id).Where("owner = ? ", user)

	var kafkaRequest api.KafkaRequest
	if err := dbConn.First(&kafkaRequest).Error; err != nil {
		return handleGetError("KafkaResource", "id", id, err)
	}

	metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationDelete)

	// delete the kafka client in mas sso
	if k.keycloakService.GetConfig().EnableAuthenticationOnKafka {
		clientName := buildKeycloakClientNameIdentifier(&kafkaRequest)
		err := k.keycloakService.DeRegisterKafkaClientInSSO(clientName)
		if err != nil {
			return errors.GeneralError("error deleting sso client: %v", err)
		}
	}

	if k.kafkaConfig.EnableKafkaTLS {
		clusterDNS, err := k.clusterService.GetClusterDNS(kafkaRequest.ClusterID)
		if err != nil || clusterDNS == "" {
			sentry.CaptureException(err)
			return errors.GeneralError("error retrieving cluster DNS: %v", err)
		}
		// We are manually creating a new IngressController on the data plane OSD clusters which the Kafka Clusters will use
		// Our ClusterDNS reference needs to match this
		if k.kafkaConfig.EnableDedicatedIngress {
			clusterDNS = strings.Replace(clusterDNS, "apps", "mk", 1)
		}

		_, err = k.ChangeKafkaCNAMErecords(&kafkaRequest, clusterDNS, "DELETE")
		if err != nil {
			return err
		}
	}

	// delete the syncset
	syncsetId := buildSyncsetIdentifier(&kafkaRequest)
	statusCode, err := k.syncsetService.Delete(syncsetId, kafkaRequest.ClusterID)

	if err != nil && statusCode != http.StatusNotFound {
		sentry.CaptureException(err)
		return errors.GeneralError("error deleting syncset: %v", err)
	}

	// soft delete the kafka request
	if err := dbConn.Delete(&kafkaRequest).Error; err != nil {
		return errors.GeneralError("unable to delete kafka request with id %s: %s", kafkaRequest.ID, err)
	}

	metrics.IncreaseKafkaSuccessOperationsCountMetric(constants.KafkaOperationDelete)

	return nil
}

// List returns all Kafka requests belonging to a user.
func (k *kafkaService) List(ctx context.Context, listArgs *ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError) {
	var kafkaRequestList api.KafkaList
	dbConn := k.connectionFactory.New()
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	user := auth.GetUsernameFromContext(ctx)
	if user == "" {
		return nil, nil, errors.Unauthenticated("user not authenticated")
	}

	orgId := auth.GetOrgIdFromContext(ctx)

	if orgId != "" {
		// filter kafka requests by organisation_id since the user is allowed to see all kafka requests of my id
		dbConn = dbConn.Where("organisation_id = ?", orgId)
	} else {
		// filter kafka requests by owner as we are dealing with service accounts which may not have an org id
		dbConn = dbConn.Where("owner = ?", user)
	}

	// TODO add integration tests
	// Apply search query
	if len(listArgs.Search) > 0 {
		searchDbQuery, err := GetSearchQuery(listArgs.Search)
		if err != nil {
			return kafkaRequestList, pagingMeta, errors.FailedToParseSearch("Unable to list kafka requests for %s: %s", user, err)
		}
		for _, dbQuery := range searchDbQuery {
			dbConn = dbConn.Where(dbQuery.query, dbQuery.value)
		}
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
	dbConn.Model(&kafkaRequestList).Count(&pagingMeta.Total)
	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// execute query
	if err := dbConn.Find(&kafkaRequestList).Error; err != nil {
		return kafkaRequestList, pagingMeta, errors.GeneralError("Unable to list kafka requests for %s: %s", user, err)
	}

	return kafkaRequestList, pagingMeta, nil
}

func (k kafkaService) Update(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	if err := dbConn.Model(kafkaRequest).Update(kafkaRequest).Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}
	return nil
}

func (k kafkaService) UpdateStatus(id string, status constants.KafkaStatus) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	if err := dbConn.Model(&api.KafkaRequest{Meta: api.Meta{ID: id}}).Update("status", status).Error; err != nil {
		return errors.GeneralError("failed to update status: %s", err.Error())
	}

	return nil
}

func (k kafkaService) ChangeKafkaCNAMErecords(kafkaRequest *api.KafkaRequest, clusterDNS string, action string) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
	domainRecordBatch := buildKafkaClusterCNAMESRecordBatch(kafkaRequest.BootstrapServerHost, clusterDNS, action, k.kafkaConfig)

	// Create AWS client with the region of this Kafka Cluster
	awsConfig := aws.Config{
		AccessKeyID:     k.awsConfig.Route53AccessKey,
		SecretAccessKey: k.awsConfig.Route53SecretAccessKey,
	}
	awsClient, err := aws.NewClient(awsConfig, kafkaRequest.Region)
	if err != nil {
		sentry.CaptureException(err)
		return nil, errors.GeneralError("Unable to create aws client: %v", err)
	}

	changeRecordsOutput, err := awsClient.ChangeResourceRecordSets(k.kafkaConfig.KafkaDomainName, domainRecordBatch)
	if err != nil {
		sentry.CaptureException(err)
		return nil, errors.GeneralError("Unable to create domain record sets: %v", err)
	}

	return changeRecordsOutput, nil
}

func buildKafkaClusterCNAMESRecordBatch(recordName string, clusterIngress string, action string, kafkaConfig *config.KafkaConfig) *route53.ChangeBatch {
	// Need to append some string to the start of the clusterIngress for the CNAME record
	clusterIngress = fmt.Sprintf("elb.%s", clusterIngress)

	recordChangeBatch := &route53.ChangeBatch{
		Changes: []*route53.Change{
			buildResourceRecordChange(recordName, clusterIngress, action),
			buildResourceRecordChange(fmt.Sprintf("admin-server-%s", recordName), clusterIngress, action),
		},
	}

	for i := 0; i < kafkaConfig.NumOfBrokers; i++ {
		recordName := fmt.Sprintf("broker-%d-%s", i, recordName)
		recordChangeBatch.Changes = append(recordChangeBatch.Changes, buildResourceRecordChange(recordName, clusterIngress, action))
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
