package services

import (
	"context"
	"reflect"
	"strconv"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/goava/di"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/metrics"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/golang/glog"
)

//go:generate moq -out data_plane_cluster_service_moq.go . DataPlaneClusterService
type DataPlaneClusterService interface {
	UpdateDataPlaneClusterStatus(ctx context.Context, clusterID string, status *dbapi.DataPlaneClusterStatus) *errors.ServiceError
	GetDataPlaneClusterConfig(ctx context.Context, clusterID string) (*dbapi.DataPlaneClusterConfig, *errors.ServiceError)
}

var _ DataPlaneClusterService = &dataPlaneClusterService{}

const dataPlaneClusterStatusCondReadyName = "Ready"

type dataPlaneClusterService struct {
	di.Inject
	ClusterService      ClusterService
	KafkaConfig         *config.KafkaConfig
	ObservabilityConfig *observatorium.ObservabilityConfiguration
}

func NewDataPlaneClusterService(config dataPlaneClusterService) *dataPlaneClusterService {
	return &config
}

func (d *dataPlaneClusterService) GetDataPlaneClusterConfig(ctx context.Context, clusterID string) (*dbapi.DataPlaneClusterConfig, *errors.ServiceError) {
	cluster, svcErr := d.ClusterService.FindClusterByID(clusterID)
	if svcErr != nil {
		return nil, svcErr
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return nil, errors.BadRequest("cluster agent with ID '%s' not found", clusterID)
	}

	return &dbapi.DataPlaneClusterConfig{
		Observability: dbapi.DataPlaneClusterConfigObservability{
			AccessToken: d.ObservabilityConfig.DataPlaneObservabilityConfig.GithubResourcesAuthToken,
			Channel:     d.ObservabilityConfig.ObservabilityConfigChannel,
			Repository:  d.ObservabilityConfig.ObservabilityConfigRepo,
			Tag:         d.ObservabilityConfig.ObservabilityConfigTag,
		},
		DynamicCapacityInfo: cluster.RetrieveDynamicCapacityInfo(),
		NetworkConfiguration: dbapi.DataPlaneClusterConfigNetwork{
			Private: cluster.AccessKafkasViaPrivateNetwork,
		},
	}, nil
}

func (d *dataPlaneClusterService) UpdateDataPlaneClusterStatus(ctx context.Context, clusterID string, status *dbapi.DataPlaneClusterStatus) *errors.ServiceError {
	cluster, svcErr := d.ClusterService.FindClusterByID(clusterID)
	if svcErr != nil {
		return svcErr
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return errors.BadRequest("cluster agent with ID '%s' not found", clusterID)
	}

	if !d.clusterCanProcessStatusReports(cluster) {
		glog.V(10).Infof("Cluster with ID '%s' is in '%s' state. Ignoring status report...", clusterID, cluster.Status)
		return nil
	}

	fleetShardOperatorReady, err := d.isFleetShardOperatorReady(status)
	if err != nil {
		return errors.ToServiceError(err)
	}
	if !fleetShardOperatorReady {
		if cluster.Status != api.ClusterWaitingForKasFleetShardOperator {
			err := d.ClusterService.UpdateStatus(*cluster, api.ClusterWaitingForKasFleetShardOperator)
			if err != nil {
				return errors.ToServiceError(err)
			}
			metrics.UpdateClusterStatusSinceCreatedMetric(*cluster, api.ClusterWaitingForKasFleetShardOperator)
		}
		glog.V(10).Infof("KAS Fleet Shard Operator not ready for Cluster ID '%s", clusterID)
		return nil
	}

	err = d.setClusterStatus(cluster, status)
	if err != nil {
		return errors.ToServiceError(err)
	}

	return nil
}

func (d *dataPlaneClusterService) setClusterStatus(cluster *api.Cluster, status *dbapi.DataPlaneClusterStatus) error {
	prevAvailableStrimziVersions, err := cluster.GetAvailableStrimziVersions()
	if err != nil {
		return err
	}

	dynamicCapacityInfo := cluster.RetrieveDynamicCapacityInfo()

	glog.Infof("Received capacity info for cluster ID '%s'. status.DynamicCapacityInfo: '%v'\n", cluster.ClusterID, status.DynamicCapacityInfo)
	glog.Infof("Current dynamic capacity info for cluster ID '%s'. DynamicCapacityInfo: '%v'\n", cluster.ClusterID, dynamicCapacityInfo)

	for instanceType, capacity := range dynamicCapacityInfo {
		newCapacity, ok := status.DynamicCapacityInfo[instanceType]
		if !ok {
			continue
		}

		dynamicCapacityInfo[instanceType] = api.DynamicCapacityInfo{
			MaxNodes:       capacity.MaxNodes,
			MaxUnits:       newCapacity.MaxUnits,
			RemainingUnits: newCapacity.RemainingUnits,
		}
	}

	glog.Infof("Updated dynamic capacity info for cluster ID '%s'. DynamicCapacityInfo: '%v'\n", cluster.ClusterID, dynamicCapacityInfo)

	err = cluster.SetDynamicCapacityInfo(dynamicCapacityInfo)

	if err != nil {
		return err
	}

	if len(status.AvailableStrimziVersions) > 0 && !reflect.DeepEqual(prevAvailableStrimziVersions, status.AvailableStrimziVersions) {
		err := cluster.SetAvailableStrimziVersions(status.AvailableStrimziVersions)
		if err != nil {
			return err
		}
		glog.Infof("Updating Strimzi operator available versions for cluster ID '%s'. Versions: '%v'\n", cluster.ClusterID, status.AvailableStrimziVersions)
	}

	if cluster.Status == api.ClusterWaitingForKasFleetShardOperator {
		metrics.UpdateClusterCreationDurationMetric(metrics.JobTypeClusterCreate, time.Since(cluster.CreatedAt))
		metrics.IncreaseClusterTotalOperationsCountMetric(constants.ClusterOperationCreate)
		metrics.IncreaseClusterSuccessOperationsCountMetric(constants.ClusterOperationCreate)
		metrics.UpdateClusterStatusSinceCreatedMetric(*cluster, api.ClusterReady)
	}

	cluster.Status = api.ClusterReady

	svcErr := d.ClusterService.Update(*cluster)

	if svcErr != nil {
		return svcErr
	}

	return nil
}

func (d *dataPlaneClusterService) isFleetShardOperatorReady(status *dbapi.DataPlaneClusterStatus) (bool, error) {
	for _, cond := range status.Conditions {
		if cond.Type == dataPlaneClusterStatusCondReadyName {
			condVal, err := strconv.ParseBool(cond.Status)
			if err != nil {
				return false, err
			}
			return condVal, nil
		}
	}
	return false, nil
}

func (d *dataPlaneClusterService) clusterCanProcessStatusReports(cluster *api.Cluster) bool {
	return cluster.Status == api.ClusterReady ||
		cluster.Status == api.ClusterFull ||
		cluster.Status == api.ClusterWaitingForKasFleetShardOperator
}
