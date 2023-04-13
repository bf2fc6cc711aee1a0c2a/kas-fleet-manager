package routes

import (
	"net/http"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/sso"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/acl"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	kerrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreHandlers "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/server"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/goava/di"
	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"

	openapicontents "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/openapi"
)

type options struct {
	di.Inject
	ConnectorsConfig          *config.ConnectorsConfig
	ProcessorsConfig          *config.ProcessorsConfig
	ServerConfig              *server.ServerConfig
	ErrorsHandler             *coreHandlers.ErrorHandler
	AuthorizeMiddleware       *acl.AccessControlListMiddleware
	KeycloakService           sso.KafkaKeycloakService
	AuthAgentService          auth.AuthAgentService
	ConnectorAdminHandler     *handlers.ConnectorAdminHandler
	ConnectorTypesHandler     *handlers.ConnectorTypesHandler
	ConnectorsHandler         *handlers.ConnectorsHandler
	ConnectorClusterHandler   *handlers.ConnectorClusterHandler
	ConnectorNamespaceHandler *handlers.ConnectorNamespaceHandler
	ProcessorsHandler         *handlers.ProcessorsHandler
	DB                        *db.ConnectionFactory
	AdminRoleAuthZConfig      *auth.AdminRoleAuthZConfig
}

func NewRouteLoader(s options) environments.RouteLoader {
	return &s
}

