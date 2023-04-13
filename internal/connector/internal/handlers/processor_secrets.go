package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
)

const OwningProcessorResourcePrefix = "/v2alpha1/processor/"

func stripProcessorSecretReferences(resource *dbapi.Processor) *errors.ServiceError {
	// clear out secrets..
	resource.ServiceAccount.ClientSecret = ""
	resource.ServiceAccount.ClientSecretRef = ""
	return nil
}

func moveProcessorSecretsToVault(resource *dbapi.Processor, vault vault.VaultService) *errors.ServiceError {
	// move secrets to a vault.
	if !shared.StringEmpty(resource.ServiceAccount.ClientSecret) {
		keyId := api.NewID()
		if err := vault.SetSecretString(keyId, resource.ServiceAccount.ClientSecret, OwningProcessorResourcePrefix+resource.ID); err != nil {
			return errors.GeneralError("could not store client secret in the vault: %v", err.Error())
		}
		resource.ServiceAccount.ClientSecret = ""
		resource.ServiceAccount.ClientSecretRef = keyId
	}
	return nil
}

func getProcessorSecretRefs(resource *dbapi.Processor) (result []string, err error) {
	if !shared.StringEmpty(resource.ServiceAccount.ClientSecretRef) {
		result = append(result, resource.ServiceAccount.ClientSecretRef)
	}
	return result, nil
}
