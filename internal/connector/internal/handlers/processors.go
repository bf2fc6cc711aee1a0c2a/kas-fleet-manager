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
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	coreServices "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"reflect"
	"strings"
)

const (
	maxProcessorIdLength   = 32
	defaultProcessorTypeId = "processor_0.1"
)

type ProcessorsHandler struct {
	processorsService     services.ProcessorsService
	processorTypesService services.ProcessorTypesService
	namespaceService      services.ConnectorNamespaceService
	vaultService          vault.VaultService
	authZService          authz.AuthZService
	processorsConfig      *config.ProcessorsConfig
}

// this is an initial guess at what operation is being performed in update
var processorStateToOperationsMap = map[public.ProcessorDesiredState]phase.ProcessorOperation{
	public.PROCESSORDESIREDSTATE_READY:   phase.CreateProcessor,
	public.PROCESSORDESIREDSTATE_STOPPED: phase.StopProcessor,
}

func NewProcessorsHandler(processorsService services.ProcessorsService, processorTypesService services.ProcessorTypesService,
	namespaceService services.ConnectorNamespaceService, vaultService vault.VaultService, authZService authz.AuthZService,
	processorsConfig *config.ProcessorsConfig) *ProcessorsHandler {
	return &ProcessorsHandler{
		processorsService:     processorsService,
		processorTypesService: processorTypesService,
		namespaceService:      namespaceService,
		vaultService:          vaultService,
		authZService:          authZService,
		processorsConfig:      processorsConfig,
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
			handlers.Validation("namespace_id", &resource.NamespaceId,
				handlers.MinLen(1),
				handlers.MaxLen(maxConnectorNamespaceIdLength),
				user.AuthorizedNamespaceUser(errors.ErrorBadRequest),
				user.ValidateNamespaceConnectorQuota()),
			handlers.Validation("kafka.id", &resource.Kafka.Id, handlers.MinLen(1), handlers.MaxLen(maxKafkaNameLength)),
			handlers.Validation("kafka.url", &resource.Kafka.Url, handlers.MinLen(1)),
			handlers.Validation("processor_type_id", &resource.ProcessorTypeId,
				handlers.WithDefault(defaultProcessorTypeId),
				handlers.IsOneOf(defaultProcessorTypeId)),
			handlers.Validation("channel", (*string)(&resource.Channel), handlers.WithDefault("stable"), handlers.MaxLen(40)),
			handlers.Validation("service_account.client_id", &resource.ServiceAccount.ClientId, handlers.MinLen(1)),
			handlers.Validation("service_account.client_secret", &resource.ServiceAccount.ClientSecret, handlers.MinLen(1)),
			handlers.Validation("desired_state", (*string)(&resource.DesiredState), handlers.WithDefault("ready"), handlers.IsOneOf(dbapi.ValidProcessorDesiredStates...)),
			validateProcessorRequest(h.processorTypesService, &resource),
			validateCreateAnnotations(resource.Annotations),
		},

		Action: func() (interface{}, *errors.ServiceError) {

			newID := api.NewID()
			addSystemAnnotations(&resource.Annotations, user)

			convResource, err := presenters.ConvertProcessorRequest(newID, &resource)
			if err != nil {
				return nil, err
			}
			convResource.Owner = user.UserId()
			convResource.OrganisationId = user.OrgId()

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

			originalSecrets, err := getProcessorSecretRefs(&dbresource.Processor)
			if err != nil {
				return nil, errors.GeneralError("could not get existing secrets: %v", err)
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
			resource.Definition = patch.Definition
			resource.ErrorHandler = patch.ErrorHandler
			resource.Annotations = patch.Annotations
			resource.Kafka = patch.Kafka
			resource.ServiceAccount = patch.ServiceAccount

			// revalidate
			user := h.authZService.GetValidationUser(r.Context())
			validates := []handlers.Validate{
				handlers.Validation("name", &resource.Name, handlers.WithDefault("New Processor"), handlers.MinLen(1), handlers.MaxLen(100)),
				handlers.Validation("service_account.client_id", &resource.ServiceAccount.ClientId, handlers.MinLen(1)),
				handlers.Validation("desired_state", (*string)(&resource.DesiredState), handlers.WithDefault("ready"), handlers.IsOneOf(dbapi.ValidProcessorDesiredStates...)),
				validateProcessorImmutableProperties(patch, originalResource),
				validatePatchAnnotations(resource.Annotations, originalResource.Annotations),
				validateProcessor(h.processorTypesService, &resource),
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

			// If we didn't change anything, then just skip the update...
			if reflect.DeepEqual(originalResource, resource) {
				return originalResource, nil
			}

			p, svcErr := presenters.ConvertProcessor(&resource)
			if svcErr != nil {
				return nil, svcErr
			}

			svcErr = moveProcessorSecretsToVault(p, h.vaultService)
			if svcErr != nil {
				return nil, svcErr
			}

			// update processor phase before desired state
			if originalResource.Status.State != public.ProcessorState(dbapi.ProcessorStatusPhasePreparing) {
				dbresource.Status.Phase = phase.ProcessorStartingPhase[operation]
				p.Status.Phase = dbresource.Status.Phase
				serr = h.processorsService.SaveStatus(r.Context(), dbresource.Status)
				if serr != nil {
					return nil, serr
				}
			}
			// update modified processor including desired state
			serr = h.processorsService.Update(r.Context(), p)
			if serr != nil {
				return nil, serr
			}

			// A new ServiceAccount.ClientSecretRef is created for every PATCH.
			// We therefore need to set up a post commit action to delete stale secrets.
			newSecrets, err := getProcessorSecretRefs(p)
			if err != nil {
				return nil, errors.GeneralError("could not get existing secrets: %v", err)
			}

			staleSecrets := StringListSubtract(originalSecrets, newSecrets...)
			if len(staleSecrets) > 0 {
				_ = db.AddPostCommitAction(r.Context(), func() {
					for _, s := range staleSecrets {
						err = h.vaultService.DeleteSecretString(s)
						if err != nil {
							logger.Logger.Errorf("failed to delete vault secret key '%s': %v", s, err)
						}
					}
				})
			}

			if err := stripProcessorSecretReferences(p); err != nil {
				return nil, err
			}

			return presenters.PresentProcessor(p)
		},
	}

	// return 202 status accepted
	handlers.Handle(w, r, cfg, http.StatusAccepted)
}

func (h ProcessorsHandler) getOperation(resource public.Processor, patch public.ProcessorRequest) (phase.ProcessorOperation, *errors.ServiceError) {
	operation, ok := processorStateToOperationsMap[patch.DesiredState]
	if !ok {
		return operation, errors.BadRequest("Unsupported patch desired state %s", patch.DesiredState)
	}
	if patch.DesiredState == resource.DesiredState {
		// desired state not changing, it's an update
		operation = phase.UpdateProcessor
	} else {
		// creating a stopped processor is a restart
		if operation == phase.CreateProcessor && resource.DesiredState == public.PROCESSORDESIREDSTATE_STOPPED {
			operation = phase.RestartProcessor
		}
	}
	return operation, nil
}

func validateProcessorImmutableProperties(patch public.ProcessorRequest, originalResource public.Processor) handlers.Validate {
	// Check User is not attempting to patch an immutable property
	return func() *errors.ServiceError {
		immutableProperties := make([]string, 0, 3)
		if patch.NamespaceId != originalResource.NamespaceId {
			immutableProperties = append(immutableProperties, "namespace_id")
		}
		if patch.ProcessorTypeId != originalResource.ProcessorTypeId {
			immutableProperties = append(immutableProperties, "processor_type_id")
		}
		if patch.Channel != originalResource.Channel {
			immutableProperties = append(immutableProperties, "channel")
		}

		if len(immutableProperties) != 0 {
			err := errors.BadRequest("An attempt was made to modify one or more immutable field(s): %s", strings.Join(immutableProperties, ", "))
			err.HttpCode = 409
			return err
		}
		return nil
	}
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
	var namespace *dbapi.ConnectorNamespace
	if namespace, err = namespaceService.Get(ctx, processor.NamespaceId); err == nil {
		_, err = phase.PerformProcessorOperation(namespace, processor, operation, updatePhase...)
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
	err = ValidateProcessorOperation(ctx, namespaceService, &p.Processor, phase.DeleteProcessor,
		func(processor *dbapi.Processor) (err *errors.ServiceError) {
			err = processorsService.SaveStatus(ctx, processor.Status)
			if err == nil {
				err = processorsService.Update(ctx, processor)
			}
			return err
		})

	return err
}
