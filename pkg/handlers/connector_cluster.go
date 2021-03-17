package handlers

import (
	"context"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/signalbus"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

var (
	maxConnectorClusterIdLength = 32
)

type connectorClusterHandler struct {
	bus            signalbus.SignalBus
	service        services.ConnectorClusterService
	config         services.ConfigService
	keycloak       services.KeycloakService
	connectorTypes services.ConnectorTypesService
	vault          services.VaultService
}

func NewConnectorClusterHandler(bus signalbus.SignalBus, service services.ConnectorClusterService, config services.ConfigService, keycloak services.KeycloakService, connectorTypes services.ConnectorTypesService, vault services.VaultService) *connectorClusterHandler {
	return &connectorClusterHandler{
		bus:            bus,
		service:        service,
		config:         config,
		keycloak:       keycloak,
		connectorTypes: connectorTypes,
		vault:          vault,
	}
}

func (h *connectorClusterHandler) Create(w http.ResponseWriter, r *http.Request) {
	var resource openapi.ConnectorCluster
	cfg := &handlerConfig{
		MarshalInto: &resource,
		Validate: []validate{
			validation("name", &resource.Metadata.Name,
				withDefault("New Cluster"), minLen(1), maxLen(100)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			convResource := presenters.ConvertConnectorCluster(resource)

			claims, err := auth.GetClaimsFromContext(r.Context())
			if err != nil {
				return nil, errors.Unauthenticated("user not authenticated")
			}
			convResource.Owner = auth.GetUsernameFromClaims(claims)
			convResource.OrganisationId = auth.GetOrgIdFromClaims(claims)
			convResource.Status.Phase = api.ConnectorClusterPhaseUnconnected

			if err := h.service.Create(r.Context(), &convResource); err != nil {
				return nil, err
			}
			return presenters.PresentConnectorCluster(convResource), nil
		},
		ErrorHandler: handleError,
	}

	// return 202 status accepted
	handle(w, r, cfg, http.StatusAccepted)
}

func (h *connectorClusterHandler) Get(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	cfg := &handlerConfig{
		Validate: []validate{
			validation("connector_cluster_id", &connectorClusterId, minLen(1), maxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			resource, err := h.service.Get(r.Context(), connectorClusterId)
			if err != nil {
				return nil, err
			}
			return presenters.PresentConnectorCluster(resource), nil
		},
		ErrorHandler: handleError,
	}
	handleGet(w, r, cfg)
}

func (h *connectorClusterHandler) Delete(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	cfg := &handlerConfig{
		Validate: []validate{
			validation("connector_cluster_id", &connectorClusterId, minLen(1), maxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			err := h.service.Delete(r.Context(), connectorClusterId)
			return nil, err
		},
		ErrorHandler: handleError,
	}
	handleDelete(w, r, cfg, http.StatusNoContent)
}

func (h *connectorClusterHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			listArgs := services.NewListArguments(r.URL.Query())
			resources, paging, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}

			resourceList := openapi.ConnectorClusterList{
				Kind:  "ConnectorClusterList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			for _, resource := range resources {
				converted := presenters.PresentConnectorCluster(resource)
				resourceList.Items = append(resourceList.Items, converted)
			}

			return resourceList, nil
		},
		ErrorHandler: handleError,
	}

	handleList(w, r, cfg)
}

func (h *connectorClusterHandler) GetAddonParameters(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	cfg := &handlerConfig{
		Validate: []validate{
			validation("connector_cluster_id", &connectorClusterId, minLen(1), maxLen(maxConnectorClusterIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {

			// To make sure the user can access the cluster....
			_, err := h.service.Get(r.Context(), connectorClusterId)
			if err != nil {
				return nil, err
			}

			acc, err := h.keycloak.RegisterConnectorFleetshardOperatorServiceAccount(connectorClusterId, connectorFleetshardOperatorRoleName)
			if err != nil {
				return false, errors.GeneralError("failed to create service account for connector cluster %s due to error: %v", connectorClusterId, err)
			}
			params := h.buildAddonParams(acc, connectorClusterId)
			result := make([]openapi.AddonParameter, len(params))
			for i, p := range params {
				result[i] = presenters.PresentAddonParameter(p)
			}
			u, eerr := h.buildTokenURL(acc)
			if eerr == nil {
				w.Header().Set("Token-Help", fmt.Sprintf(`curl --data "grant_type=client_credentials" "%s" | jq -r .access_token`, u))
			}
			return result, nil
		},
		ErrorHandler: handleError,
	}
	handleGet(w, r, cfg)
}

const (
	connectorFleetshardOperatorRoleName                  = "connector_fleetshard_operator"
	connectorFleetshardOperatorParamMasSSOBaseUrl        = "mas-sso-base-url"
	connectorFleetshardOperatorParamMasSSORealm          = "mas-sso-realm"
	connectorFleetshardOperatorParamServiceAccountId     = "client-id"
	connectorFleetshardOperatorParamServiceAccountSecret = "client-secret"
	connectorFleetshardOperatorParamClusterId            = "cluster-id"
	connectorFleetshardOperatorParamControlPlaneBaseURL  = "control-plane-base-url"
)

func (o *connectorClusterHandler) buildAddonParams(serviceAccount *api.ServiceAccount, clusterId string) []ocm.AddonParameter {
	p := []ocm.AddonParameter{
		{
			Id:    connectorFleetshardOperatorParamMasSSOBaseUrl,
			Value: o.config.GetConfig().Keycloak.BaseURL,
		},
		{
			Id:    connectorFleetshardOperatorParamMasSSORealm,
			Value: o.config.GetConfig().Keycloak.KafkaRealm.Realm,
		},
		{
			Id:    connectorFleetshardOperatorParamServiceAccountId,
			Value: serviceAccount.ClientID,
		},
		{
			Id:    connectorFleetshardOperatorParamServiceAccountSecret,
			Value: serviceAccount.ClientSecret,
		},
		{
			Id:    connectorFleetshardOperatorParamControlPlaneBaseURL,
			Value: o.config.GetConfig().Server.PublicHostURL,
		},
		{
			Id:    connectorFleetshardOperatorParamClusterId,
			Value: clusterId,
		},
	}
	return p
}

func (o *connectorClusterHandler) buildTokenURL(serviceAccount *api.ServiceAccount) (string, error) {
	u, err := url.Parse(o.config.GetConfig().Keycloak.KafkaRealm.TokenEndpointURI)
	if err != nil {
		return "", err
	}
	u.User = url.UserPassword(serviceAccount.ClientID, serviceAccount.ClientSecret)
	return u.String(), nil
}

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
		ErrorHandler: handleError,
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
					converted, err := presenters.PresentConnectorDeployment(resource)
					if err != nil {
						return list, err
					}
					list.Items = append(list.Items, converted)
				}
				return
			}

			if v := query.Get("watch"); v == "true" {
				idx := 0
				list, err := getList()
				firstPoll := true

				sub := h.bus.Subscribe(fmt.Sprintf("/kafka-connector-clusters/%s/connectors", connectorClusterId))
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

								if firstPoll {
									firstPoll = false
									// bookmark idea taken from: https://kubernetes.io/docs/reference/using-api/api-concepts/#watch-bookmarks
									return openapi.ConnectorDeploymentWatchEvent{
										Type: "BOOKMARK",
									}, nil
								}

								// get the next list..
								list, err = getList()
								if err != nil {
									return nil, err
								}
								if len(list.Items) == 0 {
									// no items?  lets wait for some to come in...
									// in the future we could also wake up based on events to provide lower notification latencies (like when updates occur)

									// lets release the DB connection so that we don't tie those up while we wait to poll again..
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
		ErrorHandler: handleError,
	}
	handleList(w, r, cfg)
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

func (h *connectorClusterHandler) UpdateDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	connectorId := mux.Vars(r)["connector_id"]
	var resource api.ConnectorDeploymentStatus

	cfg := &handlerConfig{
		MarshalInto: &resource,
		Validate: []validate{
			validation("connector_cluster_id", &connectorClusterId, minLen(1), maxLen(maxConnectorClusterIdLength)),
			validation("connector_id", &connectorClusterId, minLen(1), maxLen(maxConnectorIdLength)),
			validation("phase", &resource.Phase, isOneOf(api.AllConnectorClusterStatus...)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			err := h.service.UpdateConnectorDeploymentStatus(ctx, connectorClusterId, connectorId, resource)
			return nil, err
		},
		ErrorHandler: handleError,
	}
	handle(w, r, cfg, http.StatusNoContent)
}
