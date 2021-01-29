package services

import (
	"context"
	"strconv"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
)

type DataPlaneClusterService interface {
	UpdateDataPlaneClusterStatus(ctx context.Context, clusterID string, status *api.DataPlaneClusterStatus) *errors.ServiceError
}

var _ DataPlaneClusterService = &dataPlaneClusterService{}

const dataPlaneClusterStatusCondReadyName = "Ready"

type dataPlaneClusterService struct {
	ocmClient      ocm.Client
	clusterService ClusterService
}

func NewDataPlaneClusterService(clusterService ClusterService, ocmClient ocm.Client) *dataPlaneClusterService {
	return &dataPlaneClusterService{
		ocmClient:      ocmClient,
		clusterService: clusterService,
	}
}

func (d *dataPlaneClusterService) UpdateDataPlaneClusterStatus(ctx context.Context, clusterID string, status *api.DataPlaneClusterStatus) *errors.ServiceError {
	err := d.setClusterStatus(clusterID, status)
	if err != nil {
		return errors.ToServiceError(err)
	}
	return nil
}

func (d *dataPlaneClusterService) setClusterStatus(agentClusterID string, status *api.DataPlaneClusterStatus) error {
	cluster, svcErr := d.clusterService.FindClusterByID(agentClusterID)
	if svcErr != nil {
		return svcErr
	}
	if cluster == nil {
		// 404 is used for authenticated requests. So to distinguish the errors, we use 400 here
		return errors.BadRequest("Cluster agent with ID '%s' not found", agentClusterID)
	}

	isReady, err := d.isClusterReady(status)
	if err != nil {
		return errors.ToServiceError(err)
	}

	if isReady && cluster.Status != api.ClusterReady && cluster.Status == api.AddonInstalled {
		err := d.clusterService.UpdateStatus(*cluster, api.ClusterReady)
		return errors.ToServiceError(err)
	}

	return nil
}

func (d *dataPlaneClusterService) isClusterReady(status *api.DataPlaneClusterStatus) (bool, error) {
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
