package routes

import (
	"fmt"
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/account"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/authorization"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/generated"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/routes"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreHandlers "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/goava/di"
	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	pkgerrors "github.com/pkg/errors"
)

type options struct {
	di.Inject
	ServerConfig   *server.ServerConfig
	OCMConfig      *ocm.OCMConfig
	ProviderConfig *config.ProviderConfig
	KafkaConfig    *config.KafkaConfig

	AMSClient                   ocm.AMSClient
	Kafka                       services.KafkaService
	CloudProviders              services.CloudProvidersService
	Observatorium               services.ObservatoriumService
	Keycloak                    sso.KafkaKeycloakService
	DataPlaneCluster            services.DataPlaneClusterService
	DataPlaneKafkaService       services.DataPlaneKafkaService
	AccountService              account.AccountService
	AuthService                 authorization.Authorization
	DB                          *db.ConnectionFactory
	ClusterPlacementStrategy    services.ClusterPlacementStrategy
	ClusterService              services.ClusterService
	SupportedKafkaInstanceTypes services.SupportedKafkaInstanceTypesService

	AccessControlListMiddleware *acl.AccessControlListMiddleware
	AccessControlListConfig     *acl.AccessControlListConfig
}

func NewRouteLoader(s options) environments.RouteLoader {
	return &s
}

func (s *options) AddRoutes(mainRouter *mux.Router) error {
	basePath := fmt.Sprintf("%s/%s", routes.ApiEndpoint, routes.KafkasFleetManagementApiPrefix)
	err := s.buildApiBaseRouter(mainRouter, basePath, "kas-fleet-manager.yaml")
	if err != nil {
		return err
	}

	return nil
}

