package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func PresentEnterpriseClusterRegistrationResponse(cluster api.Cluster, fleetShardParams services.ParameterList) (public.EnterpriseClusterRegistrationResponse, *errors.ServiceError) {

	fsoParams := []public.FleetshardParameter{}

	for _, param := range fleetShardParams {
		fsoParams = append(fsoParams, public.FleetshardParameter{
			Id:    param.Id,
			Value: param.Value,
		})
	}

	reference := PresentReference(cluster.ClusterID, cluster)

	c := public.EnterpriseClusterRegistrationResponse{
		Id:                   cluster.ClusterID,
		ClusterId:            cluster.ClusterID,
		Status:               cluster.Status.String(),
		Kind:                 reference.Kind,
		Href:                 reference.Href,
		FleetshardParameters: fsoParams,
	}

	return c, nil
}

func PresentEnterpriseCluster(cluster api.Cluster) public.EnterpriseCluster {
	reference := PresentReference(cluster.ClusterID, cluster)
	return public.EnterpriseCluster{
		Id:        cluster.ClusterID,
		Status:    cluster.Status.String(),
		ClusterId: cluster.ClusterID,
		Kind:      reference.Kind,
		Href:      reference.Href,
	}
}
