package handlers

import (
	"context"
	"fmt"
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
	"io"
	"net/http"
	"reflect"
	"strings"
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

// this is an initial guess at what operation is being performed in update
var procsssorStateToOperationsMap = map[public.ProcessorDesiredState]phase.ProcessorOperation{
	public.PROCESSORDESIREDSTATE_UNASSIGNED: phase.UnassignProcessor,
	public.PROCESSORDESIREDSTATE_READY:      phase.AssignProcessor,
	public.PROCESSORDESIREDSTATE_STOPPED:    phase.StopProcessor,
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

func (h ProcessorsHandler) List(w http.ResponseWriter, r *http.Request) {
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

	processorId := mux.Vars(r)["processor_id"]
	contentType := r.Header.Get("Content-Type")

	cfg := &handlers.HandlerConfig{
		Validate: []handlers.Validate{
			handlers.Validation("processor_id", &processorId, handlers.MinLen(1), handlers.MaxLen(maxProcessorIdLength)),
			handlers.Validation("Content-Type header", &contentType, handlers.IsOneOf(APPLICATION_JSON, JSON_PATCH, MERGE_PATCH)),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			dbresource, serr := h.processorsService.Get(r.Context(), processorId)
			if serr != nil {
				return nil, serr
			}
			originalResource, _ := presenters.PresentProcessor(&dbresource.Processor)

			resource, serr := presenters.PresentProcessor(&dbresource.Processor)
			if serr != nil {
				return nil, serr
			}

			// Apply the patch..
			patchBytes, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, errors.BadRequest("failed to get patch bytes")
			}

			// get json resource
			resourceJson, serr := getResourceJson(resource)
			if serr != nil {
				return nil, serr
			}

			// convert json-patch into merge-patch, since validateProcessorPatch can only validate merge-patch
			if contentType == JSON_PATCH {
				patchBytes, err = convertToMergePatch(patchBytes, resourceJson)
				if err != nil {
					return nil, errors.BadRequest("failed to convert to merge patch: %v", err)
				}
				// changed patch bytes to merge patch
				contentType = MERGE_PATCH
			}

			patch := public.ProcessorRequest{}
			serr = PatchResource(resourceJson, contentType, patchBytes, &patch)
			if serr != nil {
				return nil, serr
			}

			// get and validate patch operation type
			var operation phase.ProcessorOperation
			if operation, serr = h.getOperation(resource, patch); err != nil {
				return nil, serr
			}
			if operation == phase.UnassignProcessor && !h.processorsConfig.ProcessorEnableUnassignedProcessors {
				return nil, errors.FieldValidationError("Unsupported processor state %s", patch.DesiredState)
			}
			if serr = ValidateProcessorOperation(r.Context(), h.namespaceService, &dbresource.Processor, operation,
				func(processor *dbapi.Processor) *errors.ServiceError {
					resource.DesiredState = public.ProcessorDesiredState(dbresource.DesiredState)
					return nil
				}); serr != nil {
				return nil, serr
			}

			// But we don't want to allow the user to update ALL fields... so copy
			// over the fields that they are allowed to modify...
			resource.Name = patch.Name
			resource.Processor = patch.Processor
			resource.Annotations = patch.Annotations
			resource.ServiceAccount = patch.ServiceAccount

			if h.processorsConfig.ProcessorEnableUnassignedProcessors {
				// check namespace id change, from unassigned to assigned and vice versa
				if operation == phase.AssignProcessor && patch.NamespaceId != "" && resource.NamespaceId == "" {
					resource.NamespaceId = patch.NamespaceId
				}
				if operation == phase.UnassignProcessor && patch.NamespaceId == "" && resource.NamespaceId != "" {
					resource.NamespaceId = patch.NamespaceId
				}
			}

			// If we didn't change anything, then just skip the update...
			if reflect.DeepEqual(originalResource, resource) {
				return originalResource, nil
			}

			// revalidate
			user := h.authZService.GetValidationUser(r.Context())
			validates := []handlers.Validate{
				handlers.Validation("name", &resource.Name, handlers.WithDefault("New Processor"), handlers.MinLen(1), handlers.MaxLen(100)),
				handlers.Validation("service_account.client_id", &resource.ServiceAccount.ClientId, handlers.MinLen(1)),
				handlers.Validation("desired_state", (*string)(&resource.DesiredState), handlers.WithDefault("ready"), handlers.IsOneOf(dbapi.ValidProcessorDesiredStates...)),
				validatePatchAnnotations(resource.Annotations, originalResource.Annotations),
				validateProcessor(&resource),
			}

			// Don't validate user's tenancy in admin api calls
			// TODO {manstis} Processors are not exposed under the admin API at the moment
			if strings.Compare(r.URL.Path, fmt.Sprintf("%s/%s", "/api/connector_mgmt/v1/admin/processors", processorId)) != 0 {
				validates = append(validates, handlers.Validation("namespace_id", &resource.NamespaceId, handlers.MaxLen(maxConnectorNamespaceIdLength), user.AuthorizedNamespaceUser(errors.ErrorBadRequest)))
			}

			for _, v := range validates {
				err := v()
				if err != nil {
					return nil, err
				}
			}

			p, svcErr := presenters.ConvertProcessor(resource)
			if svcErr != nil {
				return nil, svcErr
			}

			// update processor phase before desired state
			if originalResource.Status.State != public.ProcessorState(dbapi.ProcessorStatusPhaseAssigning) {
				dbresource.Status.Phase = phase.ProcessorStartingPhase[operation]
				p.Status.Phase = dbresource.Status.Phase
				serr = h.processorsService.SaveStatus(r.Context(), dbresource.Status)
				if serr != nil {
					return nil, serr
				}
			}
			// update modified connector including desired state
			serr = h.processorsService.Update(r.Context(), p)
			if serr != nil {
				return nil, serr
			}

			return presenters.PresentProcessor(p)
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

func (h ProcessorsHandler) getOperation(resource public.Processor, patch public.ProcessorRequest) (phase.ProcessorOperation, *errors.ServiceError) {
	operation, ok := procsssorStateToOperationsMap[patch.DesiredState]
	if !ok {
		return operation, errors.BadRequest("Unsupported patch desired state %s", patch.DesiredState)
	}
	if patch.DesiredState == resource.DesiredState {
		// desired state not changing, it's an update
		operation = phase.UpdateProcessor
	} else {
		// assigning a stopped connector is a restart
		if operation == phase.AssignProcessor && resource.DesiredState == public.PROCESSORDESIREDSTATE_STOPPED {
			operation = phase.RestartProcessor
		}
	}
	return operation, nil
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
