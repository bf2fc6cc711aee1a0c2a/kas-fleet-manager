package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	managedkafka "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/aws"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/syncsetresources"
	"github.com/getsentry/sentry-go"
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
	Delete(*api.KafkaRequest) *errors.ServiceError
	List(ctx context.Context, listArgs *ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError)
	// Returns all the requests for a given user (no paging)
	GetByClusterID(clusterID string) (api.KafkaList, *errors.ServiceError)
	GetManagedKafkaByClusterID(clusterID string) ([]managedkafka.ManagedKafka, *errors.ServiceError)
	RegisterKafkaJob(kafkaRequest *api.KafkaRequest) *errors.ServiceError
	ListByStatus(status constants.KafkaStatus) ([]*api.KafkaRequest, *errors.ServiceError)
	// UpdateStatus change the status of the Kafka cluster
	// The returned boolean is to be used to know if the update has been tried or not. An update is not tried if the
	// original status is 'deprovision' (cluster in deprovision state can't be change state) or if the final status is the
	// same as the original status. The error will contain any error encountered when attempting to update or the reason
	// why no attempt has been done
	UpdateStatus(id string, status constants.KafkaStatus) (bool, *errors.ServiceError)
	Update(kafkaRequest *api.KafkaRequest) *errors.ServiceError
	ChangeKafkaCNAMErecords(kafkaRequest *api.KafkaRequest, clusterDNS string, action string) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError)
	RegisterKafkaDeprovisionJob(ctx context.Context, id string) *errors.ServiceError
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
	clusterDNS = strings.Replace(clusterDNS, constants.DefaultIngressDnsNamePrefix, constants.ManagedKafkaIngressDnsNamePrefix, 1)
	kafkaRequest.BootstrapServerHost = fmt.Sprintf("%s.%s", truncatedKafkaIdentifier, clusterDNS)

	if k.kafkaConfig.EnableKafkaExternalCertificate {
		// If we enable KafkaTLS, the bootstrapServerHost should use the external domain name rather than the cluster domain
		kafkaRequest.BootstrapServerHost = fmt.Sprintf("%s.%s", truncatedKafkaIdentifier, k.kafkaConfig.KafkaDomainName)

		_, err = k.ChangeKafkaCNAMErecords(kafkaRequest, clusterDNS, "CREATE")
		if err != nil {
			return err
		}
	}

	var clientSecretValue string
	if k.keycloakService.GetConfig().EnableAuthenticationOnKafka {
		clientName := syncsetresources.BuildKeycloakClientNameIdentifier(kafkaRequest.ID)
		clientSecretValue, err = k.keycloakService.GetSecretForRegisteredKafkaClient(clientName)
		if err != nil || clientSecretValue == "" {
			return errors.FailedToCreateSSOClient("Failed to create sso client: %v", err)
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
		return err
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

func (k *kafkaService) ListByStatus(status constants.KafkaStatus) ([]*api.KafkaRequest, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()

	var kafkas []*api.KafkaRequest

	if err := dbConn.Model(&api.KafkaRequest{}).Scan(&kafkas).Error; err != nil {
		return nil, errors.GeneralError(err.Error())
	}

	if err := dbConn.Model(&api.KafkaRequest{}).Where("status = ?", status).Scan(&kafkas).Error; err != nil {
		return nil, errors.GeneralError(err.Error())
	}

	return kafkas, nil
}

func (k *kafkaService) Get(ctx context.Context, id string) (*api.KafkaRequest, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("id is undefined")
	}

	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, errors.Unauthenticated("user not authenticated: %s", err.Error())
	}

	user := auth.GetUsernameFromClaims(claims)
	if user == "" {
		return nil, errors.Unauthenticated("user not authenticated")
	}

	orgId := auth.GetOrgIdFromClaims(claims)
	userIsAllowedAsServiceAccount := auth.GetUserIsAllowedAsServiceAccountFromContext(ctx)

	dbConn := k.connectionFactory.New().Where("id = ?", id)

	// filter by organisationId if a user is part of an organisation and is not allowed as a service account
	filterByOrganisationId := !userIsAllowedAsServiceAccount && orgId != ""
	if filterByOrganisationId {
		dbConn = dbConn.Where("organisation_id = ?", orgId)
	} else {
		dbConn = dbConn.Where("owner = ?", user)
	}

	var kafkaRequest api.KafkaRequest
	if err := dbConn.First(&kafkaRequest).Error; err != nil {
		return nil, handleGetError("KafkaResource for user "+user, "id", id, err)
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

// RegisterKafkaJob registers a kafka deprovision job in the kafka table
func (k *kafkaService) RegisterKafkaDeprovisionJob(ctx context.Context, id string) *errors.ServiceError {
	if id == "" {
		return errors.Validation("id is undefined")
	}

	// filter kafka request by owner to only retrieve request of the current authenticated user
	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return errors.Unauthenticated("user not authenticated")
	}
	user := auth.GetUsernameFromClaims(claims)
	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Where("id = ?", id).Where("owner = ? ", user)

	var kafkaRequest api.KafkaRequest
	if err := dbConn.First(&kafkaRequest).Error; err != nil {
		return handleGetError("KafkaResource", "id", id, err)
	}
	metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationDeprovision)

	if executed, err := k.UpdateStatus(id, constants.KafkaRequestStatusDeprovision); executed {
		if err != nil {
			return handleGetError("KafkaResource", "id", id, err)
		}
		metrics.IncreaseKafkaSuccessOperationsCountMetric(constants.KafkaOperationDeprovision)
	}

	return nil
}

