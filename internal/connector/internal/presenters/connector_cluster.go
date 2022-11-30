package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
)

func ConvertClusterAnnotations(id string, annotations map[string]string) []dbapi.ConnectorClusterAnnotation {
	res := make([]dbapi.ConnectorClusterAnnotation, len(annotations))
	i := 0
	for k, v := range annotations {
		res[i].ConnectorClusterID = id
		res[i].Key = k
		res[i].Value = v

		i++
	}

	return res
}

func PresentConnectorCluster(from *dbapi.ConnectorCluster) public.ConnectorCluster {
	reference := PresentReference(from.ID, from)
	return public.ConnectorCluster{
		Id:          reference.Id,
		Kind:        reference.Kind,
		Href:        reference.Href,
		Owner:       from.Owner,
		Name:        from.Name,
		CreatedAt:   from.CreatedAt,
		ModifiedAt:  from.UpdatedAt,
		Annotations: PresentClusterAnnotations(from.Annotations),
		Status: public.ConnectorClusterStatusStatus{
			State: public.ConnectorClusterState(from.Status.Phase),
		},
	}
}

func PresentClusterAnnotations(annotations []dbapi.ConnectorClusterAnnotation) map[string]string {
	res := make(map[string]string, len(annotations))
	for _, ann := range annotations {
		res[ann.Key] = ann.Value
	}
	return res
}

func PresentPrivateConnectorCluster(from *dbapi.ConnectorCluster) private.ConnectorClusterAdminView {
	reference := PresentReference(from.ID, from)
	return private.ConnectorClusterAdminView{
		Id:          reference.Id,
		Kind:        reference.Kind,
		Href:        reference.Href,
		Owner:       from.Owner,
		Name:        from.Name,
		CreatedAt:   from.CreatedAt,
		ModifiedAt:  from.UpdatedAt,
		Annotations: PresentClusterAnnotations(from.Annotations),
		Status:      PresentConnectorClusterAdminStatus(from.Status),
	}
}
