package services

import (
	"context"
	goerrors "errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services/signalbus"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared/secrets"
	"github.com/spyzhov/ajson"

	"gorm.io/gorm"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

type ConnectorsService interface {
	Create(ctx context.Context, resource *dbapi.Connector) *errors.ServiceError
	Get(ctx context.Context, id string, tid string) (*dbapi.Connector, *errors.ServiceError)
	List(ctx context.Context, kid string, listArgs *services.ListArguments, tid string) (dbapi.ConnectorList, *api.PagingMeta, *errors.ServiceError)
	Update(ctx context.Context, resource *dbapi.Connector) *errors.ServiceError
	SaveStatus(ctx context.Context, resource dbapi.ConnectorStatus) *errors.ServiceError
	Delete(ctx context.Context, id string) *errors.ServiceError
	ForEach(f func(*dbapi.Connector) *errors.ServiceError, query string, args ...interface{}) *errors.ServiceError
}

var _ ConnectorsService = &connectorsService{}

type connectorsService struct {
	connectionFactory     *db.ConnectionFactory
	bus                   signalbus.SignalBus
	vaultService          vault.VaultService
	connectorTypesService ConnectorTypesService
}

func NewConnectorsService(connectionFactory *db.ConnectionFactory, bus signalbus.SignalBus,
	vaultService vault.VaultService, connectorTypesService ConnectorTypesService) *connectorsService {
	return &connectorsService{
		connectionFactory:     connectionFactory,
		bus:                   bus,
		vaultService:          vaultService,
		connectorTypesService: connectorTypesService,
	}
}

// Create creates a connector in the database
func (k *connectorsService) Create(ctx context.Context, resource *dbapi.Connector) *errors.ServiceError {
	//kid := resource.KafkaID
	//if kid == "" {
	//	return errors.Validation("kafka id is undefined")
	//}

	dbConn := k.connectionFactory.New()
	if err := dbConn.Save(resource).Error; err != nil {
		return errors.GeneralError("failed to create connector: %v", err)
	}

	// read it back.... to get the updated version...
	if err := dbConn.Where("id = ?", resource.ID).First(&resource).Error; err != nil {
		return services.HandleGetError("Connector", "id", resource.ID, err)
	}

	resource.Status.ID = resource.ID
	resource.Status.Phase = dbapi.ConnectorStatusPhaseAssigning
	if err := dbConn.Save(&resource.Status).Error; err != nil {
		return errors.GeneralError("failed to save status: %v", err)
	}

	_ = db.AddPostCommitAction(ctx, func() {
		// Wake up the reconcile loop...
		k.bus.Notify("reconcile:connector")
	})

	// TODO: increment connector metrics
	// metrics.IncreaseStatusCountMetric(constants.KafkaRequestStatusAccepted.String())
	return nil
}

// Get gets a connector by id from the database
func (k *connectorsService) Get(ctx context.Context, id string, tid string) (*dbapi.Connector, *errors.ServiceError) {
	if id == "" {
		return nil, errors.Validation("connector id is undefined")
	}

	dbConn := k.connectionFactory.New()
	var resource dbapi.Connector
	dbConn = dbConn.Where("id = ?", id)
	dbConn = dbConn.Preload("Status")

	var err *errors.ServiceError
	dbConn, err = filterConnectorsToOwnerOrOrg(ctx, dbConn, k.connectionFactory)
	if err != nil {
		return nil, err
	}

	if tid != "" {
		dbConn = dbConn.Where("connector_type_id = ?", tid)
	}

	if err := dbConn.First(&resource).Error; err != nil {
		return nil, services.HandleGetError("Connector", "id", id, err)
	}
	return &resource, nil
}

func filterConnectorsToOwnerOrOrg(ctx context.Context, dbConn *gorm.DB, factory *db.ConnectionFactory) (*gorm.DB, *errors.ServiceError) {

	claims, err := auth.GetClaimsFromContext(ctx)
	if err != nil {
		return dbConn, errors.Unauthenticated("user not authenticated")
	}
	owner := auth.GetUsernameFromClaims(claims)
	if owner == "" {
		return dbConn, errors.Unauthenticated("user not authenticated")
	}

	orgId := auth.GetOrgIdFromClaims(claims)
	filterByOrganisationId := auth.GetFilterByOrganisationFromContext(ctx)

	// filter by organisationId if a user is part of an organisation and is not allowed as a service account
	if filterByOrganisationId {
		// unassigned connectors with no namespace_id use owner and org
		// assigned connectors use tenant user or organisation
		dbConn = dbConn.Where("(namespace_id is null AND (owner = ? or organisation_id = ?)) OR (namespace_id is not null AND namespace_id IN (?))",
			owner,
			orgId,
			factory.New().Table("connector_namespaces").Select("id").
				Where("deleted_at is null AND (tenant_user_id = ? OR tenant_organisation_id = ?)", owner, orgId))
	} else {
		dbConn = dbConn.Where("owner = ?", owner)
	}
	return dbConn, nil
}

// Delete deletes a connector from the database.
func (k *connectorsService) Delete(ctx context.Context, id string) *errors.ServiceError {
	if id == "" {
		return errors.Validation("id is undefined")
	}
	dbConn := k.connectionFactory.New()

	var resource dbapi.Connector
	if err := dbConn.Where("id = ?", id).First(&resource).Error; err != nil {
		return services.HandleGetError("Connector", "id", id, err)
	}
	if err := dbConn.Delete(&resource).Error; err != nil {
		return errors.GeneralError("unable to delete connector with id %s: %s", resource.ID, err)
	}

	// delete the associated relations
	if err := dbConn.Where("id = ?", id).Delete(&dbapi.ConnectorStatus{}).Error; err != nil {
		return services.HandleGetError("ConnectorStatus", "id", id, err)
	}

	_ = db.AddPostCommitAction(ctx, func() {
		// delete related distributed resources...

		if resource.ServiceAccount.ClientSecretRef != "" {
			err := k.vaultService.DeleteSecretString(resource.ServiceAccount.ClientSecretRef)
			if err != nil {
				logger.Logger.Errorf("failed to delete vault secret key '%s': %v", resource.ServiceAccount.ClientSecretRef, err)
			}
		}

		if len(resource.ConnectorSpec) != 0 {
			if ct, err := k.connectorTypesService.Get(resource.ConnectorTypeId); err == nil {
				_, _ = secrets.ModifySecrets(ct.JsonSchema, resource.ConnectorSpec, func(node *ajson.Node) error {
					if node.Type() != ajson.Object {
						return nil
					}
					ref, err := node.GetKey("ref")
					if err != nil {
						return nil
					}
					r, err := ref.GetString()
					if err != nil {
						return nil
					}
					err = k.vaultService.DeleteSecretString(r)
					if err != nil {
						logger.Logger.Errorf("failed to delete vault secret key '%s': %v", r, err)
					}
					return nil
				})
			}
		}
	})

	return nil
}

// List returns all connectors visible to the user within the requested paging window.
func (k *connectorsService) List(ctx context.Context, kafka_id string, listArgs *services.ListArguments, tid string) (dbapi.ConnectorList, *api.PagingMeta, *errors.ServiceError) {
	var resourceList dbapi.ConnectorList
	dbConn := k.connectionFactory.New()
	dbConn = dbConn.Preload("Status")
	pagingMeta := &api.PagingMeta{
		Page: listArgs.Page,
		Size: listArgs.Size,
	}

	if kafka_id != "" {
		dbConn = dbConn.Where("kafka_id = ?", kafka_id)
	}

	var err *errors.ServiceError
	dbConn, err = filterConnectorsToOwnerOrOrg(ctx, dbConn, k.connectionFactory)
	if err != nil {
		return nil, nil, err
	}

	if tid != "" {
		dbConn = dbConn.Where("connector_type_id = ?", tid)
	}

	// set total, limit and paging (based on https://gitlab.cee.redhat.com/service/api-guidelines#user-content-paging)
	total := int64(pagingMeta.Total)
	dbConn.Model(&resourceList).Count(&total)
	pagingMeta.Total = int(total)
	if pagingMeta.Size > pagingMeta.Total {
		pagingMeta.Size = pagingMeta.Total
	}
	dbConn = dbConn.Offset((pagingMeta.Page - 1) * pagingMeta.Size).Limit(pagingMeta.Size)

	// default the order by name
	dbConn = dbConn.Order("name")

	// execute query
	if err := dbConn.Find(&resourceList).Error; err != nil {
		return resourceList, pagingMeta, errors.GeneralError("Unable to list connectors: %s", err)
	}

	return resourceList, pagingMeta, nil
}

func (k connectorsService) Update(ctx context.Context, resource *dbapi.Connector) *errors.ServiceError {

	// If the version is set, then lets verify that the version has not changed...
	if resource.Version != 0 {
		dbConn := k.connectionFactory.New()
		dbConn = dbConn.Where("id = ? AND version = ?", resource.ID, resource.Version)
		t := dbapi.Connector{}
		if err := dbConn.First(&t).Error; err != nil {
			return errors.BadRequest("resource version changed")
		}
	}

	dbConn := k.connectionFactory.New()
	if err := dbConn.Model(resource).Updates(resource).Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}

	// read it back.... to get the updated version...
	dbConn = k.connectionFactory.New().Where("id = ?", resource.ID)
	if err := dbConn.First(&resource).Error; err != nil {
		return services.HandleGetError("Connector", "id", resource.ID, err)
	}

	_ = db.AddPostCommitAction(ctx, func() {
		// Wake up the reconcile loop...
		k.bus.Notify("reconcile:connector")
	})

	return nil
}

