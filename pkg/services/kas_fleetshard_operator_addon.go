package services

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
)

const (
	KasFleetshardOperatorRoleName = "kas_fleetshard_operator"
)

//go:generate moq -out kas_fleetshard_operator_addon_moq.go . KasFleetshardOperatorAddon
type KasFleetshardOperatorAddon interface {
	Provision(cluster api.Cluster) (bool, *errors.ServiceError)
}

func NewKasFleetshardOperatorAddon(ssoService KeycloakService, ocm ocm.Client) KasFleetshardOperatorAddon {
	return &agentOperatorAddon{
		ssoService: ssoService,
		ocm:        ocm,
	}
}

type agentOperatorAddon struct {
	ssoService KeycloakService
	ocm        ocm.Client
}

func (o *agentOperatorAddon) Provision(cluster api.Cluster) (bool, *errors.ServiceError) {
	acc, err := o.provisionServiceAccount(cluster.ClusterID)
	if err != nil {
		return false, errors.GeneralError("failed to create service account due to err: %v", err)
	}
	return acc != nil, nil
}

func (o *agentOperatorAddon) provisionServiceAccount(clusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	return o.ssoService.RegisterKasFleetshardOperatorServiceAccount(clusterId, KasFleetshardOperatorRoleName)
}
