package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/secrets"
	"github.com/spyzhov/ajson"
)

func stripSecretReferences(resource *api.Connector, cts services.ConnectorTypesService) *errors.ServiceError {
	ct, err := cts.Get(resource.ConnectorTypeId)
	if err != nil {
		return errors.GeneralError("invalid connector type id: %s", resource.ConnectorTypeId)
	}

	// clear out secrets..
	if len(resource.ConnectorSpec) != 0 {
		updated, err := secrets.ModifySecrets(ct.JsonSchema, resource.ConnectorSpec, func(node *ajson.Node) error {
			if node.Type() == ajson.Object {
				err := node.SetObject(map[string]*ajson.Node{})
				if err != nil {
					return err
				}
			} else if node.Type() == ajson.Null {
				// don't change..
			} else {
				err := node.SetNull()
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return errors.GeneralError("could not remove connector secrets")
		}
		resource.ConnectorSpec = updated
	}
	return nil
}

func moveSecretsToVault(resource *api.Connector, cts services.ConnectorTypesService, vault services.VaultService) *errors.ServiceError {
	ct, err := cts.Get(resource.ConnectorTypeId)
	if err != nil {
		return errors.BadRequest("invalid connector type id: %s", resource.ConnectorTypeId)
	}
	// move secrets to a vault.
	if len(resource.ConnectorSpec) != 0 {
		updated, err := secrets.ModifySecrets(ct.JsonSchema, resource.ConnectorSpec, func(node *ajson.Node) error {
			if node.Type() == ajson.String {
				keyId := api.NewID()
				s, err := node.GetString()
				if err != nil {
					return err
				}
				err = vault.SetSecretString(keyId, s, "/vi/connector/"+resource.ID)
				if err != nil {
					return err
				}
				err = node.SetObject(map[string]*ajson.Node{
					"kind": ajson.StringNode("", vault.Kind()),
					"ref":  ajson.StringNode("", keyId),
				})
				if err != nil {
					return err
				}
			} else if node.Type() == ajson.Null {
				// don't change..
			} else {
				return errors.BadRequest("secret field must be set to a string: " + node.Path())
			}
			return nil
		})
		if err != nil {
			switch err := err.(type) {
			case *errors.ServiceError:
				return err
			default:
				return errors.GeneralError("could not store connectors secrets in the vault")
			}
		}
		resource.ConnectorSpec = updated
	}
	return nil
}