func (k connectorsService) SaveStatus(ctx context.Context, resource dbapi.ConnectorStatus) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	if err := dbConn.Model(resource).Save(resource).Error; err != nil {
		return errors.GeneralError("failed to update: %s", err.Error())
	}
	return nil
}

func (k connectorsService) ForEach(f func(*dbapi.Connector) *errors.ServiceError, query string, args ...interface{}) *errors.ServiceError {
	dbConn := k.connectionFactory.New()
	rows, err := dbConn.
		Model(&dbapi.Connector{}).
		Where(query, args...).
		Joins("left join connector_statuses on connector_statuses.id = connectors.id").
		Order("version").Rows()

	if err != nil {
		if goerrors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return errors.GeneralError("Unable to list connectors: %s", err)
	}
	defer rows.Close()

	for rows.Next() {
		resource := dbapi.Connector{}

		// ScanRows is a method of `gorm.DB`, it can be used to scan a row into a struct
		err := dbConn.ScanRows(rows, &resource)
		if err != nil {
			return errors.GeneralError("Unable to scan connector: %s", err)
		}

		resource.Status.ID = resource.ID
		err = dbConn.Model(&dbapi.ConnectorStatus{}).First(&resource.Status).Error
		if err != nil {
			return errors.GeneralError("Unable to load connector status: %s", err)
		}

		if serr := f(&resource); serr != nil {
			return serr
		}

	}
	return nil
}