func (s *options) AddRoutes(mainRouter *mux.Router) error {

	authorizeMiddleware := s.AuthorizeMiddleware.Authorize
	requireOrgID := auth.NewRequireOrgIDMiddleware().RequireOrgID(kerrors.ErrorUnauthenticated)

	openAPIDefinitions, err := shared.LoadOpenAPISpecFromYAML(openapicontents.ConnectorMgmtOpenAPIYAMLBytes())
	if err != nil {
		return errors.Wrap(err, "can't load OpenAPI specification")
	}

	//  /api/connector_mgmt
	apiRouter := mainRouter.PathPrefix("/api/connector_mgmt").Subrouter()

	//  /api/connector_mgmt/v1
	apiV1Router := apiRouter.PathPrefix("/v1").Subrouter()
	//  /api/connector_mgmt/v2alpha1
	apiV2Alpha1Router := apiRouter.PathPrefix("/v2alpha1").Subrouter()

	//  /api/connector_mgmt/v1/openapi
	apiV1Router.HandleFunc("/openapi", coreHandlers.NewOpenAPIHandler(openAPIDefinitions).Get).Methods(http.MethodGet)

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
	apiV1ConnectorTypesRouter := apiV1Router.PathPrefix("/kafka_connector_types").Subrouter()
	apiV1ConnectorTypesRouter.HandleFunc("/labels", s.ConnectorTypesHandler.ListLabels).Methods(http.MethodGet)
	apiV1ConnectorTypesRouter.HandleFunc("/{connector_type_id}", s.ConnectorTypesHandler.Get).Methods(http.MethodGet)
	apiV1ConnectorTypesRouter.HandleFunc("", s.ConnectorTypesHandler.List).Methods(http.MethodGet)
	apiV1ConnectorTypesRouter.Use(authorizeMiddleware)
	apiV1ConnectorTypesRouter.Use(requireOrgID)

	//  /api/connector_mgmt/v1/kafka_connectors
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "kafka_connectors",
		Kind: "ConnectorList",
	})

	apiV1ConnectorsRouter := apiV1Router.PathPrefix("/kafka_connectors").Subrouter()
	apiV1ConnectorsRouter.HandleFunc("", s.ConnectorsHandler.Create).Methods(http.MethodPost)
	apiV1ConnectorsRouter.HandleFunc("", s.ConnectorsHandler.List).Methods(http.MethodGet)
	apiV1ConnectorsRouter.HandleFunc("/{connector_id}", s.ConnectorsHandler.Get).Methods(http.MethodGet)
	apiV1ConnectorsRouter.HandleFunc("/{connector_id}", s.ConnectorsHandler.Patch).Methods(http.MethodPatch)
	apiV1ConnectorsRouter.HandleFunc("/{connector_id}", s.ConnectorsHandler.Delete).Methods(http.MethodDelete)
	apiV1ConnectorsRouter.Use(authorizeMiddleware)
	apiV1ConnectorsRouter.Use(requireOrgID)

	//  /api/connector_mgmt/v1/kafka_connector_clusters
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "kafka_connector_clusters",
		Kind: "ConnectorClusterList",
	})

	apiV1ConnectorClustersRouter := apiV1Router.PathPrefix("/kafka_connector_clusters").Subrouter()
	apiV1ConnectorClustersRouter.HandleFunc("", s.ConnectorClusterHandler.Create).Methods(http.MethodPost)
	apiV1ConnectorClustersRouter.HandleFunc("", s.ConnectorClusterHandler.List).Methods(http.MethodGet)
	apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}", s.ConnectorClusterHandler.Get).Methods(http.MethodGet)
	apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}", s.ConnectorClusterHandler.Update).Methods(http.MethodPut)
	apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}", s.ConnectorClusterHandler.Delete).Methods(http.MethodDelete)
	apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}/addon_parameters", s.ConnectorClusterHandler.GetAddonParameters).Methods(http.MethodGet)
	apiV1ConnectorClustersRouter.HandleFunc("/{connector_cluster_id}/namespaces", s.ConnectorClusterHandler.GetNamespaces).Methods(http.MethodGet)
	apiV1ConnectorClustersRouter.Use(authorizeMiddleware)
	apiV1ConnectorClustersRouter.Use(requireOrgID)

	//  /api/connector_mgmt/v1/kafka_connector_namespaces
	v1Collections = append(v1Collections, api.CollectionMetadata{
		ID:   "kafka_connector_namespaces",
		Kind: "ConnectorNamespaceList",
	})

	apiV1ConnectorNamespacesRouter := apiV1Router.PathPrefix("/kafka_connector_namespaces").Subrouter()
	apiV1ConnectorNamespacesRouter.HandleFunc("", s.ConnectorNamespaceHandler.List).Methods(http.MethodGet)
	apiV1ConnectorNamespacesRouter.HandleFunc("/eval", s.ConnectorNamespaceHandler.CreateEvaluation).Methods(http.MethodPost)
	apiV1ConnectorNamespacesRouter.HandleFunc("/{connector_namespace_id}", s.ConnectorNamespaceHandler.Get).Methods(http.MethodGet)
	if s.ConnectorsConfig.ConnectorNamespaceLifecycleAPI {
		apiV1ConnectorNamespacesRouter.HandleFunc("", s.ConnectorNamespaceHandler.Create).Methods(http.MethodPost)
		apiV1ConnectorNamespacesRouter.HandleFunc("/{connector_namespace_id}", s.ConnectorNamespaceHandler.Update).Methods(http.MethodPatch)
		apiV1ConnectorNamespacesRouter.HandleFunc("/{connector_namespace_id}", s.ConnectorNamespaceHandler.Delete).Methods(http.MethodDelete)
	} else {
		apiV1ConnectorNamespacesRouter.HandleFunc("", api.SendMethodNotAllowed).Methods(http.MethodPost)
		apiV1ConnectorNamespacesRouter.HandleFunc("/{connector_namespace_id}", api.SendMethodNotAllowed).Methods(http.MethodPatch)
		apiV1ConnectorNamespacesRouter.HandleFunc("/{connector_namespace_id}", api.SendMethodNotAllowed).Methods(http.MethodDelete)
	}
	apiV1ConnectorNamespacesRouter.Use(authorizeMiddleware)
	apiV1ConnectorNamespacesRouter.Use(requireOrgID)

	// Expose v1 API
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
	apiV1Router.HandleFunc("", v1Metadata.ServeHTTP).Methods(http.MethodGet)

	// This section adds the API's accessed by the connector agent...
	{
		// /api/connector_mgmt/v1/agent/kafka_connector_clusters/{connector_cluster_id}
		agentV1Router := apiV1Router.PathPrefix("/agent/kafka_connector_clusters/{connector_cluster_id}").Subrouter()
		agentV1Router.HandleFunc("/status", s.ConnectorClusterHandler.UpdateConnectorClusterStatus).Methods(http.MethodPut)
		agentV1Router.HandleFunc("/deployments", s.ConnectorClusterHandler.ListDeployments).Methods(http.MethodGet)
		agentV1Router.HandleFunc("/deployments/{deployment_id}", s.ConnectorClusterHandler.GetDeployment).Methods(http.MethodGet)
		agentV1Router.HandleFunc("/namespaces", s.ConnectorClusterHandler.GetAgentNamespaces).Methods(http.MethodGet)
		agentV1Router.HandleFunc("/namespaces/{namespace_id}", s.ConnectorClusterHandler.GetAgentNamespace).Methods(http.MethodGet)
		agentV1Router.HandleFunc("/namespaces/{namespace_id}/status", s.ConnectorClusterHandler.UpdateNamespaceStatus).Methods(http.MethodPut)
		agentV1Router.HandleFunc("/deployments/{deployment_id}/status", s.ConnectorClusterHandler.UpdateDeploymentStatus).Methods(http.MethodPut)
		auth.UseOperatorAuthorisationMiddleware(agentV1Router, s.KeycloakService.GetRealmConfig().ValidIssuerURI, "connector_cluster_id", s.AuthAgentService)
	}

	// This section adds the v2alpha1 API's accessible by both Users and the Processor agent...
	{
		// /api/connector_mgmt/v2alpha1/processors
		if s.ProcessorsConfig.ProcessorsEnabled {
			v2alpha1Collections := []api.CollectionMetadata{}

			//  /api/connector_mgmt/v2alpha1/processors
			v2alpha1Collections = append(v2alpha1Collections, api.CollectionMetadata{
				ID:   "processors",
				Kind: "ProcessorList",
			})

			apiV2Alpha1ProcessorsRouter := apiV2Alpha1Router.PathPrefix("/processors").Subrouter()
			apiV2Alpha1ProcessorsRouter.HandleFunc("", s.ProcessorsHandler.Create).Methods(http.MethodPost)
			apiV2Alpha1ProcessorsRouter.HandleFunc("", s.ProcessorsHandler.List).Methods(http.MethodGet)
			apiV2Alpha1ProcessorsRouter.HandleFunc("/{processor_id}", s.ProcessorsHandler.Get).Methods(http.MethodGet)
			apiV2Alpha1ProcessorsRouter.HandleFunc("/{processor_id}", s.ProcessorsHandler.Patch).Methods(http.MethodPatch)
			apiV2Alpha1ProcessorsRouter.HandleFunc("/{processor_id}", s.ProcessorsHandler.Delete).Methods(http.MethodDelete)
			apiV2Alpha1ProcessorsRouter.Use(authorizeMiddleware)
			apiV2Alpha1ProcessorsRouter.Use(requireOrgID)

			// /api/connector_mgmt/v2alpha1/agent/kafka_connector_clusters/{connector_cluster_id}
			agentV2Alpha1Router := apiV2Alpha1Router.PathPrefix("/agent/kafka_connector_clusters/{connector_cluster_id}").Subrouter()
			agentV2Alpha1Router.HandleFunc("/processors", s.ConnectorClusterHandler.ListProcessorDeployments).Methods(http.MethodGet)
			agentV2Alpha1Router.HandleFunc("/processors/{processor_deployment_id}", s.ConnectorClusterHandler.GetProcessorDeployment).Methods(http.MethodGet)
			agentV2Alpha1Router.HandleFunc("/processors/{processor_deployment_id}/status", s.ConnectorClusterHandler.UpdateProcessorDeploymentStatus).Methods(http.MethodPut)

			// Expose v2alpha1 API
			v2alpha1Metadata := api.VersionMetadata{
				ID:          "v2alpha1",
				Collections: v2alpha1Collections,
			}
			apiMetadata.Versions = append(apiMetadata.Versions, v2alpha1Metadata)
			apiV2Alpha1Router.HandleFunc("", v2alpha1Metadata.ServeHTTP).Methods(http.MethodGet)
		}
	}

	// Expose API versions
	apiRouter.HandleFunc("", apiMetadata.ServeHTTP).Methods(http.MethodGet)

	// This section adds APIs accessed by connector admins
	adminRouter := apiV1Router.PathPrefix("/admin").Subrouter()
	adminRouter.Use(auth.NewRequireIssuerMiddleware().RequireIssuer([]string{s.KeycloakService.GetConfig().AdminAPISSORealm.ValidIssuerURI}, kerrors.ErrorNotFound))
	adminRouter.Use(auth.NewRolesAuthzMiddleware(s.AdminRoleAuthZConfig).RequireRolesForMethods(kerrors.ErrorNotFound))
	adminRouter.Use(auth.NewAuditLogMiddleware().AuditLog(kerrors.ErrorNotFound))
	adminRouter.HandleFunc("/kafka_connector_clusters", s.ConnectorAdminHandler.ListConnectorClusters).Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafka_connector_clusters/{connector_cluster_id}", s.ConnectorAdminHandler.GetConnectorCluster).Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafka_connector_clusters/{connector_cluster_id}/namespaces", s.ConnectorAdminHandler.GetClusterNamespaces).Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafka_connector_clusters/{connector_cluster_id}/connectors", s.ConnectorAdminHandler.GetClusterConnectors).Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafka_connector_clusters/{connector_cluster_id}/deployments", s.ConnectorAdminHandler.GetClusterDeployments).Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafka_connector_clusters/{connector_cluster_id}/deployments/{deployment_id}", s.ConnectorAdminHandler.GetConnectorDeployment).Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafka_connector_clusters/{connector_cluster_id}/deployments/{deployment_id}", s.ConnectorAdminHandler.PatchConnectorDeployment).Methods(http.MethodPatch)
	adminRouter.HandleFunc("/kafka_connector_namespaces", s.ConnectorAdminHandler.GetConnectorNamespaces).Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafka_connector_namespaces", s.ConnectorAdminHandler.CreateConnectorNamespace).Methods(http.MethodPost)
	adminRouter.HandleFunc("/kafka_connector_namespaces/{namespace_id}", s.ConnectorAdminHandler.GetConnectorNamespace).Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafka_connector_namespaces/{namespace_id}", s.ConnectorAdminHandler.DeleteConnectorNamespace).Methods(http.MethodDelete)
	adminRouter.HandleFunc("/kafka_connector_namespaces/{namespace_id}/connectors", s.ConnectorAdminHandler.GetNamespaceConnectors).Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafka_connector_namespaces/{namespace_id}/deployments", s.ConnectorAdminHandler.GetNamespaceDeployments).Methods(http.MethodGet)
	//TODO: add, to consistency with the {connector_cluster_id}/ counterparts
	//adminRouter.HandleFunc("/kafka_connector_namespaces/{namespace_id}/deployments/{deployment_id}", s.ConnectorAdminHandler.GetNamespaceDeployment).Methods(http.MethodGet)
	//adminRouter.HandleFunc("/kafka_connector_namespaces/{namespace_id}/deployments/{deployment_id}", s.ConnectorAdminHandler.PatchCNamespaceDeployment).Methods(http.MethodPatch)
	adminRouter.HandleFunc("/kafka_connectors/{connector_id}", s.ConnectorAdminHandler.GetConnector).Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafka_connectors/{connector_id}", s.ConnectorAdminHandler.DeleteConnector).Methods(http.MethodDelete)
	adminRouter.HandleFunc("/kafka_connectors/{connector_id}", s.ConnectorAdminHandler.PatchConnector).Methods(http.MethodPatch)
	adminRouter.HandleFunc("/kafka_connector_types", s.ConnectorAdminHandler.ListConnectorTypes).Methods(http.MethodGet)
	adminRouter.HandleFunc("/kafka_connector_types/{connector_type_id}", s.ConnectorAdminHandler.GetConnectorType).Methods(http.MethodGet)

	apiRouter.Use(coreHandlers.MetricsMiddleware)
	apiRouter.Use(db.TransactionMiddleware(s.DB))
	apiRouter.Use(gorillaHandlers.CompressHandler)
	return nil
}
