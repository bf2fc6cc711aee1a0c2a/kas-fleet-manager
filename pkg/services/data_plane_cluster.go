package services

import (
	"context"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
)

type DataPlaneClusterService interface {
	UpdateDataPlaneClusterStatus(ctx context.Context, clusterID string, status *api.DataPlaneClusterStatus) *errors.ServiceError
}

var _ DataPlaneClusterService = &dataPlaneClusterService{}

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
	return nil
}
