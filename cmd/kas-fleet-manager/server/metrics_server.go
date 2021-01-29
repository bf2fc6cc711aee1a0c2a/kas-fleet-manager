package server

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"net"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
)

func NewMetricsServer() Server {
	mainRouter := mux.NewRouter()
	mainRouter.NotFoundHandler = http.HandlerFunc(api.SendNotFound)

	// metrics endpoint
	prometheusMetricsHandler := handlers.NewPrometheusMetricsHandler()
	mainRouter.Handle("/metrics", prometheusMetricsHandler.Handler())

	var mainHandler http.Handler = mainRouter

	s := &metricsServer{}
	s.httpServer = &http.Server{
		Addr:    env().Config.Metrics.BindAddress,
		Handler: mainHandler,
	}
	return s
}

type metricsServer struct {
	httpServer *http.Server
}

var _ Server = &metricsServer{}

func (s metricsServer) Listen() (listener net.Listener, err error) {
	return nil, nil
}

func (s metricsServer) Serve(listener net.Listener) {
}

func (s metricsServer) Start() {
	glog.Infof("start metrics server")
	var err error
	if env().Config.Metrics.EnableHTTPS {
		if env().Config.Server.HTTPSCertFile == "" || env().Config.Server.HTTPSKeyFile == "" {
			check(
				fmt.Errorf("Unspecified required --https-cert-file, --https-key-file"),
				"Can't start https server",
			)
		}

		// Serve with TLS
		glog.Infof("Serving Metrics with TLS at %s", env().Config.Server.BindAddress)
		err = s.httpServer.ListenAndServeTLS(env().Config.Server.HTTPSCertFile, env().Config.Server.HTTPSKeyFile)
	} else {
		glog.Infof("Serving Metrics without TLS at %s", env().Config.Metrics.BindAddress)
		err = s.httpServer.ListenAndServe()
	}
	check(err, "Metrics server terminated with errors")
	glog.Infof("Metrics server terminated")
}

func (s metricsServer) Stop() error {
	return s.httpServer.Shutdown(context.Background())
}
