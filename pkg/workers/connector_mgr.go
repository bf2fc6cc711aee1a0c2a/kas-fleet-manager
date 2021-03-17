package workers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/presenters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/secrets"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
	"github.com/spyzhov/ajson"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

// ConnectorManager represents a connector manager that periodically reconciles connector requests
type ConnectorManager struct {
	id                      string
	workerType              string
	isRunning               bool
	imStop                  chan struct{}
	syncTeardown            sync.WaitGroup
	reconciler              Reconciler
	connectorService        services.ConnectorsService
	connectorClusterService services.ConnectorClusterService
	observatoriumService    services.ObservatoriumService
	connectorTypesService   services.ConnectorTypesService
	vaultService            services.VaultService
}

// NewConnectorManager creates a new connector manager
func NewConnectorManager(id string, connectorTypesService services.ConnectorTypesService, connectorService services.ConnectorsService, connectorClusterService services.ConnectorClusterService, observatoriumService services.ObservatoriumService, vaultService services.VaultService) *ConnectorManager {
	return &ConnectorManager{
		id:                      id,
		workerType:              "connector",
		connectorService:        connectorService,
		connectorClusterService: connectorClusterService,
		observatoriumService:    observatoriumService,
		connectorTypesService:   connectorTypesService,
		vaultService:            vaultService,
	}
}

func (k *ConnectorManager) GetStopChan() *chan struct{} {
	return &k.imStop
}

func (k *ConnectorManager) GetSyncGroup() *sync.WaitGroup {
	return &k.syncTeardown
}

func (k *ConnectorManager) GetID() string {
	return k.id
}

func (c *ConnectorManager) GetWorkerType() string {
	return c.workerType
}

// Start initializes the connector manager to reconcile connector requests
func (k *ConnectorManager) Start() {
	k.reconciler.Start(k)
}

// Stop causes the process for reconciling connector requests to stop.
func (k *ConnectorManager) Stop() {
	k.reconciler.Stop(k)
}

func (c *ConnectorManager) IsRunning() bool {
	return c.isRunning
}

func (c *ConnectorManager) SetIsRunning(val bool) {
	c.isRunning = val
}

func (k *ConnectorManager) reconcile() {
	glog.V(5).Infoln("reconciling connectors")

	statuses := []api.ConnectorStatus{
		api.ConnectorStatusAssigning,
		api.ConnectorStatusDeleted,
	}
	serviceErr := k.connectorService.ForEachInStatus(statuses, func(connector *api.Connector) *errors.ServiceError {

		ctx, err := db.NewContext(context.Background())
		if err != nil {
			return errors.GeneralError("failed to create tx: %v", err)
		}
		err = db.Begin(ctx)
		if err != nil {
			return errors.GeneralError("failed to create tx: %v", err)
		}
		defer func() {
			err := db.Resolve(ctx)
			if err != nil {
				sentry.CaptureException(err)
				glog.Errorf("failed to resolve tx: %s", err.Error())
			}
		}()

		switch connector.Status {
		case api.ConnectorStatusAssigning:
			if err := k.reconcileAccepted(ctx, connector); err != nil {
				sentry.CaptureException(err)
				glog.Errorf("failed to reconcile accepted connector %s: %s", connector.ID, err.Error())
			}
		case api.ConnectorClusterPhaseDeleted:
			if err := k.reconcileDeleted(ctx, connector); err != nil {
				sentry.CaptureException(err)
				glog.Errorf("failed to reconcile accepted connector %s: %s", connector.ID, err.Error())
			}
		}
		return nil
	})
	if serviceErr != nil {
		sentry.CaptureException(serviceErr)
		glog.Errorf("failed to list connectors: %s", serviceErr.Error())
	}
}

