package integration

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	ocm2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/goava/di"
	"github.com/golang/glog"
	"net/http/httptest"
	"testing"
)

type Services struct {
	di.Inject
	DBFactory             *db.ConnectionFactory
	AppConfig             *config.ApplicationConfig
	MetricsServer         *server.MetricsServer
	HealthCheckServer     *server.HealthCheckServer
	Workers               []workers.Worker
	LeaderElectionManager *workers.LeaderElectionManager
	SignalBus             signalbus.SignalBus
	APIServer             *server.ApiServer
	BootupServices        []provider.BootService
	CloudProvidersService services.CloudProvidersService
	ClusterService        services.ClusterService
	OCM2Client            *ocm2.Client
	OCMConfig             *config.OCMConfig
	KafkaService          services.KafkaService
	ObservatoriumClient   *observatorium.Client
	ClusterManager        *workers.ClusterManager
	ServerConfig          *config.ServerConfig
}

var testServices Services

// Register a test
// This should be run before every integration test
func NewKafkaHelper(t *testing.T, server *httptest.Server) (*test.Helper, *openapi.APIClient, func()) {
	return NewKafkaHelperWithHooks(t, server, nil)
}

func NewKafkaHelperWithHooks(t *testing.T, server *httptest.Server, configurationHook interface{}) (*test.Helper, *openapi.APIClient, func()) {
	h, c, teardown := test.NewHelperWithHooks(t, server, configurationHook, kafka.ConfigProviders())
	if err := h.Env.ServiceContainer.Resolve(&testServices); err != nil {
		glog.Fatalf("Unable to initialize testing environment: %s", err.Error())
	}
	return h, c, teardown
}
