package server

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/data/generated/connector"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/data/generated/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"net"
	"net/http"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"

	_ "github.com/auth0/go-jwt-middleware"
	_ "github.com/dgrijalva/jwt-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/server/logging"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
)

const (
	version                        = "v1"
	apiEndpoint                    = "/api"
	kafkasFleetManagementApiPrefix = "kafkas_mgmt"
	oldManagedServicesApiPrefix    = "managed-services-api"
)

type apiServer struct {
	httpServer *http.Server
}

var _ Server = &apiServer{}

func env() *environments.Env {
	return environments.Environment()
}

func NewAPIServer() Server {
	s := &apiServer{}

	// mainRouter is top level "/"
	mainRouter := mux.NewRouter()
	mainRouter.NotFoundHandler = http.HandlerFunc(api.SendNotFound)
	mainRouter.MethodNotAllowedHandler = http.HandlerFunc(api.SendMethodNotAllowed)

	// Top-level middlewares

	// Sentryhttp middleware performs two operations:
	// 1) Attaches an instance of *sentry.Hub to the request’s context. Accessit by using the sentry.GetHubFromContext() method on the request
	//   NOTE this is the only way middleware, handlers, and services should be reporting to sentry, through the hub
	// 2) Reports panics to the configured sentry service
	sentryhttpOptions := sentryhttp.Options{
		Repanic:         true,
		WaitForDelivery: false,
		Timeout:         env().Config.Sentry.Timeout,
	}
	sentryMW := sentryhttp.New(sentryhttpOptions)
	mainRouter.Use(sentryMW.Handle)

	// Operation ID middleware sets a relatively unique operation ID in the context of each request for debugging purposes
	mainRouter.Use(logger.OperationIDMiddleware)

	// Request logging middleware logs pertinent information about the request and response
	mainRouter.Use(logging.RequestLoggingMiddleware)

	// build current base path /api/kafkas_mgmt
	currentBasePath := fmt.Sprintf("%s/%s", apiEndpoint, kafkasFleetManagementApiPrefix)
	s.buildApiBaseRouter(mainRouter, currentBasePath, "kas-fleet-manager.yaml")

	// build old base path /api/managed-services-api
	managedServicesApiBasePath := fmt.Sprintf("%s/%s", apiEndpoint, oldManagedServicesApiPrefix)
	s.buildApiBaseRouter(mainRouter, managedServicesApiBasePath, "managed-services-api-deprecated.yaml")

	// referring to the router as type http.Handler allows us to add middleware via more handlers
	var mainHandler http.Handler = mainRouter

	if env().Config.Server.EnableJWT {
		authnLogger, err := sdk.NewGlogLoggerBuilder().
			InfoV(glog.Level(1)).
			DebugV(glog.Level(5)).
			Build()
		check(err, "Unable to create authentication logger")

		mainHandler, err = authentication.NewHandler().
			Logger(authnLogger).
			KeysURL(env().Config.Server.JwksURL).                      //ocm JWK JSON web token signing certificates URL
			KeysFile(env().Config.Server.JwksFile).                    //ocm JWK backup JSON web token signing certificates
			KeysURL(env().Config.Keycloak.KafkaRealm.JwksEndpointURI). // mas-sso JWK Cert URL
			Error(fmt.Sprint(errors.ErrorUnauthenticated)).
			Service(errors.ERROR_CODE_PREFIX).
			Public(fmt.Sprintf("^%s/%s/?$", apiEndpoint, kafkasFleetManagementApiPrefix)).
			Public(fmt.Sprintf("^%s/%s/%s/?$", apiEndpoint, kafkasFleetManagementApiPrefix, version)).
			Public(fmt.Sprintf("^%s/%s/%s/openapi/?$", apiEndpoint, kafkasFleetManagementApiPrefix, version)).
			// TODO remove this as it is temporary code to ensure api backward compatibility
			Public(fmt.Sprintf("^%s/%s/?$", apiEndpoint, oldManagedServicesApiPrefix)).
			Public(fmt.Sprintf("^%s/%s/%s/?$", apiEndpoint, oldManagedServicesApiPrefix, version)).
			Public(fmt.Sprintf("^%s/%s/%s/openapi/?$", apiEndpoint, oldManagedServicesApiPrefix, version)).
			// END TODO
			Next(mainHandler).
			Build()
		check(err, "Unable to create authentication handler")
	}

	mainHandler = gorillahandlers.CORS(
		gorillahandlers.AllowedMethods([]string{
			http.MethodDelete,
			http.MethodGet,
			http.MethodPatch,
			http.MethodPost,
		}),
		gorillahandlers.AllowedHeaders([]string{
			"Authorization",
			"Content-Type",
		}),
		gorillahandlers.MaxAge(int((10 * time.Minute).Seconds())),
	)(mainHandler)

	mainHandler = removeTrailingSlash(mainHandler)

	s.httpServer = &http.Server{
		Addr:    env().Config.Server.BindAddress,
		Handler: mainHandler,
	}

	return s
}

