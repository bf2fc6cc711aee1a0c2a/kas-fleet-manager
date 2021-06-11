package server

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/goava/di"
	"net"
	"net/http"
	"time"

	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/authentication"

	_ "github.com/auth0/go-jwt-middleware"
	_ "github.com/dgrijalva/jwt-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/golang/glog"
	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	goerrors "errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/cmd/kas-fleet-manager/server/logging"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/data/generated/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
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

	// Lets load injected routes.
	var routeLoaders []common.RouteLoader
	if err := env().ServiceContainer.Resolve(&routeLoaders); err != nil && !goerrors.Is(err, di.ErrTypeNotExists) {
		check(err, "failed to create the route loaders")
	}
	for _, loader := range routeLoaders {
		check(loader.AddRoutes(mainRouter), "error adding routes")
	}

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
	openAPIDefinitions, err := shared.LoadOpenAPISpec(openapi.Asset, openApiFilePath)
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
	requireIssuer := auth.NewRequireIssuerMiddleware().RequireIssuer(env().Config.OCM.TokenIssuerURL, errors.ErrorUnauthenticated)
	requireTermsAcceptance := auth.NewRequireTermsAcceptanceMiddleware().RequireTermsAcceptance(env().Config.Server.EnableTermsAcceptance, ocm.NewClient(env().Clients.OCM.Connection), errors.ErrorTermsNotAccepted)

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
	apiV1KafkasRouter.HandleFunc("", kafkaHandler.List).Methods(http.MethodGet)
	apiV1KafkasRouter.Use(requireIssuer)
	apiV1KafkasRouter.Use(authorizeMiddleware)

	apiV1KafkasCreateRouter := apiV1KafkasRouter.NewRoute().Subrouter()
	apiV1KafkasCreateRouter.HandleFunc("", kafkaHandler.Create).Methods(http.MethodPost)
	apiV1KafkasCreateRouter.Use(requireTermsAcceptance)

	//  /service_accounts
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "service_accounts",
		Kind: "ServiceAccountList",
	})
	apiV1ServiceAccountsRouter := apiV1Router.PathPrefix("/{_:service[_]?accounts}").Subrouter()
	apiV1ServiceAccountsRouter.HandleFunc("", serviceAccountsHandler.ListServiceAccounts).Methods(http.MethodGet)
	apiV1ServiceAccountsRouter.HandleFunc("", serviceAccountsHandler.CreateServiceAccount).Methods(http.MethodPost)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}", serviceAccountsHandler.DeleteServiceAccount).Methods(http.MethodDelete)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}/{_:reset[-_]credentials}", serviceAccountsHandler.ResetServiceAccountCredential).Methods(http.MethodPost)
	apiV1ServiceAccountsRouter.HandleFunc("/{id}", serviceAccountsHandler.GetServiceAccountById).Methods(http.MethodGet)
	apiV1ServiceAccountsRouter.Use(requireIssuer)
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
	apiRouter.Use(handlers.MetricsMiddleware)
	apiRouter.Use(db.TransactionMiddleware)
	apiRouter.Use(gorillahandlers.CompressHandler)

	apiV1Router.HandleFunc("", v1Metadata.ServeHTTP).Methods(http.MethodGet)

	// /agent_clusters/{id}
	dataPlaneClusterHandler := handlers.NewDataPlaneClusterHandler(services.DataPlaneCluster, services.Config)
	dataPlaneKafkaHandler := handlers.NewDataPlaneKafkaHandler(services.DataPlaneKafkaService, services.Config, services.Kafka)
	apiV1DataPlaneRequestsRouter := apiV1Router.PathPrefix("/{_:agent[-_]clusters}").Subrouter()
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}", dataPlaneClusterHandler.GetDataPlaneClusterConfig).Methods(http.MethodGet)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/status", dataPlaneClusterHandler.UpdateDataPlaneClusterStatus).Methods(http.MethodPut)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/kafkas/status", dataPlaneKafkaHandler.UpdateKafkaStatuses).Methods(http.MethodPut)
	apiV1DataPlaneRequestsRouter.HandleFunc("/{id}/kafkas", dataPlaneKafkaHandler.GetAll).Methods(http.MethodGet)
	// deliberately returns 404 here if the request doesn't have the required role, so that it will appear as if the endpoint doesn't exist
	auth.UseOperatorAuthorisationMiddleware(apiV1DataPlaneRequestsRouter, auth.Kas, env().Services.Keycloak.GetConfig().KafkaRealm.ValidIssuerURI, "id")
}
