package services

import (
	"github.com/golang/glog"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
)

const (
	KasFleetshardOperatorRoleName = "kas_fleetshard_operator"
	//TODO: update them to be the right parameter names once they are decided

	// parameter names for the kas-fleetshard-operator service account
	kasFleetshardOperatorParamMasSSOBaseUrl        = "mas-sso-base-url"
	kasFleetshardOperatorParamMasSSORealm          = "mas-sso-realm"
	kasFleetshardOperatorParamServiceAccountId     = "client-id"
	kasFleetshardOperatorParamServiceAccountSecret = "client-secret"
	// parameter names for the observability stack
	kasFleetshardOperatorParamObservabilityRepo        = "observability-repo"
	kasFleetshardOperatorParamObservabilityAccessToken = "observability-access-token"
	kasFleetshardOperatorParamObservabilityChannel     = "observability-channel"
	// parameter names for the cluster id
	kasFleetshardOperatorParamClusterId = "cluster-id"
	// parameter names for the control plane url
	kasFleetshardOperatorParamControlPlaneBaseURL = "control-plane-base-url"
	// parameter names for the strimzi operator version
	kasFleetshardOperatorParamStrimziVersion = "strimzi-version"
)

//go:generate moq -out kas_fleetshard_operator_addon_moq.go . KasFleetshardOperatorAddon
type KasFleetshardOperatorAddon interface {
	Provision(cluster api.Cluster) (bool, *errors.ServiceError)
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
	glog.V(5).Infof("Provision kas-fleetshard-operator for cluster %s", cluster.ID)
	addonInstallation, addonErr := o.ocm.GetAddon(cluster.ID, api.KasFleetshardOperatorAddonId)
	if addonErr != nil {
		return false, errors.GeneralError("failed to get existing addon status due to error: %v", addonErr)
	}
	if addonInstallation != nil && addonInstallation.ID() == "" {
		glog.V(5).Infof("No existing %s addon found, create a new one", api.KasFleetshardOperatorAddonId)
		acc, pErr := o.provisionServiceAccount(cluster.ClusterID)
		if pErr != nil {
			return false, errors.GeneralError("failed to create service account for cluster %s due to error: %v", cluster.ID, pErr)
		}
		params := o.buildAddonParams(acc, cluster.ID)
		addonInstallation, addonErr = o.ocm.CreateAddonWithParams(cluster.ID, api.KasFleetshardOperatorAddonId, params)
		if addonErr != nil {
			return false, errors.GeneralError("failed to create addon for cluster %s due to error: %v", cluster.ID, addonErr)
		}
	}

	if addonInstallation != nil && addonInstallation.State() == clustersmgmtv1.AddOnInstallationStateReady {
		return true, nil
	}

	return false, nil
}

func (o *kasFleetshardOperatorAddon) provisionServiceAccount(clusterId string) (*api.ServiceAccount, *errors.ServiceError) {
	glog.V(5).Infof("Provisioning service account for cluster %s", clusterId)
	return o.ssoService.RegisterKasFleetshardOperatorServiceAccount(clusterId, KasFleetshardOperatorRoleName)
}

func (o *kasFleetshardOperatorAddon) buildAddonParams(serviceAccount *api.ServiceAccount, clusterId string) []ocm.AddonParameter {
	p := []ocm.AddonParameter{
		{
			Id:    kasFleetshardOperatorParamMasSSOBaseUrl,
			Value: o.configService.GetConfig().Keycloak.BaseURL,
		},
		{
			Id:    kasFleetshardOperatorParamMasSSORealm,
			Value: o.configService.GetConfig().Keycloak.Realm,
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
			Id:    kasFleetshardOperatorParamStrimziVersion,
			Value: o.configService.GetConfig().ClusterCreationConfig.StrimziOperatorVersion,
		},
		{
			Id:    kasFleetshardOperatorParamObservabilityRepo,
			Value: o.configService.GetObservabilityConfiguration().ObservabilityConfigRepo,
		},
		{
			Id:    kasFleetshardOperatorParamObservabilityChannel,
			Value: o.configService.GetObservabilityConfiguration().ObservabilityConfigChannel,
		},
		{
			Id:    kasFleetshardOperatorParamObservabilityAccessToken,
			Value: o.configService.GetObservabilityConfiguration().ObservabilityConfigAccessToken,
		},
	}
	return p
}
