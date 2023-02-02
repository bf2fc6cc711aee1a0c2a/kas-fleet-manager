package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
	"net/http"
	"strconv"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
)

func (h *ConnectorClusterHandler) ListProcessorDeployments(w http.ResponseWriter, r *http.Request) {
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

			getList := func() (list private.ProcessorDeploymentList, err *errors.ServiceError) {

				resources, paging, err := h.Service.ListProcessorDeployments(ctx, connectorClusterId, false, false, false, listArgs, gtVersion)
				if err != nil {
					return
				}

				list = private.ProcessorDeploymentList{
					Kind:  "ProcessorDeploymentList",
					Page:  int32(paging.Page),
					Size:  int32(paging.Size),
					Total: int32(paging.Total),
				}

				for _, resource := range resources {
					converted, serviceError := h.resolveProcessorDeploymentRefsAndPresentDeployment(&resource)
					if serviceError != nil {
						sentry.CaptureException(serviceError)
						glog.Errorf("Failed to present processor deployment %s: %v", resource.ID, serviceError)
						// also reduce size and total count
						list.Size--
						list.Total--
					} else {
						list.Items = append(list.Items, *converted)
					}
				}
				return
			}

			return getList()
		},
	}
	handlers.HandleList(w, r, cfg)
}

func (h *ConnectorClusterHandler) GetProcessorDeployment(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	processorDeploymentId := mux.Vars(r)["processor_deployment_id"]

	ctx := r.Context()
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), validateConnectorClusterId(ctx, h.Service)),
			handlers.Validation("processor_deployment_id", &processorDeploymentId, handlers.MinLen(1), handlers.MaxLen(maxProcessorDeploymentIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			var resource *dbapi.ProcessorDeployment
			resource, err := h.Service.GetProcessorDeployment(r.Context(), processorDeploymentId)
			if err != nil {
				return nil, err
			}
			if resource.ClusterID != connectorClusterId {
				return nil, errors.NotFound("Processor deployment not found")
			}

			return h.resolveProcessorDeploymentRefsAndPresentDeployment(resource)
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h *ConnectorClusterHandler) resolveProcessorDeploymentRefsAndPresentDeployment(processorDeployment *dbapi.ProcessorDeployment) (*private.ProcessorDeployment, *errors.ServiceError) {
	// avoid ignoring this deployment altogether if there is an issue in getting secrets from the vault
	invalidSecrets, err := h.Processors.ResolveProcessorRefsWithBase64Secrets(&processorDeployment.Processor)
	if err != nil {
		if invalidSecrets {
			// log error in getting secrets and signal that processor spec doesn't have secrets
			glog.Errorf("Error getting processor %s with base64 secrets: %s", processorDeployment.Processor.ID, err)
			processorDeployment.Processor.Definition = []byte("{}")
			processorDeployment.Processor.ErrorHandler = []byte("{}")
		} else {
			return &private.ProcessorDeployment{}, err
		}
	}

	converted, err := presenters.PresentProcessorDeployment(processorDeployment, !invalidSecrets)
	if err != nil {
		return &private.ProcessorDeployment{}, err
	}

	return converted, nil
}

func (h *ConnectorClusterHandler) UpdateProcessorDeploymentStatus(w http.ResponseWriter, r *http.Request) {
	connectorClusterId := mux.Vars(r)["connector_cluster_id"]
	deploymentId := mux.Vars(r)["processor_deployment_id"]
	var resource private.ProcessorDeploymentStatus

	ctx := r.Context()
	cfg := &handlers.HandlerConfig{
		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.Validation("connector_cluster_id", &connectorClusterId, handlers.MinLen(1), handlers.MaxLen(maxConnectorClusterIdLength), validateConnectorClusterId(ctx, h.Service)),
			handlers.Validation("processor_deployment_id", &deploymentId, handlers.MinLen(1), handlers.MaxLen(maxProcessorDeploymentIdLength)),
			handlers.Validation("phase", (*string)(&resource.Phase), handlers.IsOneOf(dbapi.AgentProcessorStatusPhase...)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			converted, serr := presenters.ConvertProcessorDeploymentStatus(resource)
			if serr != nil {
				return nil, serr
			}
			converted.ID = deploymentId
			err := h.Service.UpdateProcessorDeploymentStatus(ctx, converted)
			return nil, err
		},
	}
	handlers.Handle(w, r, cfg, http.StatusNoContent)
}
