package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

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
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	var resource private.ConnectorClusterStatus

	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
			handlers.Validation("phase", &resource.Phase, handlers.IsOneOf(dbapi.AllConnectorClusterStatus...)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			convResource := presenters.ConvertConnectorClusterStatus(resource)
			err := h.Service.UpdateConnectorClusterStatus(ctx, connectorClusterId, convResource)
			return nil, err
		},
	}
	handlers.Handle(w, r, cfg, http.StatusNoContent)
}

func (h *ConnectorClusterHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	// h.service.ListConnectors()
	ctx := r.Context()
	query := r.URL.Query()
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			gtVersion := int64(0)
			if v := query.Get("gt_version"); v != "" {
				gtVersion, _ = strconv.ParseInt(v, 10, 0)
			}

			listArgs := services.NewListArguments(query)

			getList := func() (list private.ConnectorDeploymentList, err *errors.ServiceError) {

				resources, paging, err := h.Service.ListConnectorDeployments(ctx, connectorClusterId, listArgs, gtVersion)
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
					converted, serviceError := h.presentDeployment(r, resource)
					if serviceError != nil {
						sentry.CaptureException(serviceError)
						glog.Errorf("failed to present connector deployment %s: %v", resource.ID, serviceError)
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

				sub := h.Bus.Subscribe(fmt.Sprintf("/kafka-connector-clusters/%s/deployments", connectorClusterId))
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
										return io.EOF, nil
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

func (h *ConnectorClusterHandler) presentDeployment(r *http.Request, resource dbapi.ConnectorDeployment) (private.ConnectorDeployment, *errors.ServiceError) {
	converted, err := presenters.PresentConnectorDeployment(resource)
	if err != nil {
		return private.ConnectorDeployment{}, err
	}

	apiSpec, err := h.Service.GetConnectorWithBase64Secrets(r.Context(), resource)
	if err != nil {
		return private.ConnectorDeployment{}, err
	}

	pc, err := presenters.PresentConnector(&apiSpec)
	if err != nil {
		return private.ConnectorDeployment{}, err
	}

	shardMetadataJson, err := h.ConnectorTypes.GetConnectorShardMetadata(resource.ConnectorTypeChannelId)
	if err != nil {
		return private.ConnectorDeployment{}, err
	}

	shardMetadata, err2 := shardMetadataJson.ShardMetadata.Object()
	if err2 != nil {
		return private.ConnectorDeployment{}, errors.GeneralError("failed to convert shard metadata")
	}

	converted.Spec.ShardMetadata = shardMetadata
	converted.Spec.ConnectorSpec = pc.Connector
	converted.Spec.DesiredState = string(pc.DesiredState)
	converted.Spec.ConnectorId = pc.Id
	converted.Spec.Kafka = private.KafkaConnectionSettings{
		Id:  pc.Kafka.Id,
		Url: pc.Kafka.Url,
	}
	converted.Spec.SchemaRegistry = private.SchemaRegistryConnectionSettings{
		Id:  pc.SchemaRegistry.Id,
		Url: pc.SchemaRegistry.Url,
	}
	converted.Spec.ServiceAccount = private.ServiceAccount{
		ClientId:     pc.ServiceAccount.ClientId,
		ClientSecret: pc.ServiceAccount.ClientSecret,
	}
	converted.Spec.ConnectorTypeId = pc.ConnectorTypeId
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

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
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

			return h.presentDeployment(r, resource)
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h *ConnectorClusterHandler) UpdateDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	deploymentId := mux.Vars(r)["deployment_id"]
	var resource private.ConnectorDeploymentStatus

	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength)),
			handlers.Validation("deployment_id", &deploymentId, handlers.MinLen(1), handlers.MaxLen(maxConnectorIdLength)),
			handlers.Validation("phase", &resource.Phase, handlers.IsOneOf(dbapi.AgentSetConnectorStatusPhase...)),
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
