package server

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services/sentry"
	"github.com/golang/glog"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/handlers"
)

func NewMetricsServer(metricsConfig *MetricsConfig, serverConfig *ServerConfig, sentryConfig *sentry.Config) *MetricsServer {
	mainRouter := mux.NewRouter()
	mainRouter.NotFoundHandler = http.HandlerFunc(api.SendNotFound)

	// metrics endpoint
	prometheusMetricsHandler := handlers.NewPrometheusMetricsHandler()
	mainRouter.Handle("/metrics", prometheusMetricsHandler.Handler())

	var mainHandler http.Handler = mainRouter

	s := &MetricsServer{
		serverConfig:  serverConfig,
		sentryTimeout: sentryConfig.Timeout,
		metricsConfig: metricsConfig,
	}
	s.httpServer = &http.Server{
		Addr:    metricsConfig.BindAddress,
		Handler: mainHandler,
	}
	return s
}

type MetricsServer struct {
	httpServer    *http.Server
	serverConfig  *ServerConfig
	metricsConfig *MetricsConfig
	sentryTimeout time.Duration
}

var _ Server = &MetricsServer{}

func (s MetricsServer) Listen() (listener net.Listener, err error) {
	return nil, nil
}

func (s MetricsServer) Serve(listener net.Listener) {
}

func (s MetricsServer) Start() {
	go s.Run()
}

func (s MetricsServer) Run() {
	glog.Infof("start metrics server")
	var err error
	if s.metricsConfig.EnableHTTPS {
		if s.serverConfig.HTTPSCertFile == "" || s.serverConfig.HTTPSKeyFile == "" {
			check(
				fmt.Errorf("Unspecified required --https-cert-file, --https-key-file"),
				"Can't start https server", s.sentryTimeout,
			)
		}

		// Serve with TLS
		glog.Infof("Serving Metrics with TLS at %s", s.serverConfig.BindAddress)
		err = s.httpServer.ListenAndServeTLS(s.serverConfig.HTTPSCertFile, s.serverConfig.HTTPSKeyFile)
	} else {
		glog.Infof("Serving Metrics without TLS at %s", s.metricsConfig.BindAddress)
		err = s.httpServer.ListenAndServe()
	}
	check(err, "Metrics server terminated with errors", s.sentryTimeout)
	glog.Infof("Metrics server terminated")
}

func (s MetricsServer) Stop() {
	err := s.httpServer.Shutdown(context.Background())
	if err != nil {
		glog.Warningf("Unable to stop health check server: %s", err)
	}
}
