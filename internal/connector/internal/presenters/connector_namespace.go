package presenters

import (
	admin "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"
	"strings"
	"time"
)

var AllNamespaceTenantKinds = map[string]public.ConnectorNamespaceTenantKind{
	string(public.CONNECTORNAMESPACETENANTKIND_USER):         public.CONNECTORNAMESPACETENANTKIND_USER,
	string(public.CONNECTORNAMESPACETENANTKIND_ORGANISATION): public.CONNECTORNAMESPACETENANTKIND_ORGANISATION,
}

func ConvertConnectorNamespaceRequest(namespaceRequest *public.ConnectorNamespaceRequest,
	userID string, organisationID string) (*dbapi.ConnectorNamespace, *errors.ServiceError) {
	result := &dbapi.ConnectorNamespace{
		Model: db.Model{
			ID: api.NewID(),
		},
		Name:      namespaceRequest.Name,
		ClusterId: namespaceRequest.ClusterId,
		Owner:     userID,
		Status: dbapi.ConnectorNamespaceStatus{
			Phase: dbapi.ConnectorNamespacePhaseDisconnected,
		},
	}
	result.Annotations = make([]dbapi.ConnectorNamespaceAnnotation, len(namespaceRequest.Annotations))
	for i, annotation := range namespaceRequest.Annotations {
		result.Annotations[i].NamespaceId = result.ID
		result.Annotations[i].Name = annotation.Name
		result.Annotations[i].Value = annotation.Value
	}
	switch namespaceRequest.Kind {
	case public.CONNECTORNAMESPACETENANTKIND_USER:
		result.TenantUserId = &userID
		result.TenantUser = &dbapi.ConnectorTenantUser{
			Model: db.Model{
				ID: userID,
			},
		}
	case public.CONNECTORNAMESPACETENANTKIND_ORGANISATION:
		if organisationID == "" {
			return nil, errors.BadRequest("missing organization for tenant organisation namespace")
		}
		result.TenantOrganisationId = &organisationID
		result.TenantOrganisation = &dbapi.ConnectorTenantOrganisation{
			Model: db.Model{
				ID: organisationID,
			},
		}
	default:
		return nil, errors.BadRequest("invalid tenant kind: %s", namespaceRequest.Kind)
	}

	return result, nil
}

func ConvertConnectorNamespaceEvalRequest(namespaceRequest *public.ConnectorNamespaceEvalRequest, userID string) *dbapi.ConnectorNamespace {
	result := &dbapi.ConnectorNamespace{
		Model: db.Model{
			ID: api.NewID(),
		},
		Name:  namespaceRequest.Name,
		Owner: userID,
		Status: dbapi.ConnectorNamespaceStatus{
			Phase: dbapi.ConnectorNamespacePhaseDisconnected,
		},
	}
	result.Annotations = make([]dbapi.ConnectorNamespaceAnnotation, len(namespaceRequest.Annotations))
	for i, annotation := range namespaceRequest.Annotations {
		result.Annotations[i].NamespaceId = result.ID
		result.Annotations[i].Name = annotation.Name
		result.Annotations[i].Value = annotation.Value
	}
	result.TenantUserId = &result.Owner
	result.TenantUser = &dbapi.ConnectorTenantUser{
		Model: db.Model{
			ID: userID,
		},
	}

	return result
}

func ConvertConnectorNamespaceWithTenantRequest(namespaceRequest *admin.ConnectorNamespaceWithTenantRequest) (*dbapi.ConnectorNamespace, *errors.ServiceError) {
	result := &dbapi.ConnectorNamespace{
		Model: db.Model{
			ID: api.NewID(),
		},
		Name:      namespaceRequest.Name,
		ClusterId: namespaceRequest.ClusterId,
		Status: dbapi.ConnectorNamespaceStatus{
			Phase: dbapi.ConnectorNamespacePhaseDisconnected,
		},
	}
	switch namespaceRequest.Tenant.Kind {
	case admin.CONNECTORNAMESPACETENANTKIND_USER:
		result.TenantUserId = &namespaceRequest.Tenant.Id
		result.TenantUser = &dbapi.ConnectorTenantUser{
			Model: db.Model{
				ID: *result.TenantUserId,
			},
		}
	case admin.CONNECTORNAMESPACETENANTKIND_ORGANISATION:
		result.TenantOrganisationId = &namespaceRequest.Tenant.Id
		result.TenantOrganisation = &dbapi.ConnectorTenantOrganisation{
			Model: db.Model{
				ID: *result.TenantOrganisationId,
			},
		}
	default:
		return nil, errors.BadRequest("invalid kind %s", namespaceRequest.Tenant.Kind)
	}
	result.Annotations = make([]dbapi.ConnectorNamespaceAnnotation, len(namespaceRequest.Annotations))
	for i, annotation := range namespaceRequest.Annotations {
		result.Annotations[i].NamespaceId = result.ID
		result.Annotations[i].Name = annotation.Name
		result.Annotations[i].Value = annotation.Value
	}

	return result, nil
}

