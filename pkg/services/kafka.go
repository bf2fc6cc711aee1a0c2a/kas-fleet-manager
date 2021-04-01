package services

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"

	syncsetresources "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/syncsetresources"
	"github.com/golang/glog"
	"time"

	"github.com/google/uuid"

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
	"github.com/getsentry/sentry-go"
)

const productId = "RHOSAKTrial"

//go:generate moq -out kafkaservice_moq.go . KafkaService
type KafkaService interface {
	HasAvailableCapacity() (bool, *errors.ServiceError)
	Create(kafkaRequest *api.KafkaRequest) *errors.ServiceError
	// Get method will retrieve the kafkaRequest instance that the give ctx has access to from the database.
	// This should be used when you want to make sure the result is filtered based on the request context.
	Get(ctx context.Context, id string) (*api.KafkaRequest, *errors.ServiceError)
	// GetById method will retrieve the KafkaRequest instance from the database without checking any permissions.
	// You should only use this if you are sure permission check is not required.
	GetById(id string) (*api.KafkaRequest, *errors.ServiceError)
	Delete(*api.KafkaRequest) *errors.ServiceError
	List(ctx context.Context, listArgs *ListArguments) (api.KafkaList, *api.PagingMeta, *errors.ServiceError)
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
	// DeprovisionKafkaForUsers registers all kafkas for deprovisioning given the list of owners
	DeprovisionKafkaForUsers(users []string) *errors.ServiceError
	DeprovisionExpiredKafkas(kafkaAgeInHours int) *errors.ServiceError
}

var _ KafkaService = &kafkaService{}

type kafkaService struct {
	connectionFactory *db.ConnectionFactory
	syncsetService    SyncsetService
	clusterService    ClusterService
	keycloakService   KeycloakService
	kafkaConfig       *config.KafkaConfig
	awsConfig         *config.AWSConfig
	quotaService      QuotaService
	mu                sync.Mutex
}

func NewKafkaService(connectionFactory *db.ConnectionFactory, syncsetService SyncsetService, clusterService ClusterService, keycloakService KeycloakService, kafkaConfig *config.KafkaConfig, awsConfig *config.AWSConfig, quotaService QuotaService) *kafkaService {
	return &kafkaService{
		connectionFactory: connectionFactory,
		syncsetService:    syncsetService,
		clusterService:    clusterService,
		keycloakService:   keycloakService,
		kafkaConfig:       kafkaConfig,
		awsConfig:         awsConfig,
		quotaService:      quotaService,
	}
}

func (k *kafkaService) HasAvailableCapacity() (bool, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()
	var count int

	if err := dbConn.Model(&api.KafkaRequest{}).Count(&count).Error; err != nil {
		return false, errors.GeneralError("failed counting kafkas: %v", err)
	}

	glog.Infof("%d of %d kafka clusters currently instantiated", count, k.kafkaConfig.KafkaCapacity.MaxCapacity)
	return count < k.kafkaConfig.KafkaCapacity.MaxCapacity, nil
}

