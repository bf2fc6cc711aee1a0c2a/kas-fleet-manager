package handlers

import (
	"context"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/authz"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/phase"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"net/http"
)

const (
	maxProcessorIdLength = 32
)

type ProcessorsHandler struct {
	processorsService services.ProcessorsService
	namespaceService  services.ConnectorNamespaceService
	vaultService      vault.VaultService
	authZService      authz.AuthZService
	processorsConfig  *config.ProcessorsConfig
}

func NewProcessorsHandler(processorsService services.ProcessorsService,
	namespaceService services.ConnectorNamespaceService, vaultService vault.VaultService, authZService authz.AuthZService,
	processorsConfig *config.ProcessorsConfig) *ProcessorsHandler {
	return &ProcessorsHandler{
		processorsService: processorsService,
		namespaceService:  namespaceService,
		vaultService:      vaultService,
		authZService:      authZService,
		processorsConfig:  processorsConfig,
	}
}

func (h ProcessorsHandler) Create(w http.ResponseWriter, r *http.Request) {

	user := h.authZService.GetValidationUser(r.Context())

	var resource public.ProcessorRequest
	cfg := &handlers.HandlerConfig{

		MarshalInto: &resource,
		Validate: []handlers.Validate{
			handlers.ValidateAsyncEnabled(r, "creating processor"),
			handlers.Validation("name", &resource.Name, handlers.WithDefault("New Processor"), handlers.MinLen(1), handlers.MaxLen(100)),
			handlers.Validation("service_account.client_id", &resource.ServiceAccount.ClientId, handlers.MinLen(1)),
			handlers.Validation("service_account.client_secret", &resource.ServiceAccount.ClientSecret, handlers.MinLen(1)),
			handlers.Validation("desired_state", (*string)(&resource.DesiredState), handlers.WithDefault("ready"), handlers.IsOneOf(dbapi.ValidProcessorDesiredStates...)),
			validateProcessorRequest(&resource),
			handlers.Validation("namespace_id", &resource.NamespaceId,
				handlers.MaxLen(maxConnectorNamespaceIdLength), user.AuthorizedNamespaceUser(errors.ErrorBadRequest), user.ValidateNamespaceConnectorQuota()),
			validateCreateAnnotations(resource.Annotations),
		},

		Action: func() (interface{}, *errors.ServiceError) {

			newID := api.NewID()
			addSystemAnnotations(&resource.Annotations, user)

			convResource, err := presenters.ConvertProcessorRequest(newID, resource)
			if err != nil {
				return nil, err
			}

			convResource.Owner = user.UserId()
			convResource.OrganisationId = user.OrgId()

			// If unassigned processors are not supported the namespace id field is required
			if !h.processorsConfig.ProcessorEnableUnassignedProcessors && (convResource.NamespaceId == nil || *convResource.NamespaceId == "") {
				return nil, errors.MinimumFieldLengthNotReached("namespace_id is not valid. Minimum length 1 is required.")
			}
			if err := ValidateProcessorOperation(r.Context(), h.namespaceService, convResource, phase.CreateProcessor); err != nil {
				return nil, err
			}

			err = moveProcessorSecretsToVault(convResource, h.vaultService)
			if err != nil {
				return nil, err
			}

			if svcErr := h.processorsService.Create(r.Context(), convResource); svcErr != nil {
				return nil, svcErr
			}

			if err := stripProcessorSecretReferences(convResource); err != nil {
				return nil, err
			}

			return presenters.PresentProcessor(convResource)
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

func (h ProcessorsHandler) Get(w http.ResponseWriter, r *http.Request) {
	processorId := mux.Vars(r)["processor_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("processor_id", &processorId, handlers.MinLen(1), handlers.MaxLen(maxProcessorIdLength)),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			resource, err := h.processorsService.Get(r.Context(), processorId)
			if err != nil {
				return nil, err
			}

			return presenters.PresentProcessorWithError(resource)
		},
	}
	handlers.HandleGet(w, r, cfg)
}

func (h *ProcessorsHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			listArgs := coreServices.NewListArguments(r.URL.Query())
			resources, paging, err := h.processorsService.List(ctx, listArgs, "")
			if err != nil {
				return nil, err
			}

			resourceList := public.ProcessorList{
				Kind:  "ProcessorList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
			}

			for _, resource := range resources {
				converted, err := presenters.PresentProcessorWithError(resource)
				if err != nil {
					glog.Errorf("processor id='%s' presentation failed: %v", resource.ID, err)
					return nil, errors.GeneralError("internal error")
				}
				resourceList.Items = append(resourceList.Items, converted)
			}

			return resourceList, nil
		},
	}

	handlers.HandleList(w, r, cfg)
}

func (h ProcessorsHandler) Patch(w http.ResponseWriter, r *http.Request) {
	cfg := &handlers.HandlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			return nil, errors.BadRequest("Not yet implemented.")
		},
	}

	// return 501 status Not Yet Implemented
	handlers.Handle(w, r, cfg, http.StatusNotImplemented)
}

func (h ProcessorsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	processorId := mux.Vars(r)["processor_id"]
	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("processor_id", &processorId, handlers.MinLen(1), handlers.MaxLen(maxProcessorIdLength)),
		},
		Action: func() (interface{}, *errors.ServiceError) {

			ctx := r.Context()
			return nil, HandleProcessorDelete(ctx, h.processorsService, h.namespaceService, processorId)
		},
	}
	handlers.HandleDelete(w, r, cfg, http.StatusNoContent)
}

func ValidateProcessorOperation(ctx context.Context, namespaceService services.ConnectorNamespaceService, processor *dbapi.Processor,
	operation phase.ProcessorOperation, updatePhase ...func(*dbapi.Processor) *errors.ServiceError) (err *errors.ServiceError) {
	if processor.NamespaceId != nil {
		var namespace *dbapi.ConnectorNamespace
		if namespace, err = namespaceService.Get(ctx, *processor.NamespaceId); err == nil {
			_, err = phase.PerformProcessorOperation(namespace, processor, operation, updatePhase...)
		}
	} else {
		// create, update and delete are the only operations allowed for processors without a namespace id
		switch operation {
		case phase.CreateProcessor, phase.UpdateProcessor, phase.DeleteProcessor:
			// valid operations
		default:
			return errors.Validation("Invalid operation %s for processor with id %s in state %s",
				operation, processor.ID, processor.DesiredState)
		}
	}
	return err
}

func HandleProcessorDelete(ctx context.Context, processorsService services.ProcessorsService,
	namespaceService services.ConnectorNamespaceService, processorId string) *errors.ServiceError {

	p, err := processorsService.Get(ctx, processorId)
	if err != nil {
		return err
	}

	// validate delete operation if processor is assigned to a namespace
	if p.NamespaceId != nil {
		err = ValidateProcessorOperation(ctx, namespaceService, &p.Processor, phase.DeleteProcessor,
			func(processor *dbapi.Processor) (err *errors.ServiceError) {
				err = processorsService.SaveStatus(ctx, processor.Status)
				if err == nil {
					err = processorsService.Update(ctx, processor)
				}
				return err
			})
	} else {
		// processor not in namespace
		p.Status.Phase = dbapi.ProcessorStatusPhaseDeleted
		p.DesiredState = dbapi.ProcessorDeleted
		err = processorsService.SaveStatus(ctx, p.Status)
		if err == nil {
			err = processorsService.Update(ctx, &p.Processor)
		}
	}
	return err
}
