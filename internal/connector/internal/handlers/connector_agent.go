package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

func (h *ConnectorClusterHandler) UpdateConnectorClusterStatus(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	var resource private.ConnectorClusterStatus

	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), validateConnectorClusterId(ctx, h.Service)),
			handlers.Validation("phase", (*string)(&resource.Phase), handlers.IsOneOf(dbapi.AgentRequestConnectorClusterStatus...)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			convResource := presenters.ConvertConnectorClusterStatus(resource)
			err := h.Service.UpdateConnectorClusterStatus(ctx, connectorClusterId, convResource)
			if err == nil {

				// get empty deleting cluster namespaces
				var namespaceList dbapi.ConnectorNamespaceList
				namespaceList, err = h.ConnectorNamespace.GetEmptyDeletingNamespaces(connectorClusterId)
				if err == nil {

					// convert namespace errors to a single error
					var errorList []*errors.ServiceError

					deletingNamespaces := make(map[string]*dbapi.ConnectorNamespace)
					for _, ns := range namespaceList {
						deletingNamespaces[ns.ID] = ns
					}

					for _, namespaceStatus := range resource.Namespaces {
						// physical namespace hasn't been deleted yet
						delete(deletingNamespaces, namespaceStatus.Id)

						err := h.ConnectorNamespace.UpdateConnectorNamespaceStatus(
							ctx,
							namespaceStatus.Id,
							presenters.ConvertConnectorNamespaceDeploymentStatus(namespaceStatus))

						if err != nil {
							errorList = append(errorList, err)
						}
					}

					// process missing namespaces as deleted
					if len(deletingNamespaces) > 0 {
						for _, namespace := range deletingNamespaces {
							// set namespace phase to deleted
							namespace.Status.Phase = dbapi.ConnectorNamespacePhaseDeleted
							if err := h.ConnectorNamespace.Update(ctx, namespace); err != nil {
								errorList = append(errorList, err)
							}
						}

						h.Bus.Notify("reconcile:connector_namespace")
					}

					// consolidate namespace update errors
					if len(errorList) > 0 {
						var msg []string
						for _, e := range errorList {
							msg = append(msg, e.Error())
						}
						reason := strings.Join(msg, ";")
						reason = strings.Replace(reason, errors.ERROR_CODE_PREFIX, errors.CONNECTOR_MGMT_ERROR_CODE_PREFIX, -1)
						err = errors.GeneralError(reason)
					}
				}
			}

			return nil, err
		},
	}
	handlers.Handle(w, r, cfg, http.StatusNoContent)
}