func (s *options) buildApiBaseRouter(mainRouter *mux.Router, basePath string, openApiFilePath string) error {
	openAPIDefinitions, err := shared.LoadOpenAPISpec(generated.Asset, openApiFilePath)
	if err != nil {
		return pkgerrors.Wrapf(err, "can't load OpenAPI specification")
	}

	kafkaHandler := handlers.NewKafkaHandler(s.Kafka, s.ProviderConfig, s.AuthService, s.KafkaConfig)
	cloudProvidersHandler := handlers.NewCloudProviderHandler(s.CloudProviders, s.ProviderConfig, s.Kafka, s.ClusterPlacementStrategy, s.KafkaConfig)
	errorsHandler := coreHandlers.NewErrorsHandler()
	serviceAccountsHandler := handlers.NewServiceAccountHandler(s.Keycloak)
	metricsHandler := handlers.NewMetricsHandler(s.Observatorium)
	supportedKafkaInstanceTypesHandler := handlers.NewSupportedKafkaInstanceTypesHandler(s.SupportedKafkaInstanceTypes, s.CloudProviders, s.Kafka, s.KafkaConfig)

	authorizeMiddleware := s.AccessControlListMiddleware.Authorize
	requireOrgID := auth.NewRequireOrgIDMiddleware().RequireOrgID(errors.ErrorUnauthenticated)
	requireIssuer := auth.NewRequireIssuerMiddleware().RequireIssuer([]string{s.ServerConfig.TokenIssuerURL}, errors.ErrorUnauthenticated)
	requireTermsAcceptance := auth.NewRequireTermsAcceptanceMiddleware().RequireTermsAcceptance(s.ServerConfig.EnableTermsAcceptance, s.AMSClient, errors.ErrorTermsNotAccepted)

	// base path. Could be /api/kafkas_mgmt
	apiRouter := mainRouter.PathPrefix(basePath).Subrouter()

	// /v1
	apiV1Router := apiRouter.PathPrefix("/v1").Subrouter()

	//  /openapi
	apiV1Router.HandleFunc("/openapi", coreHandlers.NewOpenAPIHandler(openAPIDefinitions).Get).Methods(http.MethodGet)

	//  /errors
	apiV1ErrorsRouter := apiV1Router.PathPrefix("/errors").Subrouter()
	apiV1ErrorsRouter.HandleFunc("", errorsHandler.List).Methods(http.MethodGet)
	apiV1ErrorsRouter.HandleFunc("/{id}", errorsHandler.Get).Methods(http.MethodGet)

	apiV1SsoProvider := apiV1Router.Path("/sso_providers").Subrouter()
	apiV1SsoProvider.HandleFunc("", serviceAccountsHandler.GetSsoProviders).Methods(http.MethodGet)
	apiV1SsoProvider.Use(authorizeMiddleware)

	v1Collections := []api.CollectionMetadata{}

	//  /kafkas
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "kafkas",
		Kind: "KafkaList",
	})
	apiV1KafkasRouter := apiV1Router.PathPrefix("/kafkas").Subrouter()
	apiV1KafkasRouter.HandleFunc("/{id}", kafkaHandler.Get).
		Name(logger.NewLogEvent("get-kafka", "get a kafka instance").ToString()).
		Methods(http.MethodGet)
	apiV1KafkasRouter.HandleFunc("/{id}", kafkaHandler.Delete).
		Name(logger.NewLogEvent("delete-kafka", "delete a kafka instance").ToString()).
		Methods(http.MethodDelete)
	apiV1KafkasRouter.HandleFunc("/{id}", kafkaHandler.Update).
		Name(logger.NewLogEvent("update-kafka", "update a kafka instance").ToString()).
		Methods(http.MethodPatch)
	apiV1KafkasRouter.HandleFunc("", kafkaHandler.List).
		Name(logger.NewLogEvent("list-kafka", "list all kafkas").ToString()).
		Methods(http.MethodGet)
	apiV1KafkasRouter.Use(requireIssuer)
	apiV1KafkasRouter.Use(requireOrgID)
	apiV1KafkasRouter.Use(authorizeMiddleware)

	apiV1KafkasCreateRouter := apiV1KafkasRouter.NewRoute().Subrouter()
	apiV1KafkasCreateRouter.HandleFunc("", kafkaHandler.Create).Methods(http.MethodPost)
	apiV1KafkasCreateRouter.Use(requireTermsAcceptance)

	//  /kafkas/{id}/metrics
	apiV1MetricsRouter := apiV1KafkasRouter.PathPrefix("/{id}/metrics").Subrouter()
	apiV1MetricsRouter.HandleFunc("/query_range", metricsHandler.GetMetricsByRangeQuery).
		Name(logger.NewLogEvent("get-metrics", "list metrics by range").ToString()).
		Methods(http.MethodGet)
	apiV1MetricsRouter.HandleFunc("/query", metricsHandler.GetMetricsByInstantQuery).
		Name(logger.NewLogEvent("get-metrics-instant", "get metrics by instant").ToString()).
		Methods(http.MethodGet)

	// /kafkas/{id}/metrics/federate
	// federate endpoint separated from the rest of the /kafkas endpoints as it needs to support auth from both sso.redhat.com and mas-sso
	// NOTE: this is only a temporary solution. MAS SSO auth support should be removed once we migrate to sso.redhat.com (TODO: to be done as part of MGDSTRM-6159)
	apiV1MetricsFederateRouter := apiV1Router.PathPrefix("/kafkas/{id}/metrics/federate").Subrouter()
	apiV1MetricsFederateRouter.HandleFunc("", metricsHandler.FederateMetrics).
		Name(logger.NewLogEvent("get-federate-metrics", "get federate metrics by id").ToString()).
		Methods(http.MethodGet)
	apiV1MetricsFederateRouter.Use(auth.NewRequireIssuerMiddleware().RequireIssuer([]string{s.ServerConfig.TokenIssuerURL, s.Keycloak.GetRealmConfig().ValidIssuerURI}, errors.ErrorUnauthenticated))
	apiV1MetricsFederateRouter.Use(requireOrgID)
	apiV1MetricsFederateRouter.Use(authorizeMiddleware)

	//  /service_accounts
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "service_accounts",
		Kind: "ServiceAccountList",
	})
	apiV1ServiceAccountsRouter := apiV1Router.PathPrefix("/service_accounts").Subrouter()
	apiV1ServiceAccountsRouter.
		Path("").
		Queries("client_id", "").
		HandlerFunc(serviceAccountsHandler.GetServiceAccountByClientId).
		Name(logger.NewLogEvent("get-service-accounts", "get a service account by clientId").ToString()).
		Methods(http.MethodGet)

	apiV1ServiceAccountsRouter.HandleFunc("", serviceAccountsHandler.ListServiceAccounts).
		Name(logger.NewLogEvent("list-service-accounts", "lists all service accounts").ToString()).
		Methods(http.MethodGet)
	apiV1ServiceAccountsRouter.HandleFunc("", serviceAccountsHandler.CreateServiceAccount).
		Name(logger.NewLogEvent("create-service-accounts", "create a service accounts").ToString()).
		Methods(http.MethodPost)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}", serviceAccountsHandler.DeleteServiceAccount).
		Name(logger.NewLogEvent("delete-service-accounts", "delete a service accounts").ToString()).
		Methods(http.MethodDelete)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}/reset_credentials", serviceAccountsHandler.ResetServiceAccountCredential).
		Name(logger.NewLogEvent("reset-service-accounts", "reset a service accounts").ToString()).
		Methods(http.MethodPost)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}", serviceAccountsHandler.GetServiceAccountById).
		Name(logger.NewLogEvent("get-service-accounts", "get a service account by id").ToString()).
		Methods(http.MethodGet)

	apiV1ServiceAccountsRouter.Use(requireIssuer)
	apiV1ServiceAccountsRouter.Use(requireOrgID)
	apiV1ServiceAccountsRouter.Use(authorizeMiddleware)

	//  /cloud_providers
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "cloud_providers",
		Kind: "CloudProviderList",
	})
	apiV1CloudProvidersRouter := apiV1Router.PathPrefix("/cloud_providers").Subrouter()
	apiV1CloudProvidersRouter.HandleFunc("", cloudProvidersHandler.ListCloudProviders).
		Name(logger.NewLogEvent("list-cloud-providers", "list all cloud providers").ToString()).
		Methods(http.MethodGet)
	apiV1CloudProvidersRouter.HandleFunc("/{id}/regions", cloudProvidersHandler.ListCloudProviderRegions).
		Name(logger.NewLogEvent("list-regions", "list cloud provider regions").ToString()).
		Methods(http.MethodGet)

	v1Metadata := api.VersionMetadata{
		ID:          "v1",
		Collections: v1Collections,
	}
	apiMetadata := api.Metadata{
		ID: "kafkas_mgmt",
		Versions: []api.VersionMetadata{
			v1Metadata,
		},
	}
	apiRouter.HandleFunc("", apiMetadata.ServeHTTP).Methods(http.MethodGet)
	apiRouter.Use(coreHandlers.MetricsMiddleware)
	apiRouter.Use(db.TransactionMiddleware(s.DB))
	apiRouter.Use(gorillaHandlers.CompressHandler)

	apiV1Router.HandleFunc("", v1Metadata.ServeHTTP).Methods(http.MethodGet)

	// /api/kafkas_mgmt/v1/instance_types/{cloud_provider}/{cloud_region}
	apiV1SupportedKafkaInstanceTypesRouter := apiV1Router.PathPrefix("/instance_types/{cloud_provider}/{cloud_region}").Subrouter()
	apiV1SupportedKafkaInstanceTypesRouter.HandleFunc("", supportedKafkaInstanceTypesHandler.ListSupportedKafkaInstanceTypes).
		Name(logger.NewLogEvent("list-supported-kafka-instance-types-by cloud-region", "list supported kafka instance types by cloud region").ToString()).
		Methods(http.MethodGet)
	apiV1SupportedKafkaInstanceTypesRouter.Use(requireIssuer)
	apiV1SupportedKafkaInstanceTypesRouter.Use(requireOrgID)
	apiV1SupportedKafkaInstanceTypesRouter.Use(authorizeMiddleware)

	// /agent-clusters/{id}
	dataPlaneClusterHandler := handlers.NewDataPlaneClusterHandler(s.DataPlaneCluster)
	dataPlaneKafkaHandler := handlers.NewDataPlaneKafkaHandler(s.DataPlaneKafkaService, s.Kafka)
	apiV1DataPlaneRequestsRouter := apiV1Router.PathPrefix("/agent-clusters").Subrouter()
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}", dataPlaneClusterHandler.GetDataPlaneClusterConfig).
		Name(logger.NewLogEvent("get-dataplane-cluster-config", "get dataplane cluster config by id").ToString()).
		Methods(http.MethodGet)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/status", dataPlaneClusterHandler.UpdateDataPlaneClusterStatus).
		Name(logger.NewLogEvent("update-dataplane-cluster-status", "update dataplane cluster status by id").ToString()).
		Methods(http.MethodPut)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/kafkas/status", dataPlaneKafkaHandler.UpdateKafkaStatuses).
		Name(logger.NewLogEvent("update-dataplane-kafka-status", "update dataplane kafka status by id").ToString()).
		Methods(http.MethodPut)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/kafkas", dataPlaneKafkaHandler.GetAll).
		Name(logger.NewLogEvent("list-dataplane-kafkas", "list all dataplane kafkas").ToString()).
		Methods(http.MethodGet)
	// deliberately returns 404 here if the request doesn't have the required role, so that it will appear as if the endpoint doesn't exist
	auth.UseOperatorAuthorisationMiddleware(apiV1DataPlaneRequestsRouter, s.Keycloak.GetRealmConfig().ValidIssuerURI, "id", s.ClusterService)

	adminKafkaHandler := handlers.NewAdminKafkaHandler(s.Kafka, s.AccountService, s.ProviderConfig)
	adminRouter := apiV1Router.PathPrefix("/admin").Subrouter()
	rolesMapping := map[string][]string{
		http.MethodGet:    {auth.KasFleetManagerAdminReadRole, auth.KasFleetManagerAdminWriteRole, auth.KasFleetManagerAdminFullRole},
		http.MethodPatch:  {auth.KasFleetManagerAdminWriteRole, auth.KasFleetManagerAdminFullRole},
		http.MethodDelete: {auth.KasFleetManagerAdminFullRole},
	}
	adminRouter.Use(auth.NewRequireIssuerMiddleware().RequireIssuer([]string{s.Keycloak.GetConfig().OSDClusterIDPRealm.ValidIssuerURI}, errors.ErrorNotFound))
	adminRouter.Use(auth.NewRolesAuhzMiddleware().RequireRolesForMethods(rolesMapping, errors.ErrorNotFound))
	adminRouter.Use(auth.NewAuditLogMiddleware().AuditLog(errors.ErrorNotFound))
	adminRouter.HandleFunc("/kafkas", adminKafkaHandler.List).
		Name(logger.NewLogEvent("admin-list-kafkas", "[admin] list all kafkas").ToString()).
		Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafkas/{id}", adminKafkaHandler.Get).
		Name(logger.NewLogEvent("admin-get-kafka", "[admin] get kafka by id").ToString()).
		Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafkas/{id}", adminKafkaHandler.Delete).
		Name(logger.NewLogEvent("admin-delete-kafka", "[admin] delete kafka by id").ToString()).
		Methods(http.MethodDelete)
	adminRouter.HandleFunc("/kafkas/{id}", adminKafkaHandler.Update).
		Name(logger.NewLogEvent("admin-update-kafka", "[admin] update kafka by id").ToString()).
		Methods(http.MethodPatch)

	return nil
}
