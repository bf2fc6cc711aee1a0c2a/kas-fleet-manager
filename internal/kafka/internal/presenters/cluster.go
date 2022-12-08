package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func PresentEnterpriseCluster(cluster api.Cluster, fleetShardParams services.ParameterList) (public.EnterpriseCluster, *errors.ServiceError) {

	fsoParams := []public.FleetshardParameter{}

	for _, param := range fleetShardParams {
		fsoParams = append(fsoParams, public.FleetshardParameter{
			Id:    param.Id,
			Value: param.Value,
		})
	}

	c := public.EnterpriseCluster{
		ClusterId:            cluster.ClusterID,
		Status:               cluster.Status.String(),
		FleetshardParameters: fsoParams,
	}

	return c, nil
}