func (h *ConnectorClusterHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	query := r.URL.Query()
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), validateConnectorClusterId(ctx, h.Service)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			gtVersion := int64(0)
			if v := query.Get("gt_version"); v != "" {
				gtVersion, _ = strconv.ParseInt(v, 10, 0)
			}

			listArgs := services.NewListArguments(query)

			getList := func() (list private.ConnectorDeploymentList, err *errors.ServiceError) {

				resources, paging, err := h.Service.ListConnectorDeployments(ctx, connectorClusterId, false, false, false, listArgs, gtVersion)
				if err != nil {
					return
				}

				list = private.ConnectorDeploymentList{
					Kind:  "ConnectorDeploymentList",
					Page:  int32(paging.Page),
					Size:  int32(paging.Size),
					Total: int32(paging.Total),
				}

				for _, resource := range resources {
					converted, serviceError := h.resolveConnectorRefsAndPresentDeployment(resource)
					if serviceError != nil {
						sentry.CaptureException(serviceError)
						glog.Errorf("Failed to present connector deployment %s: %v", resource.ID, serviceError)
						// also reduce size and total count
						list.Size--
						list.Total--
					} else {
						list.Items = append(list.Items, converted)
					}
				}
				return
			}

			if v := query.Get("watch"); v == "true" {
				idx := 0
				list, err := getList()
				bookmarkSent := false

				sub := h.Bus.Subscribe(fmt.Sprintf("/kafka_connector_clusters/%s/deployments", connectorClusterId))
				return handlers.EventStream{
					ContentType: "application/json;stream=watch",
					Close:       sub.Close,
					GetNextEvent: func() (interface{}, *errors.ServiceError) {
						for { // This function blocks until there is an event to return...
							if err != nil {
								return nil, err
							}
							if idx < len(list.Items) {
								result := list.Items[idx]
								gtVersion = result.Metadata.ResourceVersion
								idx += 1
								return private.ConnectorDeploymentWatchEvent{
									Type:   "CHANGE",
									Object: result,
								}, nil
							} else {

								// get the next list..
								list, err = getList()
								if err != nil {
									return nil, err
								}
								idx = 0

								// did we run out of items to send?
								if len(list.Items) == 0 {

									// bookmark idea taken from: https://kubernetes.io/docs/reference/using-api/api-concepts/#watch-bookmarks
									if !bookmarkSent {
										bookmarkSent = true
										return private.ConnectorDeploymentWatchEvent{
											Type: "BOOKMARK",
										}, nil
									}

									// Lets wait for some items to come into the list

									// release the DB connection so that we don't tie those up while we wait to poll again..
									err := db.Resolve(ctx)
									if err != nil {
										return nil, errors.GeneralError("internal error")
									}

									if waitForCancelOrTimeoutOrNotification(ctx, 30*time.Second, sub) {
										// ctx was canceled... likely due to the http connection being closed by
										// the client.  Signal the event stream is done.
										return nil, nil
									}

									// get a new DB connection...
									err = db.Begin(ctx)
									if err != nil {
										return nil, errors.GeneralError("internal error")
									}
								}
							}
						}
					},
				}, nil
			} else {
				return getList()
			}
		},
	}
	handlers.HandleList(w, r, cfg)
}

func (h *ConnectorClusterHandler) resolveConnectorRefsAndPresentDeployment(connectorDeployment dbapi.ConnectorDeployment) (private.ConnectorDeployment, *errors.ServiceError) {
	// avoid ignoring this deployment altogether if there is an issue in getting secrets from the vault
	invalidSecrets, err := h.Connectors.ResolveConnectorRefsWithBase64Secrets(&connectorDeployment.Connector)
	if err != nil {
		if invalidSecrets {
			// log error in getting secrets and signal that connector spec doesn't have secrets
			glog.Errorf("Error getting connector %s with base64 secrets: %s", connectorDeployment.Connector.ID, err)
			connectorDeployment.Connector.ConnectorSpec = []byte("{}")
		} else {
			return private.ConnectorDeployment{}, err
		}
	}

	converted, err := presenters.PresentConnectorDeployment(connectorDeployment, !invalidSecrets)
	if err != nil {
		return private.ConnectorDeployment{}, err
	}

	return converted, nil
}

// waitForCancelOrTimeoutOrNotification returns true if the context has been canceled or false after the timeout or sub signal
func waitForCancelOrTimeoutOrNotification(ctx context.Context, timeout time.Duration, sub *signalbus.Subscription) bool {
	tc, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	select {
	case <-tc.Done():
		return false
	case <-sub.Signal():
		return false
	case <-ctx.Done():
		return true
	}
}

func (h *ConnectorClusterHandler) GetDeployment(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	deploymentId := mux.Vars(r)["deployment_id"]

	ctx := r.Context()
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), validateConnectorClusterId(ctx, h.Service)),
			handlers.Validation("deployment_id", &deploymentId, handlers.MinLen(1), handlers.MaxLen(maxConnectorIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			var resource dbapi.ConnectorDeployment
			resource, err := h.Service.GetDeployment(r.Context(), deploymentId)
			if err != nil {
				return nil, err
			}
			if resource.ClusterID != connectorClusterId {
				return nil, errors.NotFound("Connector deployment not found")
			}

			return h.resolveConnectorRefsAndPresentDeployment(resource)
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h *ConnectorClusterHandler) GetAgentNamespaces(writer http.ResponseWriter, request *http.Request) {
	connectorClusterId := mux.Vars(request)["connector_cluster_id"]
	ctx := request.Context()
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), validateConnectorClusterId(ctx, h.Service)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := request.Context()

			query := request.URL.Query()
			listArgs := services.NewListArguments(query)

			gtVersion := int64(0)
			if v := query.Get("gt_version"); v != "" {
				gtVersion, _ = strconv.ParseInt(v, 10, 0)
			}

			resources, paging, err := h.ConnectorNamespace.List(ctx, []string{connectorClusterId}, listArgs, gtVersion)
			if err != nil {
				return nil, err
			}

			resourceList := private.ConnectorNamespaceDeploymentList{
				Kind:  "ConnectorNamespaceDeploymentList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			for _, resource := range resources {
				converted := presenters.PresentConnectorNamespaceDeployment(resource, h.QuotaConfig)
				resourceList.Items = append(resourceList.Items, converted)
			}

			return resourceList, nil
		},
	}

	handlers.HandleList(writer, request, cfg)
}

