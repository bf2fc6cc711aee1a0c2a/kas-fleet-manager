package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/keycloak"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/server/logging"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/sentry"
	"github.com/goava/di"

	"github.com/openshift-online/ocm-sdk-go/authentication"

	_ "github.com/auth0/go-jwt-middleware"
	sentryhttp "github.com/getsentry/sentry-go/http"
	_ "github.com/golang-jwt/jwt/v4"
	"github.com/golang/glog"
	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/logger"
)

type ApiServerReadyCondition interface {
	Wait()
}

type ApiServer struct {
	httpServer      *http.Server
	serverConfig    *ServerConfig
	sentryTimeout   time.Duration
	readyConditions []ApiServerReadyCondition
}

var _ Server = &ApiServer{}

type ServerOptions struct {
	di.Inject
	ServerConfig    *ServerConfig
	KeycloakConfig  *keycloak.KeycloakConfig
	SentryConfig    *sentry.Config
	RouteLoaders    []environments.RouteLoader
	Env             *environments.Env
	ReadyConditions []ApiServerReadyCondition `di:"optional"`
}

func NewAPIServer(options ServerOptions) *ApiServer {
	s := &ApiServer{
		httpServer:      nil,
		serverConfig:    options.ServerConfig,
		sentryTimeout:   options.SentryConfig.Timeout,
		readyConditions: options.ReadyConditions,
	}

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
		Timeout:         options.SentryConfig.Timeout,
	}
	sentryMW := sentryhttp.New(sentryhttpOptions)
	mainRouter.Use(sentryMW.Handle)

	// Operation ID middleware sets a relatively unique operation ID in the context of each request for debugging purposes
	mainRouter.Use(logger.OperationIDMiddleware)

	// Request logging middleware logs pertinent information about the request and response
	mainRouter.Use(logging.RequestLoggingMiddleware)

	for _, loader := range options.RouteLoaders {
		check(loader.AddRoutes(mainRouter), "error adding routes", options.SentryConfig.Timeout)
	}

	// referring to the router as type http.Handler allows us to add middleware via more handlers
	var mainHandler http.Handler = mainRouter
	var builder *authentication.HandlerBuilder
	options.Env.MustResolve(&builder)

	var err error
	mainHandler, err = builder.Next(mainHandler).Build()
	check(err, "Unable to create authentication handler", options.SentryConfig.Timeout)

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
		Addr:    options.ServerConfig.BindAddress,
		Handler: mainHandler,
	}

	return s
}

// Serve start the blocking call to Serve.
// Useful for breaking up ListenAndServer (Start) when you require the server to be listening before continuing
func (s *ApiServer) Serve(listener net.Listener) {
	var err error
	if s.serverConfig.EnableHTTPS {
		// Check https cert and key path path
		if s.serverConfig.HTTPSCertFile == "" || s.serverConfig.HTTPSKeyFile == "" {
			check(
				fmt.Errorf("Unspecified required --https-cert-file, --https-key-file"),
				"Can't start https server",
				s.sentryTimeout,
			)
		}

		// Serve with TLS
		glog.Infof("Serving with TLS at %s", s.serverConfig.BindAddress)
		err = s.httpServer.ServeTLS(listener, s.serverConfig.HTTPSCertFile, s.serverConfig.HTTPSKeyFile)
	} else {
		glog.Infof("Serving without TLS at %s", s.serverConfig.BindAddress)
		err = s.httpServer.Serve(listener)
	}

	// Web server terminated.
	check(err, "Web server terminated with errors", s.sentryTimeout)
	glog.Info("Web server terminated")
}

// Listen only start the listener, not the server.
// Useful for breaking up ListenAndServer (Start) when you require the server to be listening before continuing
func (s *ApiServer) Listen() (listener net.Listener, err error) {
	return net.Listen("tcp", s.serverConfig.BindAddress)
}

func (s *ApiServer) Start() {
	go s.Run()
}

// Start listening on the configured port and start the server. This is a convenience wrapper for Listen() and Serve(listener Listener)
func (s *ApiServer) Run() {
	listener, err := s.Listen()
	if err != nil {
		glog.Fatalf("Unable to start API server: %s", err)
	}

	// Before we start processing requests, wait
	// for the server to be ready to run.
	for _, condition := range s.readyConditions {
		condition.Wait()
	}

	s.Serve(listener)
}

func (s *ApiServer) Stop() {
	err := s.httpServer.Shutdown(context.Background())
	if err != nil {
		glog.Warningf("Unable to stop API server: %s", err)
	}
}
