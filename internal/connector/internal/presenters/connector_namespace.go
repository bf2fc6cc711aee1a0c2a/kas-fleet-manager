package presenters

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"strings"
	"time"
)

const (
	UserKind         string = "user"
	OrganisationKind string = "organisation"
)

func ConvertConnectorNamespaceRequest(namespaceRequest *public.ConnectorNamespaceRequest) *dbapi.ConnectorNamespace {
	result := &dbapi.ConnectorNamespace{
		Model: db.Model{
			ID: api.NewID(),
		},
		Name:      namespaceRequest.Name,
		ClusterId: namespaceRequest.ClusterId,
	}
	result.Annotations = make([]dbapi.ConnectorNamespaceAnnotation, len(namespaceRequest.Annotations))
	for i, annotation := range namespaceRequest.Annotations {
		result.Annotations[i].NamespaceId = result.ID
		result.Annotations[i].Name = annotation.Name
		result.Annotations[i].Value = annotation.Value
	}

	return result
}

func ConvertConnectorNamespaceEvalRequest(namespaceRequest *public.ConnectorNamespaceEvalRequest) *dbapi.ConnectorNamespace {
	result := &dbapi.ConnectorNamespace{
		Model: db.Model{
			ID: api.NewID(),
		},
		Name: namespaceRequest.Name,
	}
	result.Annotations = make([]dbapi.ConnectorNamespaceAnnotation, len(namespaceRequest.Annotations))
	for i, annotation := range namespaceRequest.Annotations {
		result.Annotations[i].NamespaceId = result.ID
		result.Annotations[i].Name = annotation.Name
		result.Annotations[i].Value = annotation.Value
	}

	return result
}

func ConvertConnectorNamespaceWithTenantRequest(namespaceRequest *private.ConnectorNamespaceWithTenantRequest) *dbapi.ConnectorNamespace {
	result := &dbapi.ConnectorNamespace{
		Model: db.Model{
			ID: api.NewID(),
		},
		Name:      namespaceRequest.Name,
		ClusterId: namespaceRequest.ClusterId,
	}
	switch namespaceRequest.Tenant.Kind {
	case UserKind:
		result.TenantUserId = &namespaceRequest.Tenant.UserId
		result.TenantUser = &dbapi.ConnectorTenantUser{
			Model: db.Model{
				ID: *result.TenantUserId,
			},
		}
	case OrganisationKind:
		result.TenantOrganisationId = &namespaceRequest.Tenant.OrganisationId
		result.TenantOrganisation = &dbapi.ConnectorTenantOrganisation{
			Model: db.Model{
				ID: *result.TenantOrganisationId,
			},
		}
	default:
		// ignore, should have been validated earlier
	}
	result.Annotations = make([]dbapi.ConnectorNamespaceAnnotation, len(namespaceRequest.Annotations))
	for i, annotation := range namespaceRequest.Annotations {
		result.Annotations[i].NamespaceId = result.ID
		result.Annotations[i].Name = annotation.Name
		result.Annotations[i].Value = annotation.Value
	}

	return result
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
		Version:    namespace.Version,

		Name:        namespace.Name,
		ClusterId:   namespace.ClusterId,
		Tenant:      public.ConnectorNamespaceTenant{},
		Annotations: annotations,
	}
	if namespace.TenantUser != nil {
		result.Tenant.Kind = UserKind
		result.Tenant.UserId = namespace.TenantUser.ID
	}
	if namespace.TenantOrganisation != nil {
		result.Tenant.Kind = OrganisationKind
		result.Tenant.OrganisationId = namespace.TenantOrganisation.ID
	}
	if namespace.Expiration != nil {
		result.Expiration = getTimestamp(*namespace.Expiration)
	}

	return result
}

func PresentPrivateConnectorNamespace(namespace *dbapi.ConnectorNamespace) private.ConnectorNamespace {
	annotations := make([]private.ConnectorNamespaceRequestMetaAnnotations, len(namespace.Annotations))
	for i, anno := range namespace.Annotations {
		annotations[i].Name = anno.Name
		annotations[i].Value = anno.Value
	}

	reference := PresentReference(namespace.ID, namespace)
	result := private.ConnectorNamespace{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,

		CreatedAt:  namespace.CreatedAt,
		ModifiedAt: namespace.UpdatedAt,
		Owner:      namespace.Owner,
		Version:    namespace.Version,

		Name:        namespace.Name,
		ClusterId:   namespace.ClusterId,
		Tenant:      private.ConnectorNamespaceTenant{},
		Annotations: annotations,
	}
	if namespace.TenantUser != nil {
		result.Tenant.Kind = UserKind
		result.Tenant.UserId = namespace.TenantUser.ID
	}
	if namespace.TenantOrganisationId != nil {
		result.Tenant.Kind = OrganisationKind
		result.Tenant.OrganisationId = namespace.TenantOrganisation.ID
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