// Delete deletes a kafka request and its corresponding syncset from
// the associated cluster it was deployed on. Deleting the syncset will
// delete all resources (Kafka CR, Project) associated with the syncset.
// The kafka object in the database will be updated with a deleted_at
// timestamp.
func (k *kafkaService) Delete(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	// if the we don't have the clusterID we can only delete the row from the database
	if kafkaRequest.ClusterID != "" {
		// delete the kafka client in mas sso
		if k.keycloakService.GetConfig().EnableAuthenticationOnKafka {
			clientName := syncsetresources.BuildKeycloakClientNameIdentifier(kafkaRequest.ID)
			keycloakErr := k.keycloakService.DeRegisterKafkaClientInSSO(clientName)
			if keycloakErr != nil {
				return errors.GeneralError("error deleting sso client: %v", keycloakErr)
			}
		}

		if k.kafkaConfig.EnableKafkaExternalCertificate {
			clusterDNS, err := k.clusterService.GetClusterDNS(kafkaRequest.ClusterID)
			if err != nil || clusterDNS == "" {
				sentry.CaptureException(err)
				return errors.GeneralError("error retrieving cluster DNS: %v", err)
			}
			clusterDNS = strings.Replace(clusterDNS, constants.DefaultIngressDnsNamePrefix, constants.ManagedKafkaIngressDnsNamePrefix, 1)

			_, err = k.ChangeKafkaCNAMErecords(kafkaRequest, clusterDNS, "DELETE")
			if err != nil {
				return err
			}
		}

		// delete the syncset
		syncsetId := buildSyncsetIdentifier(kafkaRequest)
		statucCode, err := k.syncsetService.Delete(syncsetId, kafkaRequest.ClusterID)

		if err != nil && statucCode != http.StatusNotFound {
			sentry.CaptureException(err)
			return errors.GeneralError("error deleting syncset: %v", err)
		}
	}

	// soft delete the kafka request
	if err := dbConn.Delete(kafkaRequest).Error; err != nil {
		return errors.GeneralError("unable to delete kafka request with id %s: %s", kafkaRequest.ID, err)
	}

	metrics.IncreaseKafkaTotalOperationsCountMetric(constants.KafkaOperationDelete)
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

	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return nil, nil, errors.Unauthenticated("user not authenticated")
	}

	user := auth.GetUsernameFromClaims(claims)
	if user == "" {
		return nil, nil, errors.Unauthenticated("user not authenticated")
	}

	orgId := auth.GetOrgIdFromClaims(claims)
	userIsAllowedAsServiceAccount := auth.GetUserIsAllowedAsServiceAccountFromContext(ctx)
	// filter by organisationId if a user is part of an organisation and is not allowed as a service account
	filterByOrganisationId := !userIsAllowedAsServiceAccount && orgId != ""

	if filterByOrganisationId {
		// filter kafka requests by organisation_id since the user is allowed to see all kafka requests of my id
		dbConn = dbConn.Where("organisation_id = ?", orgId)
	} else {
		// filter kafka requests by owner as we are dealing with service accounts which may not have an org id
		dbConn = dbConn.Where("owner = ?", user)
	}

	// Apply search query
	if len(listArgs.Search) > 0 {
		searchDbQuery, err := GetSearchQuery(listArgs.Search)
		if err != nil {
			return kafkaRequestList, pagingMeta, errors.FailedToParseSearch("Unable to list kafka requests for %s: %s", user, err)
		}

		dbConn = dbConn.Where(searchDbQuery.query, searchDbQuery.values...)
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

func (k *kafkaService) GetByClusterID(clusterID string) (api.KafkaList, *errors.ServiceError) {
	dbConn := k.connectionFactory.New().Where("cluster_id = ?", clusterID)

	var kafkaRequestList api.KafkaList
	if err := dbConn.Find(&kafkaRequestList).Error; err != nil {
		return kafkaRequestList, errors.GeneralError("Unable to list kafka requests %s", err)
	}
	return kafkaRequestList, nil
}

func (k *kafkaService) GetManagedKafkaByClusterID(clusterID string) ([]managedkafka.ManagedKafka, *errors.ServiceError) {
	kafkaRequestList, err := k.GetByClusterID(clusterID)
	if err != nil {
		return nil, err
	}

	var res []managedkafka.ManagedKafka
	// convert kafka requests to managed kafka
	for _, kafkaRequest := range kafkaRequestList {
		mk := BuildManagedKafkaCR(kafkaRequest, k.kafkaConfig, k.keycloakService.GetConfig(), buildKafkaNamespaceIdentifier(kafkaRequest))
		res = append(res, *mk)
	}

	return res, nil
}

func (k kafkaService) Update(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	if err := dbConn.Model(kafkaRequest).Update(kafkaRequest).Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}
	return nil
}

