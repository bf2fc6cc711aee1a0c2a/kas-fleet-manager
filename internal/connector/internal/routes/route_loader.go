package routes

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/connector/internal/generated"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/connector/internal/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/environments"
	coreHandlers "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/shared"
	"github.com/goava/di"
	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

type options struct {
	di.Inject
	ConnectorsConfig    *config.ConnectorsConfig
	ServerConfig        *server.ServerConfig
	ErrorsHandler       *coreHandlers.ErrorHandler
	AuthorizeMiddleware *acl.AccessControlListMiddleware
	KeycloakService     services.DinosaurKeycloakService

	ConnectorTypesHandler   *handlers.ConnectorTypesHandler
	ConnectorsHandler       *handlers.ConnectorsHandler
	ConnectorClusterHandler *handlers.ConnectorClusterHandler
	DB                      *db.ConnectionFactory
}

func NewRouteLoader(s options) environments.RouteLoader {
	return &s
}

func (s *options) AddRoutes(mainRouter *mux.Router) error {

	openAPIDefinitions, err := shared.LoadOpenAPISpec(generated.Asset, "connector_mgmt.yaml")
	if err != nil {
		return errors.Wrap(err, "Can't load OpenAPI specification")
	}

	//  /api/connector_mgmt
	apiRouter := mainRouter.PathPrefix("/api/connector_mgmt").Subrouter()

	//  /api/connector_mgmt/v1
	apiV1Router := apiRouter.PathPrefix("/v1").Subrouter()

	//  /api/connector_mgmt/v1/openapi
	apiV1Router.HandleFunc("/openapi", coreHandlers.NewOpenAPIHandler(openAPIDefinitions).Get).Methods(http.MethodGet)

	//  /api/connector_mgmt/v1/graphql
	if s.ConnectorsConfig.GraphqlAPIURL != "" {
		graphqlHandler, err := handlers.NewGraphqlHandler(mainRouter, "/api/connector_mgmt/v1/graphql", s.ConnectorsConfig.GraphqlAPIURL)
		if err != nil {
			return errors.Wrap(err, "can't create graphql handler")
		}
		apiV1Router.Handle("/graphql", graphqlHandler.GraphQL).Methods(http.MethodGet, http.MethodPost)
		apiV1Router.Handle("/graphiql", graphqlHandler.GraphIQL).Methods(http.MethodGet)
		apiV1Router.HandleFunc("/schema.graphql", graphqlHandler.GetSchema).Methods(http.MethodGet)
	}

	//  /api/connector_mgmt/v1/errors
	apiV1ErrorsRouter := apiV1Router.PathPrefix("/errors").Subrouter()
	apiV1ErrorsRouter.HandleFunc("", s.ErrorsHandler.List).Methods(http.MethodGet)
	apiV1ErrorsRouter.HandleFunc("/{id}", s.ErrorsHandler.Get).Methods(http.MethodGet)

	v1Collections := []api.CollectionMetadata{}

	//  /api/connector_mgmt/v1/dinosaur_connector_types
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "dinosaur_connector_types",
		Kind: "ConnectorTypeList",
	})
	apiV1ConnectorTypesRouter := apiV1Router.PathPrefix("/{_:dinosaur[-_]connector[-_]types}").Subrouter()
	apiV1ConnectorTypesRouter.HandleFunc("/{connector_type_id}", s.ConnectorTypesHandler.Get).Methods(http.MethodGet)
	apiV1ConnectorTypesRouter.HandleFunc("", s.ConnectorTypesHandler.List).Methods(http.MethodGet)
	apiV1ConnectorTypesRouter.Use(s.AuthorizeMiddleware.Authorize)

	//  /api/connector_mgmt/v1/dinosaur_connectors
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "dinosaur_connectors",
		Kind: "ConnectorList",
	})

	apiV1ConnectorsRouter := apiV1Router.PathPrefix("/{_:dinosaur[-_]connectors}").Subrouter()
	apiV1ConnectorsRouter.HandleFunc("", s.ConnectorsHandler.Create).Methods(http.MethodPost)
	apiV1ConnectorsRouter.HandleFunc("", s.ConnectorsHandler.List).Methods(http.MethodGet)
	apiV1ConnectorsRouter.HandleFunc("/{connector_id}", s.ConnectorsHandler.Get).Methods(http.MethodGet)
	apiV1ConnectorsRouter.HandleFunc("/{connector_id}", s.ConnectorsHandler.Patch).Methods(http.MethodPatch)
	apiV1ConnectorsRouter.HandleFunc("/{connector_id}", s.ConnectorsHandler.Delete).Methods(http.MethodDelete)
	apiV1ConnectorsRouter.Use(s.AuthorizeMiddleware.Authorize)

	//  /api/connector_mgmt/v1/dinosaur_connectors_of/{connector_type_id}
	apiV1ConnectorsTypedRouter := apiV1Router.PathPrefix("/{_:dinosaur[-_]connectors[-_]of}/{connector_type_id}").Subrouter()
	apiV1ConnectorsTypedRouter.HandleFunc("", s.ConnectorsHandler.Create).Methods(http.MethodPost)
	apiV1ConnectorsTypedRouter.HandleFunc("", s.ConnectorsHandler.List).Methods(http.MethodGet)
	apiV1ConnectorsTypedRouter.HandleFunc("/{connector_id}", s.ConnectorsHandler.Get).Methods(http.MethodGet)
	apiV1ConnectorsRouter.HandleFunc("/{connector_id}", s.ConnectorsHandler.Patch).Methods(http.MethodPatch)
	apiV1ConnectorsRouter.Use(s.AuthorizeMiddleware.Authorize)

	//  /api/connector_mgmt/v1/dinosaur_connector_clusters
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "dinosaur_connector_clusters",
		Kind: "ConnectorClusterList",
	})

	apiV1ConnectorClustersRouter := apiV1Router.PathPrefix("/{_:dinosaur[-_]connector[-_]clusters}").Subrouter()
	apiV1ConnectorClustersRouter.HandleFunc("", s.ConnectorClusterHandler.Create).Methods(http.MethodPost)
	apiV1ConnectorClustersRouter.HandleFunc("", s.ConnectorClusterHandler.List).Methods(http.MethodGet)
	apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}", s.ConnectorClusterHandler.Get).Methods(http.MethodGet)
	apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}", s.ConnectorClusterHandler.Update).Methods(http.MethodPut)
	apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}", s.ConnectorClusterHandler.Delete).Methods(http.MethodDelete)
	apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}/{_:addon[-_]parameters}", s.ConnectorClusterHandler.GetAddonParameters).Methods(http.MethodGet)
	apiV1ConnectorClustersRouter.Use(s.AuthorizeMiddleware.Authorize)

	// This section adds the API's accessed by the connector agent...
	{
		//  /api/connector_mgmt/v1/dinosaur_connector_clusters/{id}
		agentRouter := apiV1Router.PathPrefix("/{_:dinosaur[-_]connector[-_]clusters}/{connector_cluster_id}").Subrouter()
		agentRouter.HandleFunc("/status", s.ConnectorClusterHandler.UpdateConnectorClusterStatus).Methods(http.MethodPut)
		agentRouter.HandleFunc("/deployments", s.ConnectorClusterHandler.ListDeployments).Methods(http.MethodGet)
		agentRouter.HandleFunc("/deployments/{deployment_id}", s.ConnectorClusterHandler.GetDeployment).Methods(http.MethodGet)
		agentRouter.HandleFunc("/deployments/{deployment_id}/status", s.ConnectorClusterHandler.UpdateDeploymentStatus).Methods(http.MethodPut)
		auth.UseOperatorAuthorisationMiddleware(agentRouter, auth.Connector, s.KeycloakService.GetConfig().DinosaurRealm.ValidIssuerURI, "connector_cluster_id")
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

	apiRouter.Use(coreHandlers.MetricsMiddleware)
	apiRouter.Use(db.TransactionMiddleware(s.DB))
	apiRouter.Use(gorillaHandlers.CompressHandler)
	return nil
}