// RegisterKafkaJob registers a new job in the kafka table
func (k *kafkaService) RegisterKafkaJob(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
	k.mu.Lock()
	defer k.mu.Unlock()
	if hasCapacity, err := k.HasAvailableCapacity(); err != nil {
		return err
	} else if !hasCapacity {
		glog.Warningf("Cluster capacity(%d) exhausted", k.kafkaConfig.KafkaCapacity.MaxCapacity)
		return errors.TooManyKafkaInstancesReached("cluster capacity exhausted")
	}
	//cluster id can't be nil. generating random temporary id.
	//reserve is false, checking whether a user can reserve a quota or not
	if k.kafkaConfig.EnableQuotaService {
		isAllowed, _, err := k.quotaService.ReserveQuota(productId, kafkaRequest.ClusterID, uuid.New().String(), kafkaRequest.Owner, false, "single")
		if err != nil {
			return errors.FailedToCheckQuota("%v", err)
		}
		if !isAllowed {
			return errors.InsufficientQuotaError("Insufficient Quota")
		}
	}
	dbConn := k.connectionFactory.New()
	kafkaRequest.Version = k.kafkaConfig.DefaultKafkaVersion
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

	if k.keycloakService.GetConfig().EnableAuthenticationOnKafka {
		kafkaRequest.SsoClientID = syncsetresources.BuildKeycloakClientNameIdentifier(kafkaRequest.ID)
		kafkaRequest.SsoClientSecret, err = k.keycloakService.RegisterKafkaClientInSSO(kafkaRequest.SsoClientID, kafkaRequest.OrganisationId)
		if err != nil || kafkaRequest.SsoClientSecret == "" {
			sentry.CaptureException(err)
			return errors.FailedToCreateSSOClient("failed to create sso client %s:%v", kafkaRequest.SsoClientID, err)
		}
	}

	// only use sync set if EnableKasFleetshardSync is set to false
	if !k.kafkaConfig.EnableKasFleetshardSync {
		// create the syncset builder
		syncsetBuilder, syncsetId, err := newKafkaSyncsetBuilder(kafkaRequest, k.kafkaConfig, k.keycloakService.GetConfig())
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
	}

	// Update the Kafka Request record in the database
	// Only updates the fields below
	updatedKafkaRequest := &api.KafkaRequest{
		Meta: api.Meta{
			ID: kafkaRequest.ID,
		},
		BootstrapServerHost: kafkaRequest.BootstrapServerHost,
		SsoClientID:         kafkaRequest.SsoClientID,
		SsoClientSecret:     kafkaRequest.SsoClientSecret,
	}
	if err := k.Update(updatedKafkaRequest); err != nil {
		return errors.GeneralError("failed to update kafka request: %v", err)
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

func (k *kafkaService) DeprovisionKafkaForUsers(users []string) *errors.ServiceError {
	dbConn := k.connectionFactory.New().Model(&api.KafkaRequest{}).Where("owner IN (?)", users)

	if err := dbConn.Update("status", constants.KafkaRequestStatusDeprovision).Error; err != nil {
		return errors.GeneralError("Unable to deprovision kafka requests for users %s", err)
	}

	return nil
}

func (k *kafkaService) DeprovisionExpiredKafkas(kafkaAgeInHours int) *errors.ServiceError {
	dbConn := k.connectionFactory.New().Model(&api.KafkaRequest{}).Where("created_at  <=  ?", time.Now().Add(-1*time.Duration(kafkaAgeInHours)*time.Hour))

	if err := dbConn.Update("status", constants.KafkaRequestStatusDeprovision).Error; err != nil {
		return errors.GeneralError("unable to deprovision expired kafkas: %v", err)
	}
	glog.Infof("%v kafka_request's lifespans are over %d hours and have had their status updated to deprovisioning", dbConn.RowsAffected, kafkaAgeInHours)

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
			clientId := syncsetresources.BuildKeycloakClientNameIdentifier(kafkaRequest.ID)
			keycloakErr := k.keycloakService.DeRegisterClientInSSO(clientId)
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

		// only delete the sync set if kas-fleetshard sync is not enabled
		// otherwise the kas-fleetshard will get the information through the endpoints
		if !k.kafkaConfig.EnableKasFleetshardSync {
			// delete the syncset
			syncsetId := buildSyncsetIdentifier(kafkaRequest)
			statucCode, err := k.syncsetService.Delete(syncsetId, kafkaRequest.ClusterID)

			if err != nil && statucCode != http.StatusNotFound {
				sentry.CaptureException(err)
				return errors.GeneralError("error deleting syncset: %v", err)
			}
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

func (k *kafkaService) GetManagedKafkaByClusterID(clusterID string) ([]managedkafka.ManagedKafka, *errors.ServiceError) {
	dbConn := k.connectionFactory.New().Where("cluster_id = ? AND status IN (?)", clusterID, []string{constants.KafkaRequestStatusProvisioning.String(), constants.KafkaRequestStatusDeprovision.String(), constants.KafkaRequestStatusReady.String()})
	var kafkaRequestList api.KafkaList
	if err := dbConn.Find(&kafkaRequestList).Error; err != nil {
		return nil, errors.GeneralError("Unable to list kafka requests %s", err)
	}

	var res []managedkafka.ManagedKafka
	// convert kafka requests to managed kafka
	for _, kafkaRequest := range kafkaRequestList {
		ns := buildKafkaNamespaceIdentifier(kafkaRequest)
		ns, _ = replaceNamespaceSpecialChar(ns)
		mk := BuildManagedKafkaCR(kafkaRequest, k.kafkaConfig, k.keycloakService.GetConfig(), ns)
		res = append(res, *mk)
	}

	return res, nil
}

func (k *kafkaService) Update(kafkaRequest *api.KafkaRequest) *errors.ServiceError {
	dbConn := k.connectionFactory.New()

	if err := dbConn.Model(kafkaRequest).Update(kafkaRequest).Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}
	return nil
}

func (k *kafkaService) UpdateStatus(id string, status constants.KafkaStatus) (bool, *errors.ServiceError) {
	dbConn := k.connectionFactory.New()

	if kafka, err := k.GetById(id); err != nil {
		return true, errors.GeneralError("failed to update status: %s", err.Error())
	} else {
		if kafka.Status == constants.KafkaRequestStatusDeprovision.String() {
			// only allow to chnage the status to "deleted" if the cluster is already in "deprovision" status
			if status != constants.KafkaRequestStatusDeleted {
				return false, errors.GeneralError("failed to update status: cluster is deprovisioning")
			}
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

func (k *kafkaService) ChangeKafkaCNAMErecords(kafkaRequest *api.KafkaRequest, clusterDNS string, action string) (*route53.ChangeResourceRecordSetsOutput, *errors.ServiceError) {
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
			// Currently ignored
			Capacity: managedkafka.Capacity{
				IngressEgressThroughputPerSec: kafkaConfig.KafkaCapacity.IngressEgressThroughputPerSec,
				TotalMaxConnections:           kafkaConfig.KafkaCapacity.TotalMaxConnections,
				MaxDataRetentionSize:          kafkaConfig.KafkaCapacity.MaxDataRetentionSize,
				MaxPartitions:                 kafkaConfig.KafkaCapacity.MaxPartitions,
				MaxDataRetentionPeriod:        kafkaConfig.KafkaCapacity.MaxDataRetentionPeriod,
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
				Kafka: kafkaRequest.Version,
				//TODO: we should remove the strimzi version as it should not be specified here
				Strimzi: "0.21.1",
			},
			Deleted: kafkaRequest.Status == constants.KafkaRequestStatusDeprovision.String(),
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
			CustomClaimCheck:       syncsetresources.BuildCustomClaimCheck(kafkaRequest),
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