// Serve start the blocking call to Serve.
// Useful for breaking up ListenAndServer (Start) when you require the server to be listening before continuing
func (s apiServer) Serve(listener net.Listener) {
	var err error
	if env().Config.Server.EnableHTTPS {
		// Check https cert and key path path
		if env().Config.Server.HTTPSCertFile == "" || env().Config.Server.HTTPSKeyFile == "" {
			check(
				fmt.Errorf("Unspecified required --https-cert-file, --https-key-file"),
				"Can't start https server",
			)
		}

		// Serve with TLS
		glog.Infof("Serving with TLS at %s", env().Config.Server.BindAddress)
		err = s.httpServer.ServeTLS(listener, env().Config.Server.HTTPSCertFile, env().Config.Server.HTTPSKeyFile)
	} else {
		glog.Infof("Serving without TLS at %s", env().Config.Server.BindAddress)
		err = s.httpServer.Serve(listener)
	}

	// Web server terminated.
	check(err, "Web server terminated with errors")
	glog.Info("Web server terminated")
}

// Listen only start the listener, not the server.
// Useful for breaking up ListenAndServer (Start) when you require the server to be listening before continuing
func (s apiServer) Listen() (listener net.Listener, err error) {
	return net.Listen("tcp", env().Config.Server.BindAddress)
}

// Start listening on the configured port and start the server. This is a convenience wrapper for Listen() and Serve(listener Listener)
func (s apiServer) Start() {
	listener, err := s.Listen()
	if err != nil {
		glog.Fatalf("Unable to start API server: %s", err)
	}
	s.Serve(listener)

	// after the server exits but before the application terminates
	// we need to explicitly close Go's sql connection pool.
	// this needs to be called *exactly* once during an app's lifetime.
	env().DBFactory.Close()
}

func (s apiServer) Stop() error {
	return s.httpServer.Shutdown(context.Background())
}