func (k kafkaService) UpdateStatus(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()

	if kafka, err := k.GetById(id); err != nil {
		return true, errors.GeneralError("failed to update status: %s", err.Error())
	} else {
		if kafka.Status == constants.KafkaRequestStatusDeprovision.String() {
			return false, errors.GeneralError("failed to update status: cluster is deprovisioning")
		}

		if kafka.Status == status.String() {
			// no update needed
			return false, errors.GeneralError("failed to update status: the cluster %s is already in %s state", id, status.String())
		}
	}

	if err := dbConn.Model(&api.KafkaRequest{Meta: api.Meta{ID: id}}).Update("status", status).Error; err != nil {
		return true, errors.GeneralError("failed to update status: %s", err.Error())
	}

	return true, nil
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

func BuildManagedKafkaCR(kafkaRequest *api.KafkaRequest, kafkaConfig *config.KafkaConfig, keycloakConfig *config.KeycloakConfig, namespace string) *managedkafka.ManagedKafka {
	managedKafkaCR := &managedkafka.ManagedKafka{
		Name:        kafkaRequest.Name,
		Id:          kafkaRequest.ID,
		PlacementId: kafkaRequest.PlacementId,
		ObjectMeta: metav1.ObjectMeta{
			Name:      kafkaRequest.Name,
			Namespace: namespace,
		},
		Spec: managedkafka.ManagedKafkaSpec{
			// Currently ignored
			Capacity: managedkafka.Capacity{
				IngressEgressThroughputPerSec: "",
				TotalMaxConnections:           0,
				MaxDataRetentionSize:          "",
				MaxPartitions:                 0,
				MaxDataRetentionPeriod:        "",
			},
			Endpoint: managedkafka.EndpointSpec{
				BootstrapServerHost: kafkaRequest.BootstrapServerHost,
				Tls: managedkafka.TlsSpec{
					Cert: kafkaConfig.KafkaTLSCert,
					Key:  kafkaConfig.KafkaTLSKey,
				},
			},
			// These values must be changed as soon as we will have the real values
			Versions: managedkafka.VersionsSpec{
				Kafka:   "2.6.0",
				Strimzi: "0.21.1",
			},
			Deleted: kafkaRequest.Status == constants.KafkaRequestStatusDeprovision.String(),
		},
		Status: managedkafka.ManagedKafkaStatus{},
	}

	if keycloakConfig.EnableAuthenticationOnKafka {
		managedKafkaCR.Spec.OAuth = managedkafka.OAuthSpec{
			ClientId:               keycloakConfig.KafkaRealm.ClientID,
			ClientSecret:           keycloakConfig.KafkaRealm.ClientSecret,
			TokenEndpointURI:       keycloakConfig.KafkaRealm.TokenEndpointURI,
			JwksEndpointURI:        keycloakConfig.KafkaRealm.JwksEndpointURI,
			ValidIssuerEndpointURI: keycloakConfig.KafkaRealm.ValidIssuerURI,
			UserNameClaim:          keycloakConfig.UserNameClaim,
			TlsTrustedCertificate:  keycloakConfig.TLSTrustedCertificatesValue,
		}
	}

	return managedKafkaCR
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
