package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
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
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/data/generated/openapi"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
)

const (
	managedServicesApi = "managed-services-api"
	version            = "v1"
	apiEndpoint        = "/api"
)

type apiServer struct {
	httpServer *http.Server
}

var _ Server = &apiServer{}

func env() *environments.Env {
	return environments.Environment()
}

func NewAPIServer() Server {
	var err error

	s := &apiServer{}
	services := &env().Services

	openAPIDefinitions, err := s.loadOpenAPISpec("kas-fleet-manager.yaml")
	if err != nil {
		check(err, "Can't load OpenAPI specification")
	}

	kafkaHandler := handlers.NewKafkaHandler(services.Kafka, services.Config)
	cloudProvidersHandler := handlers.NewCloudProviderHandler(services.CloudProviders, services.Config)
	errorsHandler := handlers.NewErrorsHandler()
	serviceAccountsHandler := handlers.NewServiceAccountHandler(services.Keycloak)
	metricsHandler := handlers.NewMetricsHandler(services.Observatorium)

	/* TODO
	var authzMiddleware auth.AuthorizationMiddleware = auth.NewAuthzMiddlewareMock()
	if env().Config.Server.EnableAuthz {
		authzMiddleware = auth.NewAuthzMiddleware(services.AccessReview, env().Config.Authz.AuthzRules)
	}
	*/

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

	//  /api/managed-services-api
	apiRouter := mainRouter.PathPrefix("/api/managed-services-api").Subrouter()
	apiRouter.HandleFunc("", api.SendAPI).Methods(http.MethodGet)
	apiRouter.Use(MetricsMiddleware)
	apiRouter.Use(db.TransactionMiddleware)
	apiRouter.Use(gorillahandlers.CompressHandler)

	//  /api/managed-services-api/v1
	apiV1Router := apiRouter.PathPrefix("/v1").Subrouter()
	apiV1Router.HandleFunc("", api.SendAPIV1).Methods(http.MethodGet)
	apiV1Router.HandleFunc("/", api.SendAPIV1).Methods(http.MethodGet)

	//  /api/managed-services-api/v1/openapi
	apiV1Router.HandleFunc("/openapi", handlers.NewOpenAPIHandler(openAPIDefinitions).Get).Methods(http.MethodGet)

	//  /api/managed-services-api/v1/errors
	apiV1ErrorsRouter := apiV1Router.PathPrefix("/errors").Subrouter()
	apiV1ErrorsRouter.HandleFunc("", errorsHandler.List).Methods(http.MethodGet)
	apiV1ErrorsRouter.HandleFunc("/{id}", errorsHandler.Get).Methods(http.MethodGet)

	//  /api/managed-services-api/v1/kafkas
	apiV1KafkasRouter := apiV1Router.PathPrefix("/kafkas").Subrouter()
	apiV1KafkasRouter.HandleFunc("/{id}", kafkaHandler.Get).Methods(http.MethodGet)
	apiV1KafkasRouter.HandleFunc("/{id}", kafkaHandler.Delete).Methods(http.MethodDelete)
	apiV1KafkasRouter.HandleFunc("", kafkaHandler.Create).Methods(http.MethodPost)
	apiV1KafkasRouter.HandleFunc("", kafkaHandler.List).Methods(http.MethodGet)

	if env().Services.Config.IsAllowListEnabled() {
		apiV1KafkasRouter.Use(acl.NewAllowListMiddleware(env().Services.Config).Authorize)
	}

	//  /api/managed-services-api/v1/cloud_providers
	apiV1CloudProvidersRouter := apiV1Router.PathPrefix("/cloud_providers").Subrouter()
	apiV1CloudProvidersRouter.HandleFunc("", cloudProvidersHandler.ListCloudProviders).Methods(http.MethodGet)
	apiV1CloudProvidersRouter.HandleFunc("/{id}/regions", cloudProvidersHandler.ListCloudProviderRegions).Methods(http.MethodGet)

	apiV1ServiceAccountsRouter := apiV1Router.PathPrefix("/serviceaccounts").Subrouter()
	apiV1ServiceAccountsRouter.HandleFunc("", serviceAccountsHandler.ListServiceAccounts).Methods(http.MethodGet)
	apiV1ServiceAccountsRouter.HandleFunc("", serviceAccountsHandler.CreateServiceAccount).Methods(http.MethodPost)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}", serviceAccountsHandler.DeleteServiceAccount).Methods(http.MethodDelete)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}/reset-credentials", serviceAccountsHandler.ResetServiceAccountCredential).Methods(http.MethodPost)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}", serviceAccountsHandler.GetServiceAccountById).Methods(http.MethodGet)

	//  /api/managed-services-api/v1/kafkas/{id}/metrics
	apiV1MetricsRouter := apiV1KafkasRouter.PathPrefix("/{id}/metrics").Subrouter()
	apiV1MetricsRouter.HandleFunc("/query_range", metricsHandler.GetMetricsByRangeQuery).Methods(http.MethodGet)
	apiV1MetricsRouter.HandleFunc("/query", metricsHandler.GetMetricsByInstantQuery).Methods(http.MethodGet)

	if env().Config.ConnectorsConfig.Enabled {

		//  /api/managed-services-api/v1/connector-types
		connectorTypesHandler := handlers.NewConnectorTypesHandler(services.ConnectorTypes)
		apiV1ConnectorTypesRouter := apiV1Router.PathPrefix("/connector-types").Subrouter()
		apiV1ConnectorTypesRouter.HandleFunc("/{id}", connectorTypesHandler.Get).Methods(http.MethodGet)
		apiV1ConnectorTypesRouter.HandleFunc("/{id}/{path:.*}", connectorTypesHandler.ProxyToExtensionService).Methods(http.MethodGet)
		apiV1ConnectorTypesRouter.HandleFunc("", connectorTypesHandler.List).Methods(http.MethodGet)

		//  /api/managed-services-api/v1/kafkas/{id}/connector-deployments
		connectorsHandler := handlers.NewConnectorsHandler(services.Kafka, services.Connectors, services.ConnectorTypes)
		apiV1ConnectorsRouter := apiV1KafkasRouter.PathPrefix("/{id}/connector-deployments").Subrouter()
		apiV1ConnectorsRouter.HandleFunc("/{cid}", connectorsHandler.Get).Methods(http.MethodGet)
		apiV1ConnectorsRouter.HandleFunc("/{cid}", connectorsHandler.Delete).Methods(http.MethodDelete)
		apiV1ConnectorsRouter.HandleFunc("", connectorsHandler.Create).Methods(http.MethodPost)
		apiV1ConnectorsRouter.HandleFunc("", connectorsHandler.List).Methods(http.MethodGet)

		//  /api/managed-services-api/v1/kafkas/{id}/connector-deployments-of/{tid}
		apiV1ConnectorsTypedRouter := apiV1KafkasRouter.PathPrefix("/{id}/connector-deployments-of/{tid}").Subrouter()
		apiV1ConnectorsTypedRouter.HandleFunc("/{cid}", connectorsHandler.Get).Methods(http.MethodGet)
		apiV1ConnectorsTypedRouter.HandleFunc("", connectorsHandler.Create).Methods(http.MethodPost)
		apiV1ConnectorsTypedRouter.HandleFunc("", connectorsHandler.List).Methods(http.MethodGet)
	}

	///api/managed-services-api/v1/agent-clusters/{id}
	if env().Config.ClusterCreationConfig.EnableKasFleetshardOperator {
		dataPlaneClusterHandler := handlers.NewDataPlaneClusterHandler(services.DataPlaneCluster, services.Config)
		dataPlaneKafkaHandler := handlers.NewDataPlaneKafkaHandler(services.DataPlaneKafkaService, services.Config)
		apiV1DataPlaneRequestsRouter := apiV1Router.PathPrefix("/agent-clusters").Subrouter()
		apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/status", dataPlaneClusterHandler.UpdateDataPlaneClusterStatus).Methods(http.MethodPut)
		apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/kafkas/status", dataPlaneKafkaHandler.UpdateKafkaStatuses).Methods(http.MethodPut)
		rolesAuthzMiddleware := auth.NewRolesAuhzMiddleware()
		dataPlaneAuthzMiddleware := auth.NewDataPlaneAuthzMiddleware()
		// deliberately returns 404 here if the request doesn't have the required role, so that it will appear as if the endpoint doesn't exist
		apiV1DataPlaneRequestsRouter.Use(rolesAuthzMiddleware.RequireRealmRole("kas_fleetshard_operator", errors.ErrorNotFound), dataPlaneAuthzMiddleware.CheckClusterId)
	}

	// referring to the router as type http.Handler allows us to add middleware via more handlers
	var mainHandler http.Handler = mainRouter

	if env().Config.Server.EnableJWT {
		mainHandler = auth.SetClaimsInContextHandler(mainHandler)
		authnLogger, err := sdk.NewGlogLoggerBuilder().
			InfoV(glog.Level(1)).
			DebugV(glog.Level(5)).
			Build()
		check(err, "Unable to create authentication logger")

		mainHandler, err = authentication.NewHandler().
			Logger(authnLogger).
			KeysURL(env().Config.Server.JwkCertURL).
			Error(fmt.Sprint(errors.ErrorUnauthenticated)).
			Service(errors.ERROR_CODE_PREFIX).
			Public(fmt.Sprintf("^%s/%s/?$", apiEndpoint, managedServicesApi)).
			Public(fmt.Sprintf("^%s/%s/%s/?$", apiEndpoint, managedServicesApi, version)).
			Public(fmt.Sprintf("^%s/%s/%s/openapi/?$", apiEndpoint, managedServicesApi, version)).
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

func (s *apiServer) loadOpenAPISpec(asset string) (data []byte, err error) {
	data, err = openapi.Asset(asset)
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
