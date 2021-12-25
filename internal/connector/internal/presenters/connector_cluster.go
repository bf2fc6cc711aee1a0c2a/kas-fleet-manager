package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/admin/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
)

func ConvertConnectorCluster(from public.ConnectorCluster) dbapi.ConnectorCluster {
	return dbapi.ConnectorCluster{
		Name:  from.Name,
	}
}

//func PresentConnectorCluster(from dbapi.ConnectorCluster) public.ConnectorCluster {
//	//reference := PresentReference(from.ID, from)
//	return public.ConnectorCluster{
//		//Id:   reference.Id,
//		//Kind: reference.Kind,
//		//Href: reference.Href,
//		//Owner:     from.Owner,
//		Name:      from.Name,
//		//CreatedAt: from.CreatedAt,
//		//ModifiedAt: from.UpdatedAt,
//		//Status: public.ConnectorClusterInstanceStatusStatus{
//		//	State: public.ConnectorClusterState(from.Status.Phase),
//		//},
//	}
//}

func PresentConnectorClusterAdmin(from dbapi.ConnectorCluster) private.ConnectorCluster {
	reference := PresentReference(from.ID, from)
	return private.ConnectorCluster{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Metadata: private.ConnectorClusterAllOfMetadata{
			Owner:     from.Owner,
			Name:      from.Name,
			CreatedAt: from.CreatedAt,
			UpdatedAt: from.UpdatedAt,
		},
		Status: from.Status.Phase,
	}
}
