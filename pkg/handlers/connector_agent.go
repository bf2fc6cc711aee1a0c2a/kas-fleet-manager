package handlers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

func (h *connectorClusterHandler) UpdateConnectorClusterStatus(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	var resource openapi.ConnectorClusterStatus

	cfg := &handlerConfig{
		MarshalInto: &resource,
		Validate: []validate{
			validation("connector_cluster_id", &connectorClusterId, minLen(1), maxLen(maxConnectorClusterIdLength)),
			validation("phase", &resource.Phase, isOneOf(api.AllConnectorClusterStatus...)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			convResource := presenters.ConvertConnectorClusterStatus(resource)
			err := h.service.UpdateConnectorClusterStatus(ctx, connectorClusterId, convResource)
			return nil, err
		},
	}
	handle(w, r, cfg, http.StatusNoContent)
}

func (h *connectorClusterHandler) ListDeployments(w http.ResponseWriter, r *http.Request) {
	// h.service.ListConnectors()
	ctx := r.Context()
	query := r.URL.Query()
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]

	cfg := &handlerConfig{
		Validate: []validate{
			validation("connector_cluster_id", &connectorClusterId, minLen(1), maxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			gtVersion := int64(0)
			if v := query.Get("gt_version"); v != "" {
				gtVersion, _ = strconv.ParseInt(v, 10, 0)
			}

			listArgs := services.NewListArguments(query)

			getList := func() (list openapi.ConnectorDeploymentList, err *errors.ServiceError) {

				resources, paging, err := h.service.ListConnectorDeployments(ctx, connectorClusterId, listArgs, gtVersion)
				if err != nil {
					return
				}

				list = openapi.ConnectorDeploymentList{
					Kind:  "ConnectorDeploymentList",
					Page:  int32(paging.Page),
					Size:  int32(paging.Size),
					Total: int32(paging.Total),
				}

				for _, resource := range resources {
					converted, serviceError := h.presentDeployment(r, resource)
					if serviceError != nil {
						sentry.CaptureException(err)
						glog.Errorf("failed to present connector deployment %s: %v", resource.ID, err)
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

				sub := h.bus.Subscribe(fmt.Sprintf("/kafka-connector-clusters/%s/deployments", connectorClusterId))
				return eventStream{
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
								return openapi.ConnectorDeploymentWatchEvent{
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
										return openapi.ConnectorDeploymentWatchEvent{
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
	handleList(w, r, cfg)
}

func (h *connectorClusterHandler) presentDeployment(r *http.Request, resource api.ConnectorDeployment) (openapi.ConnectorDeployment, *errors.ServiceError) {
	converted, err := presenters.PresentConnectorDeployment(resource)
	if err != nil {
		return openapi.ConnectorDeployment{}, err
	}

	apiSpec, err := h.service.GetConnectorWithBase64Secrets(r.Context(), resource)
	if err != nil {
		return openapi.ConnectorDeployment{}, err
	}

	pc, err := presenters.PresentConnector(&apiSpec)
	if err != nil {
		return openapi.ConnectorDeployment{}, err
	}

	catalogEntry, err := h.connectorTypes.GetConnectorCatalogEntry(pc.ConnectorTypeId, pc.Channel)
	if err != nil {
		return openapi.ConnectorDeployment{}, err
	}

	converted.Spec.ShardMetadata = catalogEntry.ShardMetadata
	converted.Spec.ConnectorSpec = pc.ConnectorSpec
	converted.Spec.DesiredState = pc.DesiredState
	converted.Spec.ConnectorId = pc.Id
	converted.Spec.KafkaId = pc.Metadata.KafkaId
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

func (h *connectorClusterHandler) GetDeployment(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	deploymentId := mux.Vars(r)["deployment_id"]

	cfg := &handlerConfig{
		Validate: []validate{
			validation("connector_cluster_id", &connectorClusterId, minLen(1), maxLen(maxConnectorClusterIdLength)),
			validation("deployment_id", &deploymentId, minLen(1), maxLen(maxConnectorIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			resource, err := h.service.GetDeployment(r.Context(), deploymentId)
			if err != nil {
				return nil, err
			}
			if resource.ClusterID != connectorClusterId {
				return nil, errors.NotFound("Connector deployment not found")
			}

			return h.presentDeployment(r, resource)
		},
	}
	handleGet(w, r, cfg)
}

func (h *connectorClusterHandler) UpdateDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	deploymentId := mux.Vars(r)["deployment_id"]
	var resource openapi.ConnectorDeploymentStatus

	cfg := &handlerConfig{
		MarshalInto: &resource,
		Validate: []validate{
			validation("connector_cluster_id", &connectorClusterId, minLen(1), maxLen(maxConnectorClusterIdLength)),
			validation("deployment_id", &deploymentId, minLen(1), maxLen(maxConnectorIdLength)),
			validation("phase", &resource.Phase, isOneOf(api.AgentSetConnectorStatusPhase...)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			converted, serr := presenters.ConvertConnectorDeploymentStatus(resource)
			if serr != nil {
				return nil, serr
			}
			converted.ID = deploymentId
			err := h.service.UpdateConnectorDeploymentStatus(ctx, converted)
			return nil, err
		},
	}
	handle(w, r, cfg, http.StatusNoContent)
}
