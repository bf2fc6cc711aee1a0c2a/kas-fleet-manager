package routes

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/data/generated/connector"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/common"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	corehandlers "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/goava/di"
	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"net/http"
)

type options struct {
	di.Inject
	ConnectorsConfig    *config.ConnectorsConfig
	ErrorsHandler       *corehandlers.ErrorHandler
	AuthorizeMiddleware *acl.AccessControlListMiddleware
	KeycloakService     services.KafkaKeycloakService

	ConnectorTypesHandler   *handlers.ConnectorTypesHandler
	ConnectorsHandler       *handlers.ConnectorsHandler
	ConnectorClusterHandler *handlers.ConnectorClusterHandler
}

func NewRouteLoader(s options) common.RouteLoader {
	return &s
}

func (s *options) AddRoutes(mainRouter *mux.Router) error {
	if s.ConnectorsConfig.Enabled {

		openAPIDefinitions, err := shared.LoadOpenAPISpec(connector.Asset, "openapi.yaml")
		if err != nil {
			return errors.Wrap(err, "Can't load OpenAPI specification")
		}

		//  /api/connector_mgmt
		apiRouter := mainRouter.PathPrefix("/api/connector_mgmt").Subrouter()

		//  /api/connector_mgmt/v1
		apiV1Router := apiRouter.PathPrefix("/v1").Subrouter()

		//  /api/connector_mgmt/v1/openapi
		apiV1Router.HandleFunc("/openapi", corehandlers.NewOpenAPIHandler(openAPIDefinitions).Get).Methods(http.MethodGet)

		//  /api/connector_mgmt/v1/errors
		apiV1ErrorsRouter := apiV1Router.PathPrefix("/errors").Subrouter()
		apiV1ErrorsRouter.HandleFunc("", s.ErrorsHandler.List).Methods(http.MethodGet)
		apiV1ErrorsRouter.HandleFunc("/{id}", s.ErrorsHandler.Get).Methods(http.MethodGet)

		v1Collections := []api.CollectionMetadata{}

		//  /api/connector_mgmt/v1/kafka_connector_types
		v1Collections = append(v1Collections, api.CollectionMetadata{
			ID:   "kafka_connector_types",
			Kind: "ConnectorTypeList",
		})
		apiV1ConnectorTypesRouter := apiV1Router.PathPrefix("/{_:kafka[-_]connector[-_]types}").Subrouter()
		apiV1ConnectorTypesRouter.HandleFunc("/{connector_type_id}", s.ConnectorTypesHandler.Get).Methods(http.MethodGet)
		apiV1ConnectorTypesRouter.HandleFunc("", s.ConnectorTypesHandler.List).Methods(http.MethodGet)
		apiV1ConnectorTypesRouter.Use(s.AuthorizeMiddleware.Authorize)

		//  /api/connector_mgmt/v1/kafka_connectors
		v1Collections = append(v1Collections, api.CollectionMetadata{
			ID:   "kafka_connectors",
			Kind: "ConnectorList",
		})

		apiV1ConnectorsRouter := apiV1Router.PathPrefix("/{_:kafka[-_]connectors}").Subrouter()
		apiV1ConnectorsRouter.HandleFunc("", s.ConnectorsHandler.Create).Methods(http.MethodPost)
		apiV1ConnectorsRouter.HandleFunc("", s.ConnectorsHandler.List).Methods(http.MethodGet)
		apiV1ConnectorsRouter.HandleFunc("/{connector_id}", s.ConnectorsHandler.Get).Methods(http.MethodGet)
		apiV1ConnectorsRouter.HandleFunc("/{connector_id}", s.ConnectorsHandler.Patch).Methods(http.MethodPatch)
		apiV1ConnectorsRouter.HandleFunc("/{connector_id}", s.ConnectorsHandler.Delete).Methods(http.MethodDelete)
		apiV1ConnectorsRouter.Use(s.AuthorizeMiddleware.Authorize)

		//  /api/connector_mgmt/v1/kafka_connectors_of/{connector_type_id}
		apiV1ConnectorsTypedRouter := apiV1Router.PathPrefix("/{_:kafka[-_]connectors[-_]of}/{connector_type_id}").Subrouter()
		apiV1ConnectorsTypedRouter.HandleFunc("", s.ConnectorsHandler.Create).Methods(http.MethodPost)
		apiV1ConnectorsTypedRouter.HandleFunc("", s.ConnectorsHandler.List).Methods(http.MethodGet)
		apiV1ConnectorsTypedRouter.HandleFunc("/{connector_id}", s.ConnectorsHandler.Get).Methods(http.MethodGet)
		apiV1ConnectorsRouter.HandleFunc("/{connector_id}", s.ConnectorsHandler.Patch).Methods(http.MethodPatch)
		apiV1ConnectorsRouter.Use(s.AuthorizeMiddleware.Authorize)

		//  /api/connector_mgmt/v1/kafka_connector_clusters
		v1Collections = append(v1Collections, api.CollectionMetadata{
			ID:   "kafka_connector_clusters",
			Kind: "ConnectorClusterList",
		})

		apiV1ConnectorClustersRouter := apiV1Router.PathPrefix("/{_:kafka[-_]connector[-_]clusters}").Subrouter()
		apiV1ConnectorClustersRouter.HandleFunc("", s.ConnectorClusterHandler.Create).Methods(http.MethodPost)
		apiV1ConnectorClustersRouter.HandleFunc("", s.ConnectorClusterHandler.List).Methods(http.MethodGet)
		apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}", s.ConnectorClusterHandler.Get).Methods(http.MethodGet)
		apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}", s.ConnectorClusterHandler.Delete).Methods(http.MethodDelete)
		apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}/{_:addon[-_]parameters}", s.ConnectorClusterHandler.GetAddonParameters).Methods(http.MethodGet)
		apiV1ConnectorClustersRouter.Use(s.AuthorizeMiddleware.Authorize)

		// This section adds the API's accessed by the connector agent...
		{
			//  /api/connector_mgmt/v1/kafka_connector_clusters/{id}
			agentRouter := apiV1Router.PathPrefix("/{_:kafka[-_]connector[-_]clusters}/{connector_cluster_id}").Subrouter()
			agentRouter.HandleFunc("/status", s.ConnectorClusterHandler.UpdateConnectorClusterStatus).Methods(http.MethodPut)
			agentRouter.HandleFunc("/deployments", s.ConnectorClusterHandler.ListDeployments).Methods(http.MethodGet)
			agentRouter.HandleFunc("/deployments/{deployment_id}", s.ConnectorClusterHandler.GetDeployment).Methods(http.MethodGet)
			agentRouter.HandleFunc("/deployments/{deployment_id}/status", s.ConnectorClusterHandler.UpdateDeploymentStatus).Methods(http.MethodPut)
			auth.UseOperatorAuthorisationMiddleware(agentRouter, auth.Connector, s.KeycloakService.GetConfig().KafkaRealm.ValidIssuerURI, "connector_cluster_id")
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

		apiRouter.Use(corehandlers.MetricsMiddleware)
		apiRouter.Use(db.TransactionMiddleware)
		apiRouter.Use(gorillahandlers.CompressHandler)
	}
	return nil
}