func ConvertConnectorNamespaceStatus(from private.ConnectorNamespaceStatus) *dbapi.ConnectorNamespaceStatus {
	return &dbapi.ConnectorNamespaceStatus{
		Phase:              from.Phase,
		Version:            from.Version,
		ConnectorsDeployed: from.ConnectorsDeployed,
		Conditions:         ConvertConditions(from.Conditions),
	}
}

func PresentConnectorNamespace(namespace *dbapi.ConnectorNamespace) public.ConnectorNamespace {
	annotations := make([]public.ConnectorNamespaceRequestMetaAnnotations, len(namespace.Annotations))
	for i, anno := range namespace.Annotations {
		annotations[i].Name = anno.Name
		annotations[i].Value = anno.Value
	}

	reference := PresentReference(namespace.ID, namespace)
	result := public.ConnectorNamespace{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,

		CreatedAt:  namespace.CreatedAt,
		ModifiedAt: namespace.UpdatedAt,
		Owner:      namespace.Owner,

		Name:        namespace.Name,
		ClusterId:   namespace.ClusterId,
		Tenant:      public.ConnectorNamespaceTenant{},
		Annotations: annotations,

		Status: public.ConnectorNamespaceStatus{
			State:              public.ConnectorNamespaceState(namespace.Status.Phase),
			Version:            namespace.Status.Version,
			ConnectorsDeployed: namespace.Status.ConnectorsDeployed,
			Error:              getError(namespace.Status.Conditions),
		},
	}
	if namespace.TenantUser != nil {
		result.Tenant.Kind = public.CONNECTORNAMESPACETENANTKIND_USER
		result.Tenant.Id = namespace.TenantUser.ID
	}
	if namespace.TenantOrganisation != nil {
		result.Tenant.Kind = public.CONNECTORNAMESPACETENANTKIND_ORGANISATION
		result.Tenant.Id = namespace.TenantOrganisation.ID
	}
	if namespace.Expiration != nil {
		result.Expiration = getTimestamp(*namespace.Expiration)
	}

	return result
}

func PresentPrivateConnectorNamespace(namespace *dbapi.ConnectorNamespace) admin.ConnectorNamespace {
	annotations := make([]admin.ConnectorNamespaceRequestMetaAnnotations, len(namespace.Annotations))
	for i, anno := range namespace.Annotations {
		annotations[i].Name = anno.Name
		annotations[i].Value = anno.Value
	}

	reference := PresentReference(namespace.ID, namespace)
	result := admin.ConnectorNamespace{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,

		CreatedAt:  namespace.CreatedAt,
		ModifiedAt: namespace.UpdatedAt,
		Owner:      namespace.Owner,

		Name:        namespace.Name,
		ClusterId:   namespace.ClusterId,
		Tenant:      admin.ConnectorNamespaceTenant{},
		Annotations: annotations,

		Status: admin.ConnectorNamespaceStatus{
			State:              admin.ConnectorNamespaceState(namespace.Status.Phase),
			Version:            namespace.Status.Version,
			ConnectorsDeployed: namespace.Status.ConnectorsDeployed,
			Error:              getError(namespace.Status.Conditions),
		},
	}
	if namespace.TenantUser != nil {
		result.Tenant.Kind = admin.CONNECTORNAMESPACETENANTKIND_USER
		result.Tenant.Id = namespace.TenantUser.ID
	}
	if namespace.TenantOrganisationId != nil {
		result.Tenant.Kind = admin.CONNECTORNAMESPACETENANTKIND_ORGANISATION
		result.Tenant.Id = namespace.TenantOrganisation.ID
	}
	if namespace.Expiration != nil {
		result.Expiration = getTimestamp(*namespace.Expiration)
	}

	return result
}

func getTimestamp(expiration time.Time) string {
	bytes, _ := json.Marshal(expiration)
	return strings.Trim(string(bytes), "\"")
}

func getError(conditions dbapi.ConditionList) string {
	// TODO convert conditions to error message
	return ""
}
