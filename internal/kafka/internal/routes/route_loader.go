package routes

import (
	"fmt"
	"net/http"

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
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
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

	OCM                   ocm.Client
	AMS                   ocm.AMSClient
	Kafka                 services.KafkaService
	CloudProviders        services.CloudProvidersService
	Observatorium         services.ObservatoriumService
	Keycloak              coreServices.KafkaKeycloakService
	DataPlaneCluster      services.DataPlaneClusterService
	DataPlaneKafkaService services.DataPlaneKafkaService
	DB                    *db.ConnectionFactory

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

	kafkaHandler := handlers.NewKafkaHandler(s.Kafka, s.ProviderConfig)
	cloudProvidersHandler := handlers.NewCloudProviderHandler(s.CloudProviders, s.ProviderConfig)
	errorsHandler := coreHandlers.NewErrorsHandler()
	serviceAccountsHandler := handlers.NewServiceAccountHandler(s.Keycloak)
	metricsHandler := handlers.NewMetricsHandler(s.Observatorium)
	serviceStatusHandler := handlers.NewServiceStatusHandler(s.Kafka, s.AccessControlListConfig)

	authorizeMiddleware := s.AccessControlListMiddleware.Authorize
	requireOrgID := auth.NewRequireOrgIDMiddleware().RequireOrgID(errors.ErrorUnauthenticated)
	requireIssuer := auth.NewRequireIssuerMiddleware().RequireIssuer(s.OCMConfig.TokenIssuerURL, errors.ErrorUnauthenticated)
	requireTermsAcceptance := auth.NewRequireTermsAcceptanceMiddleware().RequireTermsAcceptance(s.ServerConfig.EnableTermsAcceptance, s.AMS, errors.ErrorTermsNotAccepted)

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

	// /status
	apiV1Status := apiV1Router.PathPrefix("/status").Subrouter()
	apiV1Status.HandleFunc("", serviceStatusHandler.Get).Methods(http.MethodGet)
	apiV1Status.Use(requireIssuer)

	v1Collections := []api.CollectionMetadata{}

	//  /kafkas
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "kafkas",
		Kind: "KafkaList",
	})
	apiV1KafkasRouter := apiV1Router.PathPrefix("/kafkas").Subrouter()
	apiV1KafkasRouter.HandleFunc("/{id}", kafkaHandler.Get).Methods(http.MethodGet)
	apiV1KafkasRouter.HandleFunc("/{id}", kafkaHandler.Delete).Methods(http.MethodDelete)
	apiV1KafkasRouter.HandleFunc("/{id}", kafkaHandler.Update).Methods(http.MethodPatch)
	apiV1KafkasRouter.HandleFunc("", kafkaHandler.List).Methods(http.MethodGet)
	apiV1KafkasRouter.Use(requireIssuer)
	apiV1KafkasRouter.Use(requireOrgID)
	apiV1KafkasRouter.Use(authorizeMiddleware)

	apiV1KafkasCreateRouter := apiV1KafkasRouter.NewRoute().Subrouter()
	apiV1KafkasCreateRouter.HandleFunc("", kafkaHandler.Create).Methods(http.MethodPost)
	apiV1KafkasCreateRouter.Use(requireTermsAcceptance)

	//  /service_accounts
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "service_accounts",
		Kind: "ServiceAccountList",
	})
	apiV1ServiceAccountsRouter := apiV1Router.PathPrefix("/service_accounts").Subrouter()
	apiV1ServiceAccountsRouter.HandleFunc("", serviceAccountsHandler.ListServiceAccounts).Methods(http.MethodGet)
	apiV1ServiceAccountsRouter.HandleFunc("", serviceAccountsHandler.CreateServiceAccount).Methods(http.MethodPost)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}", serviceAccountsHandler.DeleteServiceAccount).Methods(http.MethodDelete)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}/reset_credentials", serviceAccountsHandler.ResetServiceAccountCredential).Methods(http.MethodPost)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}", serviceAccountsHandler.GetServiceAccountById).Methods(http.MethodGet)
	apiV1ServiceAccountsRouter.Use(requireIssuer)
	apiV1ServiceAccountsRouter.Use(requireOrgID)
	apiV1ServiceAccountsRouter.Use(authorizeMiddleware)

	//  /cloud_providers
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "cloud_providers",
		Kind: "CloudProviderList",
	})
	apiV1CloudProvidersRouter := apiV1Router.PathPrefix("/cloud_providers").Subrouter()
	apiV1CloudProvidersRouter.HandleFunc("", cloudProvidersHandler.ListCloudProviders).Methods(http.MethodGet)
	apiV1CloudProvidersRouter.HandleFunc("/{id}/regions", cloudProvidersHandler.ListCloudProviderRegions).Methods(http.MethodGet)

	//  /kafkas/{id}/metrics
	apiV1MetricsRouter := apiV1KafkasRouter.PathPrefix("/{id}/metrics").Subrouter()
	apiV1MetricsRouter.HandleFunc("/query_range", metricsHandler.GetMetricsByRangeQuery).Methods(http.MethodGet)
	apiV1MetricsRouter.HandleFunc("/query", metricsHandler.GetMetricsByInstantQuery).Methods(http.MethodGet)

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

	// /agent_clusters/{id}
	dataPlaneClusterHandler := handlers.NewDataPlaneClusterHandler(s.DataPlaneCluster)
	dataPlaneKafkaHandler := handlers.NewDataPlaneKafkaHandler(s.DataPlaneKafkaService, s.Kafka)
	apiV1DataPlaneRequestsRouter := apiV1Router.PathPrefix("/{_:agent[-_]clusters}").Subrouter()
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}", dataPlaneClusterHandler.GetDataPlaneClusterConfig).Methods(http.MethodGet)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/status", dataPlaneClusterHandler.UpdateDataPlaneClusterStatus).Methods(http.MethodPut)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/kafkas/status", dataPlaneKafkaHandler.UpdateKafkaStatuses).Methods(http.MethodPut)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/kafkas", dataPlaneKafkaHandler.GetAll).Methods(http.MethodGet)
	// deliberately returns 404 here if the request doesn't have the required role, so that it will appear as if the endpoint doesn't exist
	auth.UseOperatorAuthorisationMiddleware(apiV1DataPlaneRequestsRouter, auth.Kas, s.Keycloak.GetConfig().KafkaRealm.ValidIssuerURI, "id")

	adminKafkaHandler := handlers.NewAdminKafkaHandler(s.Kafka, s.ProviderConfig)
	adminRouter := apiV1Router.PathPrefix("/admin").Subrouter()
	rolesMapping := map[string][]string{
		http.MethodGet:    {auth.KasFleetManagerAdminReadRole, auth.KasFleetManagerAdminWriteRole, auth.KasFleetManagerAdminFullRole},
		http.MethodPatch:  {auth.KasFleetManagerAdminWriteRole, auth.KasFleetManagerAdminFullRole},
		http.MethodDelete: {auth.KasFleetManagerAdminFullRole},
	}
	adminRouter.Use(auth.NewRequireIssuerMiddleware().RequireIssuer(s.Keycloak.GetConfig().OSDClusterIDPRealm.ValidIssuerURI, errors.ErrorNotFound))
	adminRouter.Use(auth.NewRolesAuhzMiddleware().RequireRolesForMethods(rolesMapping, errors.ErrorNotFound))
	adminRouter.Use(auth.NewAuditLogMiddleware().AuditLog(errors.ErrorNotFound))
	adminRouter.HandleFunc("/kafkas", adminKafkaHandler.List).Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafkas/{id}", adminKafkaHandler.Get).Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafkas/{id}", adminKafkaHandler.Delete).Methods(http.MethodDelete)
	adminRouter.HandleFunc("/kafkas/{id}", adminKafkaHandler.Update).Methods(http.MethodPatch)

	return nil
}
