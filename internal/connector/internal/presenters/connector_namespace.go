package presenters

import (
	"fmt"
	admin "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/profiles"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/handlers"
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

	result.Annotations = ConvertNamespaceAnnotations(result.ID, namespaceRequest.Annotations)

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

	result.Annotations = ConvertNamespaceAnnotations(result.ID, namespaceRequest.Annotations)

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
	if namespaceRequest.Expiration != "" {
		expiration, err := time.Parse(time.RFC3339, namespaceRequest.Expiration)
		if err != nil {
			return nil, errors.BadRequest("invalid namespace expiration '%s': %s", namespaceRequest.Expiration, err)
		}
		result.Expiration = &expiration
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

	result.Annotations = ConvertNamespaceAnnotations(result.ID, namespaceRequest.Annotations)

	return result, nil
}

func ConvertConnectorNamespaceDeploymentStatus(from private.ConnectorNamespaceDeploymentStatus) *dbapi.ConnectorNamespaceStatus {
	return &dbapi.ConnectorNamespaceStatus{
		Phase:              dbapi.ConnectorNamespacePhaseEnum(from.Phase),
		Version:            from.Version,
		ConnectorsDeployed: from.ConnectorsDeployed,
		Conditions:         ConvertConditions(from.Conditions),
	}
}

func PresentConnectorNamespace(namespace *dbapi.ConnectorNamespace, quotaConfig *config.ConnectorsQuotaConfig) public.ConnectorNamespace {

	var quota config.NamespaceQuota
	annotations := make(map[string]string, len(namespace.Annotations))
	for _, anno := range namespace.Annotations {
		annotations[anno.Key] = anno.Value
		if anno.Key == profiles.AnnotationProfileKey {
			// TODO handle unknown profiles instead of using defaults
			quota, _ = quotaConfig.GetNamespaceQuota(anno.Value)
		}
	}

	reference := PresentReference(namespace.ID, namespace)
	result := public.ConnectorNamespace{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,

		CreatedAt:       namespace.CreatedAt,
		ModifiedAt:      namespace.UpdatedAt,
		Owner:           namespace.Owner,
		ResourceVersion: namespace.Version,
		Quota: public.ConnectorNamespaceQuota{
			Connectors:     quota.Connectors,
			MemoryRequests: quota.MemoryRequests,
			MemoryLimits:   quota.MemoryLimits,
			CpuRequests:    quota.CPURequests,
			CpuLimits:      quota.CPULimits,
		},

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

func PresentConnectorNamespaceDeployment(namespace *dbapi.ConnectorNamespace, quotaConfig *config.ConnectorsQuotaConfig) private.ConnectorNamespaceDeployment {
	var quota config.NamespaceQuota
	annotations := make(map[string]string, len(namespace.Annotations))
	for _, anno := range namespace.Annotations {
		annotations[anno.Key] = anno.Value
		if anno.Key == profiles.AnnotationProfileKey {
			// TODO handle unknown profiles instead of using defaults
			quota, _ = quotaConfig.GetNamespaceQuota(anno.Value)
		}
	}

	reference := handlers.PresentReferenceWith(
		namespace.ID,
		namespace,
		func(i interface{}) string {
			return "ConnectorNamespaceDeployment"
		},
		func(id string, obj interface{}) string {
			return fmt.Sprintf("/api/connector_mgmt/v1/agent/kafka_connector_clusters/%s/namespaces/%s", namespace.ClusterId, namespace.ID)
		})

	result := private.ConnectorNamespaceDeployment{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,

		CreatedAt:       namespace.CreatedAt,
		ModifiedAt:      namespace.UpdatedAt,
		Owner:           namespace.Owner,
		ResourceVersion: namespace.Version,
		Quota: private.ConnectorNamespaceQuota{
			Connectors:     quota.Connectors,
			MemoryRequests: quota.MemoryRequests,
			MemoryLimits:   quota.MemoryLimits,
			CpuRequests:    quota.CPURequests,
			CpuLimits:      quota.CPULimits,
		},

		Name:        namespace.Name,
		ClusterId:   namespace.ClusterId,
		Tenant:      private.ConnectorNamespaceTenant{},
		Annotations: annotations,

		Status: private.ConnectorNamespaceStatus{
			State:              private.ConnectorNamespaceState(namespace.Status.Phase),
			Version:            namespace.Status.Version,
			ConnectorsDeployed: namespace.Status.ConnectorsDeployed,
			Error:              getError(namespace.Status.Conditions),
		},
	}
	if namespace.TenantUser != nil {
		result.Tenant.Kind = private.CONNECTORNAMESPACETENANTKIND_USER
		result.Tenant.Id = namespace.TenantUser.ID
	}
	if namespace.TenantOrganisation != nil {
		result.Tenant.Kind = private.CONNECTORNAMESPACETENANTKIND_ORGANISATION
		result.Tenant.Id = namespace.TenantOrganisation.ID
	}
	if namespace.Expiration != nil {
		result.Expiration = getTimestamp(*namespace.Expiration)
	}

	return result
}

func PresentPrivateConnectorNamespace(namespace *dbapi.ConnectorNamespace, quotaConfig *config.ConnectorsQuotaConfig) admin.ConnectorNamespace {

	var quota config.NamespaceQuota
	annotations := make(map[string]string, len(namespace.Annotations))
	for _, anno := range namespace.Annotations {
		annotations[anno.Key] = anno.Value
		if anno.Key == profiles.AnnotationProfileKey {
			quota, _ = quotaConfig.GetNamespaceQuota(anno.Value)
		}
	}

	reference := PresentReference(namespace.ID, namespace)
	result := admin.ConnectorNamespace{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,

		CreatedAt:       namespace.CreatedAt,
		ModifiedAt:      namespace.UpdatedAt,
		Owner:           namespace.Owner,
		ResourceVersion: namespace.Version,
		Quota: admin.ConnectorNamespaceQuota{
			Connectors:     quota.Connectors,
			MemoryRequests: quota.MemoryRequests,
			MemoryLimits:   quota.MemoryLimits,
			CpuRequests:    quota.CPURequests,
			CpuLimits:      quota.CPULimits,
		},

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

func PresentNamespaceAnnotations(annotations []dbapi.ConnectorNamespaceAnnotation) map[string]string {
	if len(annotations) == 0 {
		return nil
	}
	res := make(map[string]string)
	for _, ann := range annotations {
		res[ann.Key] = ann.Value
	}
	return res
}

func getTimestamp(expiration time.Time) string {
	bytes, _ := json.Marshal(expiration)
	return strings.Trim(string(bytes), "\"")
}

func getError(conditions dbapi.ConditionList) string {
	var err []string
	for _, condition := range conditions {
		// look for *Failure type conditions
		if strings.HasSuffix(condition.Type, "Failure") {
			err = append(err, fmt.Sprintf("%s: %s", condition.Reason, condition.Message))
		}
	}
	return strings.Join(err, "; ")
}

func ConvertNamespaceAnnotations(namespaceId string, annotations map[string]string) []dbapi.ConnectorNamespaceAnnotation {
	result := make([]dbapi.ConnectorNamespaceAnnotation, len(annotations))

	index := 0
	for key, val := range annotations {
		result[index].NamespaceId = namespaceId
		result[index].Key = key
		result[index].Value = val

		index++
	}

	return result
}
