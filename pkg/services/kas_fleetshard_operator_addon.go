package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/golang/glog"
)

const (
	KasFleetshardOperatorRoleName = "kas_fleetshard_operator"

	//parameter names for the kas-fleetshard-operator service account
	kasFleetshardOperatorParamMasSSOBaseUrl        = "sso-auth-server-url"
	kasFleetshardOperatorParamServiceAccountId     = "sso-client-id"
	kasFleetshardOperatorParamServiceAccountSecret = "sso-secret"
	// parameter names for the cluster id
	kasFleetshardOperatorParamClusterId = "cluster-id"
	// parameter names for the control plane url
	kasFleetshardOperatorParamControlPlaneBaseURL = "control-plane-url"
	//parameter names for fleetshardoperator synchronizer
	kasFleetshardOperatorParamPollinterval   = "poll-interval"
	kasFleetshardOperatorParamResyncInterval = "resync-interval"
)

//go:generate moq -out kas_fleetshard_operator_addon_moq.go . KasFleetshardOperatorAddon
type KasFleetshardOperatorAddon interface {
	Provision(cluster api.Cluster) (bool, *errors.ServiceError)
	ReconcileParameters(cluster api.Cluster) *errors.ServiceError
	RemoveServiceAccount(cluster api.Cluster) *errors.ServiceError
}

func NewKasFleetshardOperatorAddon(ssoService KeycloakService, configService ConfigService, providerFactory clusters.ProviderFactory) KasFleetshardOperatorAddon {
	return &kasFleetshardOperatorAddon{
		ssoService:      ssoService,
		configService:   configService,
		providerFactory: providerFactory,
	}
}

type kasFleetshardOperatorAddon struct {
	ssoService      KeycloakService
	providerFactory clusters.ProviderFactory
	configService   ConfigService
}

func (o *kasFleetshardOperatorAddon) Provision(cluster api.Cluster) (bool, *errors.ServiceError) {
	if cluster.ProviderType != api.ClusterProviderOCM {
		// TODO: in the future we can add implementations for other providers by applying the OLM resources for the kas fleetshard operator
		return false, errors.NotImplemented("addon installation is not implemented for provider type %s", cluster.ProviderType)
	}

	kasFleetshardAddonID := o.configService.GetConfig().OCM.KasFleetshardAddonID
	params, paramsErr := o.getAddonParams(cluster)
	if paramsErr != nil {
		return false, paramsErr
	}
	p, err := o.providerFactory.GetAddonProvider(cluster.ProviderType)
	if err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, "failed to get provider implementation")
	}
	glog.V(5).Infof("Provision addon %s for cluster %s", kasFleetshardAddonID, cluster.ClusterID)
	spec := &types.ClusterSpec{
		InternalID:     cluster.ClusterID,
		ExternalID:     cluster.ExternalID,
		Status:         cluster.Status,
		AdditionalInfo: cluster.ClusterSpec,
	}
	if ready, err := p.InstallAddonWithParams(spec, kasFleetshardAddonID, params); err != nil {
		return false, errors.NewWithCause(errors.ErrorGeneral, err, "failed to install addon %s for cluster %s", kasFleetshardAddonID, cluster.ClusterID)
	} else {
		return ready, nil
	}
}

func (o *kasFleetshardOperatorAddon) ReconcileParameters(cluster api.Cluster) *errors.ServiceError {
	if cluster.ProviderType != api.ClusterProviderOCM {
		// TODO: in the future we can add implementations for other providers by applying the OLM resources for the kas fleetshard operator
		return errors.NotImplemented("addon installation is not implemented for provider type %s", cluster.ProviderType)
	}

	kasFleetshardAddonID := o.configService.GetConfig().OCM.KasFleetshardAddonID
	params, paramsErr := o.getAddonParams(cluster)
	if paramsErr != nil {
		return paramsErr
	}
	p, err := o.providerFactory.GetAddonProvider(cluster.ProviderType)
	if err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to get provider implementation")
	}

	glog.V(5).Infof("Reconcile parameters for addon %s on cluster %s", kasFleetshardAddonID, cluster.ClusterID)
	spec := &types.ClusterSpec{
		InternalID:     cluster.ClusterID,
		ExternalID:     cluster.ExternalID,
		Status:         cluster.Status,
		AdditionalInfo: cluster.ClusterSpec,
	}
	if err := p.UpdateAddonWithParams(spec, kasFleetshardAddonID, params); err != nil {
		return errors.NewWithCause(errors.ErrorGeneral, err, "failed to update parameters for addon %s for cluster %s", kasFleetshardAddonID, cluster.ClusterID)
	} else {
		glog.V(5).Infof("Addon parameters for addon %s on cluster %s are updated", kasFleetshardAddonID, cluster.ClusterID)
		return nil
	}
}

func (o *kasFleetshardOperatorAddon) getAddonParams(cluster api.Cluster) ([]ocm.AddonParameter, *errors.ServiceError) {
	acc, pErr := o.provisionServiceAccount(cluster.ClusterID)
	if pErr != nil {
		return nil, errors.GeneralError("failed to create service account for cluster %s due to error: %v", cluster.ClusterID, pErr)
	}
	params := o.buildAddonParams(acc, cluster.ClusterID)
	return params, nil
}

func (o *kasFleetshardOperatorAddon) provisionServiceAccount(clusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Provisioning service account for cluster %s", clusterId)
	return o.ssoService.RegisterKasFleetshardOperatorServiceAccount(clusterId, KasFleetshardOperatorRoleName)
}

func (o *kasFleetshardOperatorAddon) buildAddonParams(serviceAccount *api.ServiceAccount, clusterId string) []ocm.AddonParameter {
	p := []ocm.AddonParameter{

		{
			Id:    kasFleetshardOperatorParamMasSSOBaseUrl,
			Value: o.configService.GetConfig().Keycloak.KafkaRealm.ValidIssuerURI,
		},
		{
			Id:    kasFleetshardOperatorParamServiceAccountId,
			Value: serviceAccount.ClientID,
		},
		{
			Id:    kasFleetshardOperatorParamServiceAccountSecret,
			Value: serviceAccount.ClientSecret,
		},
		{
			Id:    kasFleetshardOperatorParamControlPlaneBaseURL,
			Value: o.configService.GetConfig().Server.PublicHostURL,
		},
		{
			Id:    kasFleetshardOperatorParamClusterId,
			Value: clusterId,
		},
		{
			Id:    kasFleetshardOperatorParamPollinterval,
			Value: o.configService.GetConfig().KasFleetShardConfig.PollInterval,
		},
		{
			Id:    kasFleetshardOperatorParamResyncInterval,
			Value: o.configService.GetConfig().KasFleetShardConfig.ResyncInterval,
		},
	}
	return p
}

func (o *kasFleetshardOperatorAddon) RemoveServiceAccount(cluster api.Cluster) *errors.ServiceError {
	glog.V(5).Infof("Removing kas-fleetshard-operator service account for cluster %s", cluster.ClusterID)
	return o.ssoService.DeRegisterKasFleetshardOperatorServiceAccount(cluster.ClusterID)
}
