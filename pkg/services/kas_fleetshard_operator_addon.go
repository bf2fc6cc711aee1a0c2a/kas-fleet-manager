package services

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/golang/glog"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const (
	KasFleetshardOperatorRoleName = "kas_fleetshard_operator"
	//TODO: update them to be the right parameter names once they are decided

	// parameter names for the kas-fleetshard-operator service account
	//TODO review later when kas-fleetshard-operator is accepting parameters
	//kasFleetshardOperatorParamMasSSOBaseUrl        = "mas-sso-base-url"
	//kasFleetshardOperatorParamMasSSORealm          = "mas-sso-realm"
	//kasFleetshardOperatorParamServiceAccountId     = "client-id"
	//kasFleetshardOperatorParamServiceAccountSecret = "client-secret"
	//// parameter names for the observability stack
	//kasFleetshardOperatorParamObservabilityRepo        = "observability-repo"
	//kasFleetshardOperatorParamObservabilityAccessToken = "observability-access-token"
	//kasFleetshardOperatorParamObservabilityChannel     = "observability-channel"
	//// parameter names for the cluster id
	//kasFleetshardOperatorParamClusterId = "cluster-id"
	//// parameter names for the control plane url
	//kasFleetshardOperatorParamControlPlaneBaseURL = "control-plane-base-url"
	//// parameter names for the strimzi operator version
	//kasFleetshardOperatorParamStrimziVersion = "strimzi-version"
)

//go:generate moq -out kas_fleetshard_operator_addon_moq.go . KasFleetshardOperatorAddon
type KasFleetshardOperatorAddon interface {
	Provision(cluster api.Cluster) (bool, *errors.ServiceError)
	ReconcileParameters(cluster api.Cluster) *errors.ServiceError
}

func NewKasFleetshardOperatorAddon(ssoService KeycloakService, ocm ocm.Client, configService ConfigService) KasFleetshardOperatorAddon {
	return &kasFleetshardOperatorAddon{
		ssoService:    ssoService,
		ocm:           ocm,
		configService: configService,
	}
}

type kasFleetshardOperatorAddon struct {
	ssoService    KeycloakService
	ocm           ocm.Client
	configService ConfigService
}

func (o *kasFleetshardOperatorAddon) Provision(cluster api.Cluster) (bool, *errors.ServiceError) {
	glog.V(5).Infof("Provision kas-fleetshard-operator for cluster %s", cluster.ClusterID)
	addonInstallation, addonErr := o.ocm.GetAddon(cluster.ClusterID, api.KasFleetshardOperatorAddonId)
	if addonErr != nil {
		return false, errors.GeneralError("failed to get existing addon status due to error: %v", addonErr)
	}
	if addonInstallation != nil && addonInstallation.ID() == "" {
		glog.V(5).Infof("No existing %s addon found, create a new one", api.KasFleetshardOperatorAddonId)
		acc, pErr := o.provisionServiceAccount(cluster.ClusterID)
		if pErr != nil {
			return false, errors.GeneralError("failed to create service account for cluster %s due to error: %v", cluster.ClusterID, pErr)
		}
		params := o.buildAddonParams(acc, cluster.ClusterID)
		addonInstallation, addonErr = o.ocm.CreateAddonWithParams(cluster.ClusterID, api.KasFleetshardOperatorAddonId, params)
		if addonErr != nil {
			return false, errors.GeneralError("failed to create addon for cluster %s due to error: %v", cluster.ClusterID, addonErr)
		}
	}

	if addonInstallation != nil && addonInstallation.State() == clustersmgmtv1.AddOnInstallationStateReady {
		return true, nil
	}

	return false, nil
}

func (o *kasFleetshardOperatorAddon) ReconcileParameters(cluster api.Cluster) *errors.ServiceError {
	glog.V(5).Infof("Reconcile parameters for kas-fleetshard operator on cluster %s", cluster.ClusterID)
	addonInstallation, addonErr := o.ocm.GetAddon(cluster.ClusterID, api.KasFleetshardOperatorAddonId)
	if addonErr != nil {
		return errors.GeneralError("failed to get existing addon status due to error: %v", addonErr)
	}
	if addonInstallation == nil || addonInstallation.ID() == "" {
		glog.Warningf("no valid installation for kas-fleetshard operator found on cluster %s", cluster.ClusterID)
		return errors.BadRequest("no valid kas-fleetshard addon for cluster %s", cluster.ClusterID)
	}
	glog.V(5).Infof("Found existing addon %s, updating parameters", addonInstallation.ID())
	acc, pErr := o.provisionServiceAccount(cluster.ClusterID)
	if pErr != nil {
		return errors.GeneralError("failed to create service account for cluster %s due to error: %v", cluster.ClusterID, pErr)
	}
	params := o.buildAddonParams(acc, cluster.ClusterID)
	addonInstallation, addonErr = o.ocm.UpdateAddonParameters(cluster.ClusterID, addonInstallation.ID(), params)
	if addonErr != nil {
		return errors.GeneralError("failed to update parameters for addon %s on cluster %s due to error: %v", addonInstallation.ID(), cluster.ClusterID, addonErr)
	}
	glog.V(5).Infof("Addon parameters for addon %s on cluster %s are updated", addonInstallation.ID(), cluster.ClusterID)
	return nil
}

func (o *kasFleetshardOperatorAddon) provisionServiceAccount(clusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Provisioning service account for cluster %s", clusterId)
	return o.ssoService.RegisterKasFleetshardOperatorServiceAccount(clusterId, KasFleetshardOperatorRoleName)
}

func (o *kasFleetshardOperatorAddon) buildAddonParams(serviceAccount *api.ServiceAccount, clusterId string) []ocm.AddonParameter {
	p := []ocm.AddonParameter{
		//TODO review later when kas-fleetshard-operator is accepting parameters
		//{
		//	Id:    kasFleetshardOperatorParamMasSSOBaseUrl,
		//	Value: o.configService.GetConfig().Keycloak.BaseURL,
		//},
		//{
		//	Id:    kasFleetshardOperatorParamMasSSORealm,
		//	Value: o.configService.GetConfig().Keycloak.KafkaRealm.Realm,
		//},
		//{
		//	Id:    kasFleetshardOperatorParamServiceAccountId,
		//	Value: serviceAccount.ClientID,
		//},
		//{
		//	Id:    kasFleetshardOperatorParamServiceAccountSecret,
		//	Value: serviceAccount.ClientSecret,
		//},
		//{
		//	Id:    kasFleetshardOperatorParamControlPlaneBaseURL,
		//	Value: o.configService.GetConfig().Server.PublicHostURL,
		//},
		//{
		//	Id:    kasFleetshardOperatorParamClusterId,
		//	Value: clusterId,
		//},
		//{
		//	Id:    kasFleetshardOperatorParamStrimziVersion,
		//	Value: o.configService.GetConfig().ClusterCreationConfig.StrimziOperatorVersion,
		//},
		//{
		//	Id:    kasFleetshardOperatorParamObservabilityRepo,
		//	Value: o.configService.GetObservabilityConfiguration().ObservabilityConfigRepo,
		//},
		//{
		//	Id:    kasFleetshardOperatorParamObservabilityChannel,
		//	Value: o.configService.GetObservabilityConfiguration().ObservabilityConfigChannel,
		//},
		//{
		//	Id:    kasFleetshardOperatorParamObservabilityAccessToken,
		//	Value: o.configService.GetObservabilityConfiguration().ObservabilityConfigAccessToken,
		//},
	}
	return p
}