func (k *ConnectorManager) reconcileAccepted(ctx context.Context, connector *api.Connector) error {
	switch connector.TargetKind {
	case api.AddonTargetKind:

		cluster, err := k.connectorClusterService.FindReadyCluster(connector.Owner, connector.OrganisationId, connector.AddonClusterId)
		if err != nil {
			return fmt.Errorf("failed to find cluster for connector request %s: %w", connector.ID, err)
		}
		if cluster != nil {

			address, serr := k.connectorTypesService.GetServiceAddress(connector.ConnectorTypeId)
			if serr != nil {
				return fmt.Errorf("failed to get service address for connector type %s: %w", connector.ConnectorTypeId, err)
			}

			deployment := api.ConnectorDeployment{
				Meta: api.Meta{
					ID: api.NewID(),
				},
				ClusterID:            cluster.ID,
				ConnectorTypeService: address,
			}

			clusterStatus := presenters.PresentConnectorClusterStatus(cluster.Status)

			serr = getSecretsFromVaultAsBase64(connector, k.connectorTypesService, k.vaultService)
			if serr != nil {
				return fmt.Errorf("failed to extract secrets from the vault: %v", serr)
			}

			connectorSpec, err := connector.ConnectorSpec.Object()
			if err != nil {
				return err
			}

			apiSpec, err := k.reifY(context.Background(), address, connector.ConnectorTypeId, openapi.ConnectorReifyRequest{
				ConnectorId:     connector.ID,
				ResourceVersion: connector.Version,
				KafkaId:         connector.KafkaID,
				ClusterId:       cluster.ID,
				ClusterStatus:   clusterStatus,
				ConnectorSpec:   connectorSpec,
			})
			if err != nil {
				return fmt.Errorf("failed to reify deployment for conenctor %s: %w", connector.ID, err)
			}

			spec, serr := presenters.ConvertConnectorDeploymentSpec(apiSpec)
			if serr != nil {
				return err
			}

			deployment.Spec = spec
			deployment.Spec.ConnectorId = connector.ID
			if serr = k.connectorClusterService.CreateDeployment(ctx, &deployment); serr != nil {
				return fmt.Errorf("failed to create connector deployment for connector %s: %w", connector.ID, err)
			}

			connector.ClusterID = cluster.ID
			connector.Status = api.ConnectorStatusAssigned
			if serr = k.connectorService.Update(ctx, connector); serr != nil {
				return fmt.Errorf("failed to update connector %s with cluster details: %w", connector.ID, err)
			}

		}

	default:
		return fmt.Errorf("target kind not supported: %s", connector.TargetKind)
	}
	return nil
}

func getSecretsFromVaultAsBase64(resource *api.Connector, cts services.ConnectorTypesService, vault services.VaultService) *errors.ServiceError {
	ct, err := cts.Get(resource.ConnectorTypeId)
	if err != nil {
		return errors.BadRequest("invalid connector type id: %s", resource.ConnectorTypeId)
	}
	// move secrets to a vault.
	if len(resource.ConnectorSpec) != 0 {
		updated, err := secrets.ModifySecrets(ct.JsonSchema, resource.ConnectorSpec, func(node *ajson.Node) error {
			if node.Type() == ajson.Object {
				ref, err := node.GetKey("ref")
				if err != nil {
					return err
				}
				r, err := ref.GetString()
				if err != nil {
					return err
				}
				v, err := vault.GetSecretString(r)
				if err != nil {
					return err
				}

				encoded := base64.StdEncoding.EncodeToString([]byte(v))
				err = node.SetObject(map[string]*ajson.Node{
					"kind":  ajson.StringNode("", "base64"),
					"value": ajson.StringNode("", encoded),
				})
				if err != nil {
					return err
				}
			} else if node.Type() == ajson.Null {
				// don't change..
			} else {
				return fmt.Errorf("secret field must be set to an object: " + node.Path())
			}
			return nil
		})
		if err != nil {
			return errors.GeneralError("could not store connectors secrets in the vault")
		}
		resource.ConnectorSpec = updated
	}
	return nil
}

func (k *ConnectorManager) reifY(ctx context.Context, serviceUrl string, typeId string, reifyRequest openapi.ConnectorReifyRequest) (openapi.ConnectorDeploymentSpec, error) {

	result := openapi.ConnectorDeploymentSpec{}
	url := fmt.Sprintf("%s/api/managed-services-api/v1/kafka-connector-types/%s/reify/spec", serviceUrl, typeId)

	requestBytes, err := json.Marshal(reifyRequest)
	if err != nil {
		return result, err
	}

	// glog.Infoln("reify request: ", string(requestBytes))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(requestBytes))
	if err != nil {
		return result, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	c := &http.Client{}

	resp, err := c.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		all, err := ioutil.ReadAll(io.LimitReader(resp.Body, 400))
		if err != nil {
			return result, err
		}
		return result, fmt.Errorf("expected 200: but got %d, body: %s", resp.StatusCode, string(all))
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		return result, fmt.Errorf("expected Content-Type: application/json, but got %s", contentType)
	}

	//all, err := io.ReadAll(resp.Body)
	//if err != nil {
	//	return result, err
	//}
	//glog.Infoln("reify response: ", string(all))
	//
	//err = json.Unmarshal(all, &result)
	//if err != nil {
	//	return result, err
	//}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, nil

}

func (k *ConnectorManager) reconcileDeleted(ctx context.Context, connector *api.Connector) error {
	if err := k.connectorService.Delete(ctx, connector.ID); err != nil {
		return fmt.Errorf("failed to delete connector %s: %w", connector.ID, err)
	}
	return nil
}