func (s *apiServer) buildApiBaseRouter(mainRouter *mux.Router, basePath string, openApiFilePath string) {
	openAPIDefinitions, err := s.loadOpenAPISpec(openapi.Asset, openApiFilePath)
	if err != nil {
		check(err, "Can't load OpenAPI specification")
	}

	services := &env().Services
	kafkaHandler := handlers.NewKafkaHandler(services.Kafka, services.Config)
	cloudProvidersHandler := handlers.NewCloudProviderHandler(services.CloudProviders, services.Config)
	errorsHandler := handlers.NewErrorsHandler()
	serviceAccountsHandler := handlers.NewServiceAccountHandler(services.Keycloak)
	metricsHandler := handlers.NewMetricsHandler(services.Observatorium)
	serviceStatusHandler := handlers.NewServiceStatusHandler(services.Kafka, services.Config)

	authorizeMiddleware := acl.NewAccessControlListMiddleware(env().Services.Config).Authorize
	ocmAuthzMiddleware := auth.NewOCMAuthorizationMiddleware()
	ocmAuthzMiddlewareRequireIssuer := ocmAuthzMiddleware.RequireIssuer(env().Config.OCM.TokenIssuerURL, errors.ErrorUnauthenticated)
	ocmAuthzMiddlewareRequireTermsAcceptance := ocmAuthzMiddleware.RequireTermsAcceptance(env().Config.Server.EnableTermsAcceptance, ocm.NewClient(env().Clients.OCM.Connection), errors.ErrorTermsNotAccepted)

	// base path. Could be /api/kafkas_mgmt or /api/managed-services-api
	apiRouter := mainRouter.PathPrefix(basePath).Subrouter()

	// /v1
	apiV1Router := apiRouter.PathPrefix("/v1").Subrouter()

	//  /openapi
	apiV1Router.HandleFunc("/openapi", handlers.NewOpenAPIHandler(openAPIDefinitions).Get).Methods(http.MethodGet)

	//  /errors
	apiV1ErrorsRouter := apiV1Router.PathPrefix("/errors").Subrouter()
	apiV1ErrorsRouter.HandleFunc("", errorsHandler.List).Methods(http.MethodGet)
	apiV1ErrorsRouter.HandleFunc("/{id}", errorsHandler.Get).Methods(http.MethodGet)

	// /status
	apiV1Status := apiV1Router.PathPrefix("/status").Subrouter()
	apiV1Status.HandleFunc("", serviceStatusHandler.Get).Methods(http.MethodGet)
	apiV1Status.Use(ocmAuthzMiddlewareRequireIssuer)

	v1Collections := []api.CollectionMetadata{}

	//  /kafkas
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "kafkas",
		Kind: "KafkaList",
	})
	apiV1KafkasRouter := apiV1Router.PathPrefix("/kafkas").Subrouter()
	apiV1KafkasRouter.HandleFunc("/{id}", kafkaHandler.Get).Methods(http.MethodGet)
	apiV1KafkasRouter.HandleFunc("/{id}", kafkaHandler.Delete).Methods(http.MethodDelete)
	apiV1KafkasRouter.HandleFunc("", kafkaHandler.List).Methods(http.MethodGet)
	apiV1KafkasRouter.Use(ocmAuthzMiddlewareRequireIssuer)
	apiV1KafkasRouter.Use(authorizeMiddleware)

	apiV1KafkasCreateRouter := apiV1KafkasRouter.NewRoute().Subrouter()
	apiV1KafkasCreateRouter.HandleFunc("", kafkaHandler.Create).Methods(http.MethodPost)
	apiV1KafkasCreateRouter.Use(ocmAuthzMiddlewareRequireTermsAcceptance)

	//  /serviceaccounts
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "serviceaccounts",
		Kind: "ServiceAccountList",
	})
	apiV1ServiceAccountsRouter := apiV1Router.PathPrefix("/serviceaccounts").Subrouter()
	apiV1ServiceAccountsRouter.HandleFunc("", serviceAccountsHandler.ListServiceAccounts).Methods(http.MethodGet)
	apiV1ServiceAccountsRouter.HandleFunc("", serviceAccountsHandler.CreateServiceAccount).Methods(http.MethodPost)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}", serviceAccountsHandler.DeleteServiceAccount).Methods(http.MethodDelete)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}/reset-credentials", serviceAccountsHandler.ResetServiceAccountCredential).Methods(http.MethodPost)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}", serviceAccountsHandler.GetServiceAccountById).Methods(http.MethodGet)
	apiV1ServiceAccountsRouter.Use(ocmAuthzMiddlewareRequireIssuer)
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

	if env().Config.ConnectorsConfig.Enabled {
		//  /kafka-connector-types

		openAPIDefinitions, err := s.loadOpenAPISpec(connector.Asset, "openapi.yaml")
		if err != nil {
			check(err, "Can't load OpenAPI specification")
		}

		//  /api/connector_mgmt
		apiRouter := mainRouter.PathPrefix("/api/connector_mgmt").Subrouter()

		//  /api/connector_mgmt/v1
		apiV1Router := apiRouter.PathPrefix("/v1").Subrouter()

		//  /api/connector_mgmt/v1/openapi
		apiV1Router.HandleFunc("/openapi", handlers.NewOpenAPIHandler(openAPIDefinitions).Get).Methods(http.MethodGet)

		v1Collections := []api.CollectionMetadata{}

		//  /api/connector_mgmt/v1/kafka-connector-types
		v1Collections = append(v1Collections, api.CollectionMetadata{
			ID:   "kafka-connector-types",
			Kind: "ConnectorTypeList",
		})
		connectorTypesHandler := handlers.NewConnectorTypesHandler(services.ConnectorTypes)
		apiV1ConnectorTypesRouter := apiV1Router.PathPrefix("/kafka-connector-types").Subrouter()
		apiV1ConnectorTypesRouter.HandleFunc("/{connector_type_id}", connectorTypesHandler.Get).Methods(http.MethodGet)
		apiV1ConnectorTypesRouter.HandleFunc("/{connector_type_id}/{path:.*}", connectorTypesHandler.ProxyToExtensionService).Methods(http.MethodGet)
		apiV1ConnectorTypesRouter.HandleFunc("", connectorTypesHandler.List).Methods(http.MethodGet)
		apiV1ConnectorTypesRouter.Use(authorizeMiddleware)

		//  /api/connector_mgmt/v1/kafka-connectors
		v1Collections = append(v1Collections, api.CollectionMetadata{
			ID:   "kafka-connectors",
			Kind: "ConnectorList",
		})
		connectorsHandler := handlers.NewConnectorsHandler(services.Kafka, services.Connectors, services.ConnectorTypes, services.Vault)
		apiV1ConnectorsRouter := apiV1Router.PathPrefix("/kafka-connectors").Subrouter()
		apiV1ConnectorsRouter.HandleFunc("", connectorsHandler.Create).Methods(http.MethodPost)
		apiV1ConnectorsRouter.HandleFunc("", connectorsHandler.List).Methods(http.MethodGet)
		apiV1ConnectorsRouter.HandleFunc("/{connector_id}", connectorsHandler.Get).Methods(http.MethodGet)
		apiV1ConnectorsRouter.HandleFunc("/{connector_id}", connectorsHandler.Patch).Methods(http.MethodPatch)
		apiV1ConnectorsRouter.HandleFunc("/{connector_id}", connectorsHandler.Delete).Methods(http.MethodDelete)
		apiV1ConnectorsRouter.Use(authorizeMiddleware)

		//  /api/connector_mgmt/v1/kafka-connectors-of/{connector_type_id}
		apiV1ConnectorsTypedRouter := apiV1Router.PathPrefix("/kafka-connectors-of/{connector_type_id}").Subrouter()
		apiV1ConnectorsTypedRouter.HandleFunc("", connectorsHandler.Create).Methods(http.MethodPost)
		apiV1ConnectorsTypedRouter.HandleFunc("", connectorsHandler.List).Methods(http.MethodGet)
		apiV1ConnectorsTypedRouter.HandleFunc("/{connector_id}", connectorsHandler.Get).Methods(http.MethodGet)
		apiV1ConnectorsRouter.HandleFunc("/{connector_id}", connectorsHandler.Patch).Methods(http.MethodPatch)
		apiV1ConnectorsRouter.Use(authorizeMiddleware)

		//  /api/connector_mgmt/v1/kafka-connector-clusters
		v1Collections = append(v1Collections, api.CollectionMetadata{
			ID:   "kafka-connector-clusters",
			Kind: "ConnectorClusterList",
		})
		connectorClusterHandler := handlers.NewConnectorClusterHandler(services.SignalBus, services.ConnectorCluster, services.Config, services.Keycloak, services.ConnectorTypes, services.Vault)
		apiV1ConnectorClustersRouter := apiV1Router.PathPrefix("/kafka-connector-clusters").Subrouter()
		apiV1ConnectorClustersRouter.HandleFunc("", connectorClusterHandler.Create).Methods(http.MethodPost)
		apiV1ConnectorClustersRouter.HandleFunc("", connectorClusterHandler.List).Methods(http.MethodGet)
		apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}", connectorClusterHandler.Get).Methods(http.MethodGet)
		apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}", connectorClusterHandler.Delete).Methods(http.MethodDelete)
		apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}/addon-parameters", connectorClusterHandler.GetAddonParameters).Methods(http.MethodGet)
		apiV1ConnectorClustersRouter.Use(authorizeMiddleware)

		// This section adds the API's accessed by the connector agent...
		{
			//  /api/connector_mgmt/v1/kafka-connector-agent-clusters/{id}
			agentRouter := apiV1Router.PathPrefix("/kafka-connector-clusters/{connector_cluster_id}").Subrouter()
			agentRouter.HandleFunc("/status", connectorClusterHandler.UpdateConnectorClusterStatus).Methods(http.MethodPut)
			agentRouter.HandleFunc("/deployments", connectorClusterHandler.ListDeployments).Methods(http.MethodGet)
			agentRouter.HandleFunc("/deployments/{deployment_id}", connectorClusterHandler.GetDeployment).Methods(http.MethodGet)
			agentRouter.HandleFunc("/deployments/{deployment_id}/status", connectorClusterHandler.UpdateDeploymentStatus).Methods(http.MethodPut)
			auth.UseOperatorAuthorisationMiddleware(agentRouter, auth.Connector, env().Services.Keycloak.GetConfig().KafkaRealm.ValidIssuerURI, "connector_cluster_id")
		}

		v1Metadata := api.VersionMetadata{
			ID:          "v1",
			Collections: v1Collections,
		}
		apiMetadata := api.Metadata{
			ID: "connector_mgmt",
			Versions: []api.VersionMetadata{
				v1Metadata,
			},
		}

		apiRouter.HandleFunc("", apiMetadata.ServeHTTP).Methods(http.MethodGet)
		apiV1Router.HandleFunc("", v1Metadata.ServeHTTP).Methods(http.MethodGet)

		apiRouter.Use(MetricsMiddleware)
		apiRouter.Use(db.TransactionMiddleware)
		apiRouter.Use(gorillahandlers.CompressHandler)
	}

	///agent-clusters/{id}
	if env().Config.Kafka.EnableKasFleetshardSync {
		dataPlaneClusterHandler := handlers.NewDataPlaneClusterHandler(services.DataPlaneCluster, services.Config)
		dataPlaneKafkaHandler := handlers.NewDataPlaneKafkaHandler(services.DataPlaneKafkaService, services.Config, services.Kafka)
		apiV1DataPlaneRequestsRouter := apiV1Router.PathPrefix("/agent-clusters").Subrouter()
		apiV1DataPlaneRequestsRouter.HandleFunc("/{id}", dataPlaneClusterHandler.GetDataPlaneClusterConfig).Methods(http.MethodGet)
		apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/status", dataPlaneClusterHandler.UpdateDataPlaneClusterStatus).Methods(http.MethodPut)
		apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/kafkas/status", dataPlaneKafkaHandler.UpdateKafkaStatuses).Methods(http.MethodPut)
		apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/kafkas", dataPlaneKafkaHandler.GetAll).Methods(http.MethodGet)
		// deliberately returns 404 here if the request doesn't have the required role, so that it will appear as if the endpoint doesn't exist
		auth.UseOperatorAuthorisationMiddleware(apiV1DataPlaneRequestsRouter, auth.Kas, env().Services.Keycloak.GetConfig().KafkaRealm.ValidIssuerURI, "id")
	}

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
	apiRouter.Use(MetricsMiddleware)
	apiRouter.Use(db.TransactionMiddleware)
	apiRouter.Use(gorillahandlers.CompressHandler)

	apiV1Router.HandleFunc("", v1Metadata.ServeHTTP).Methods(http.MethodGet)

}

func (s *apiServer) loadOpenAPISpec(assetFunc func(name string) ([]byte, error), asset string) (data []byte, err error) {
	data, err = assetFunc(asset)
	if err != nil {
		err = errors.GeneralError(
			"can't load OpenAPI specification from asset '%s'",
			asset,
		)
		return
	}
	data, err = yaml.YAMLToJSON(data)
	if err != nil {
		err = errors.GeneralError(
			"can't convert OpenAPI specification loaded from asset '%s' from YAML to JSON",
			asset,
		)
		return
	}
	return
}
