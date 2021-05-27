package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/secrets"
	"github.com/spyzhov/ajson"
)

func stripSecretReferences(resource *api.Connector, ct *api.ConnectorType) *errors.ServiceError {
	// clear out secrets..
	resource.Kafka.ClientSecret = ""
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

func moveSecretsToVault(resource *api.Connector, ct *api.ConnectorType, vault services.VaultService) *errors.ServiceError {
	// move secrets to a vault.
	keyId := api.NewID()
	if err := vault.SetSecretString(keyId, resource.Kafka.ClientSecret, "/vi/connector/"+resource.ID); err != nil {
		return errors.GeneralError("could not store kafka client secret in the vault")
	}
	resource.Kafka.ClientSecret = keyId

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

func getSecretRefs(resource *api.Connector, ct *api.ConnectorType) (result []string, err error) {

	if resource.Kafka.ClientSecret != "" {
		result = append(result, resource.Kafka.ClientSecret)
	}

	// find the existing secrets...
	if len(resource.ConnectorSpec) != 0 {
		_, err := secrets.ModifySecrets(ct.JsonSchema, resource.ConnectorSpec, func(node *ajson.Node) error {
			if node.Type() != ajson.Object {
				return nil
			}
			ref, err := node.GetKey("ref")
			if err != nil {
				return nil
			}
			key, err := ref.GetString()
			if err != nil {
				return nil
			}
			result = append(result, key)
			return nil
		})
		if err != nil {
			switch err := err.(type) {
			case *errors.ServiceError:
				return result, err
			default:
				return result, errors.GeneralError("could not store connectors secrets in the vault")
			}
		}
	}
	return
}