func (h *ConnectorClusterHandler) doGetNamespace(w http.ResponseWriter, r *http.Request, presenter func(*dbapi.ConnectorNamespace, *config.ConnectorsQuotaConfig) interface{}) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	namespaceId := mux.Vars(r)["namespace_id"]

	ctx := r.Context()
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), validateConnectorClusterId(ctx, h.Service)),
			handlers.Validation("namespace_id", &namespaceId, handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			var resource *dbapi.ConnectorNamespace
			resource, err := h.ConnectorNamespace.Get(r.Context(), namespaceId)
			if err != nil {
				return nil, err
			}
			if resource.ClusterId != connectorClusterId {
				return nil, errors.NotFound("Connector namespace %s not found", namespaceId)
			}

			return presenter(resource, h.QuotaConfig), nil
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h *ConnectorClusterHandler) GetNamespace(w http.ResponseWriter, r *http.Request) {
	h.doGetNamespace(w, r, func(ns *dbapi.ConnectorNamespace, quotaConfig *config.ConnectorsQuotaConfig) interface{} {
		return presenters.PresentConnectorNamespace(ns, quotaConfig)
	})
}

func (h *ConnectorClusterHandler) GetAgentNamespace(w http.ResponseWriter, r *http.Request) {
	h.doGetNamespace(w, r, func(ns *dbapi.ConnectorNamespace, quotaConfig *config.ConnectorsQuotaConfig) interface{} {
		return presenters.PresentConnectorNamespaceDeployment(ns, quotaConfig)
	})
}

func (h *ConnectorClusterHandler) UpdateDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	deploymentId := mux.Vars(r)["deployment_id"]
	var resource private.ConnectorDeploymentStatus

	ctx := r.Context()
	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), validateConnectorClusterId(ctx, h.Service)),
			handlers.Validation("deployment_id", &deploymentId, handlers.MinLen(1), handlers.MaxLen(maxConnectorIdLength)),
			handlers.Validation("phase", (*string)(&resource.Phase), handlers.IsOneOf(dbapi.AgentConnectorStatusPhase...)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			converted, serr := presenters.ConvertConnectorDeploymentStatus(resource)
			if serr != nil {
				return nil, serr
			}
			converted.ID = deploymentId
			err := h.Service.UpdateConnectorDeploymentStatus(ctx, converted)
			return nil, err
		},
	}
	handlers.Handle(w, r, cfg, http.StatusNoContent)
}

func (h *ConnectorClusterHandler) UpdateNamespaceStatus(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	namespaceId := mux.Vars(r)["namespace_id"]
	var resource private.ConnectorNamespaceDeploymentStatus

	ctx := r.Context()
	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), validateConnectorClusterId(ctx, h.Service)),
			handlers.Validation("namespace_id", &namespaceId, handlers.MinLen(1), handlers.MaxLen(maxConnectorNamespaceIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			converted := presenters.ConvertConnectorNamespaceDeploymentStatus(resource)
			err := h.ConnectorNamespace.UpdateConnectorNamespaceStatus(ctx, namespaceId, converted)
			return nil, err
		},
	}
	handlers.Handle(w, r, cfg, http.StatusNoContent)
}
