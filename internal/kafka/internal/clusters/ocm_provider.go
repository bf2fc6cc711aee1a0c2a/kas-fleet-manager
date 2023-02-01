package clusters

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	svcErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/golang/glog"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	ipdAlreadyCreatedErrorToCheck = "already exists"

	// In OCM, number of compute nodes in a Multi-AZ cluster must be a multiple of
	// this number
	ocmMultiAZClusterNodeScalingMultiple = 3
)

var (
	// AMS quota filters
	AMSQuotaAddonResourceNamePrefix        = "addon"
	AMSQuotaOSDProductName          string = "OSD"
	AMSQuotaComputeNodeResourceType string = "compute.node"
	AMSQuotaClusterResourceType     string = "cluster"
)

type OCMProvider struct {
	ocmClient      ocm.Client
	clusterBuilder ClusterBuilder
	ocmConfig      *ocm.OCMConfig
}

// blank assignment to verify that OCMProvider implements Provider
var _ Provider = &OCMProvider{}

func (o *OCMProvider) Create(request *types.ClusterRequest) (*types.ClusterSpec, error) {
	// Build a new OSD cluster object
	newCluster, err := o.clusterBuilder.NewOCMClusterFromCluster(request)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build OCM cluster")
	}

	// Send POST request to /api/clusters_mgmt/v1/clusters to create a new OSD cluster
	createdCluster, err := o.ocmClient.CreateCluster(newCluster)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to create OCM cluster")
	}

	result := &types.ClusterSpec{
		Status: api.ClusterProvisioning,
	}
	if createdCluster.ID() != "" {
		result.InternalID = createdCluster.ID()
	}
	if createdCluster.ExternalID() != "" {
		result.ExternalID = createdCluster.ExternalID()
	}
	return result, nil
}

func (o *OCMProvider) GetCluster(clusterID string) (types.ClusterSpec, error) {
	cluster, err := o.ocmClient.GetCluster(clusterID)
	if err != nil {
		return types.ClusterSpec{}, errors.Wrapf(err, "failed to get cluster %s", clusterID)
	}
	return types.ClusterSpec{
		MultiAZ:       cluster.MultiAZ(),
		CloudProvider: cluster.CloudProvider().ID(),
		Region:        cluster.Region().ID(),
		ExternalID:    cluster.ExternalID(),
	}, nil
}

func (o *OCMProvider) CheckClusterStatus(spec *types.ClusterSpec) (*types.ClusterSpec, error) {
	ocmCluster, err := o.ocmClient.GetCluster(spec.InternalID)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cluster %s", spec.InternalID)
	}
	clusterStatus := ocmCluster.Status()
	if spec.Status == "" {
		spec.Status = api.ClusterProvisioning
	}

	spec.StatusDetails = clusterStatus.ProvisionErrorMessage()

	if clusterStatus.State() == clustersmgmtv1.ClusterStateReady {
		if spec.ExternalID == "" {
			externalId, ok := ocmCluster.GetExternalID()
			if !ok {
				return nil, errors.Errorf("external ID for cluster %s cannot be found", ocmCluster.ID())
			}
			spec.ExternalID = externalId
		}
		spec.Status = api.ClusterProvisioned
	}
	if clusterStatus.State() == clustersmgmtv1.ClusterStateError {
		spec.Status = api.ClusterFailed
	}
	return spec, nil
}

func (o *OCMProvider) Delete(spec *types.ClusterSpec) (bool, error) {
	code, err := o.ocmClient.DeleteCluster(spec.InternalID)
	if err != nil && code != http.StatusNotFound {
		return false, errors.Wrapf(err, "failed to delete cluster %s", spec.InternalID)
	}
	return code == http.StatusNotFound, nil
}

func (o *OCMProvider) GetClusterDNS(clusterSpec *types.ClusterSpec) (string, error) {
	clusterDNS, err := o.ocmClient.GetClusterDNS(clusterSpec.InternalID)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get dns for cluster %s", clusterSpec.InternalID)
	}
	return clusterDNS, nil
}

func (o *OCMProvider) AddIdentityProvider(clusterSpec *types.ClusterSpec, identityProviderInfo types.IdentityProviderInfo) (*types.IdentityProviderInfo, error) {
	if identityProviderInfo.OpenID != nil {
		idpId, err := o.addOpenIDIdentityProvider(clusterSpec, *identityProviderInfo.OpenID)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to add identity provider for cluster %s", clusterSpec.InternalID)
		}
		identityProviderInfo.OpenID.ID = idpId
		return &identityProviderInfo, nil
	}
	return nil, nil
}

// RemoveResources deletes the SyncSet from a cluster. If the SyncSet (or the cluster) is not found, no error is returned
func (o *OCMProvider) RemoveResources(clusterSpec *types.ClusterSpec, syncSetName string) error {
	status, err := o.ocmClient.DeleteSyncSet(clusterSpec.InternalID, syncSetName)
	if err != nil {
		// the SyncSet may already have been deleted in a previous run and the cluster is
		// waiting for the resources to be terminated
		if status == http.StatusNotFound {
			return nil
		}
		return err
	}

	if status != http.StatusNoContent {
		return errors.New(fmt.Sprintf("unexpected response code when deleting sync set: %v", status))
	}

	return nil
}

func (o *OCMProvider) ApplyResources(clusterSpec *types.ClusterSpec, resources types.ResourceSet) (*types.ResourceSet, error) {
	existingSyncset, err := o.ocmClient.GetSyncSet(clusterSpec.InternalID, resources.Name)
	syncSetFound := true
	if err != nil {
		svcErr := svcErrors.ToServiceError(err)
		if !svcErr.Is404() {
			return nil, err
		}
		syncSetFound = false
	}

	if !syncSetFound {
		glog.V(10).Infof("SyncSet for cluster %s not found. Creating it...", clusterSpec.InternalID)
		_, syncsetErr := o.createSyncSet(clusterSpec.InternalID, resources)
		if syncsetErr != nil {
			return nil, errors.Wrapf(syncsetErr, "failed to create syncset for cluster %s", clusterSpec.InternalID)
		}
	} else {
		glog.V(10).Infof("SyncSet for cluster %s already created", clusterSpec.InternalID)
		_, syncsetErr := o.updateSyncSet(clusterSpec.InternalID, resources, existingSyncset)
		if syncsetErr != nil {
			return nil, errors.Wrapf(syncsetErr, "failed to update syncset for cluster %s", clusterSpec.InternalID)
		}
	}

	return &resources, nil
}

func (o *OCMProvider) InstallStrimzi(clusterSpec *types.ClusterSpec) (bool, error) {
	return o.installAddon(clusterSpec, o.ocmConfig.StrimziOperatorAddonID)
}

func (o *OCMProvider) InstallClusterLogging(clusterSpec *types.ClusterSpec, params []types.Parameter) (bool, error) {
	return o.installAddonWithParams(clusterSpec, o.ocmConfig.ClusterLoggingOperatorAddonID, params)
}

func (o *OCMProvider) InstallKasFleetshard(clusterSpec *types.ClusterSpec, params []types.Parameter) (bool, error) {
	return o.installAddonWithParams(clusterSpec, o.ocmConfig.KasFleetshardAddonID, params)
}

func (o *OCMProvider) installAddon(clusterSpec *types.ClusterSpec, addonID string) (bool, error) {
	clusterId := clusterSpec.InternalID
	addonInstallation, err := o.ocmClient.GetAddon(clusterId, addonID)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get addon %s for cluster %s", addonID, clusterSpec.InternalID)
	}

	// Addon needs to be installed if addonInstallation doesn't exist
	if addonInstallation.ID() == "" {
		addonInstallation, err = o.ocmClient.CreateAddon(clusterId, addonID)
		if err != nil {
			return false, errors.Wrapf(err, "failed to create addon %s for cluster %s", addonID, clusterSpec.InternalID)
		}
	}

	// The cluster is ready when the state reports ready
	if addonInstallation.State() == clustersmgmtv1.AddOnInstallationStateReady {
		return true, nil
	}

	return false, nil
}

func (o *OCMProvider) installAddonWithParams(clusterSpec *types.ClusterSpec, addonId string, params []types.Parameter) (bool, error) {
	addonInstallation, addonErr := o.ocmClient.GetAddon(clusterSpec.InternalID, addonId)
	if addonErr != nil {
		return false, errors.Wrapf(addonErr, "failed to get addon %s for cluster %s", addonId, clusterSpec.InternalID)
	}

	if addonInstallation != nil && addonInstallation.ID() == "" {
		glog.V(5).Infof("No existing %s addon found, create a new one", addonId)
		addonInstallation, addonErr = o.ocmClient.CreateAddonWithParams(clusterSpec.InternalID, addonId, params)
		if addonErr != nil {
			return false, errors.Wrapf(addonErr, "failed to create addon %s for cluster %s", addonId, clusterSpec.InternalID)
		}
	}

	if addonInstallation != nil && addonInstallation.State() == clustersmgmtv1.AddOnInstallationStateReady {
		addonInstallation, addonErr = o.ocmClient.UpdateAddonParameters(clusterSpec.InternalID, addonInstallation.ID(), params)
		if addonErr != nil {
			return false, errors.Wrapf(addonErr, "failed to update parameters for addon %s on cluster %s", addonInstallation.ID(), clusterSpec.InternalID)
		}
		return true, nil
	}

	return false, nil
}

func (o *OCMProvider) GetCloudProviders() (*types.CloudProviderInfoList, error) {
	list := types.CloudProviderInfoList{}
	providerList, err := o.ocmClient.GetCloudProviders()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get cloud providers from OCM")
	}
	var items []types.CloudProviderInfo
	providerList.Each(func(item *clustersmgmtv1.CloudProvider) bool {
		p := types.CloudProviderInfo{
			ID:          item.ID(),
			Name:        item.Name(),
			DisplayName: item.DisplayName(),
		}
		items = append(items, p)
		return true
	})
	list.Items = items
	return &list, nil
}

func (o *OCMProvider) GetCloudProviderRegions(providerInfo types.CloudProviderInfo) (*types.CloudProviderRegionInfoList, error) {
	list := types.CloudProviderRegionInfoList{}
	cp, err := clustersmgmtv1.NewCloudProvider().ID(providerInfo.ID).Name(providerInfo.Name).DisplayName(providerInfo.DisplayName).Build()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to build cloud provider object")
	}
	regionsList, err := o.ocmClient.GetRegions(cp)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get regions for provider %s", providerInfo.Name)
	}
	var items []types.CloudProviderRegionInfo
	regionsList.Each(func(item *clustersmgmtv1.CloudRegion) bool {
		r := types.CloudProviderRegionInfo{
			ID:              item.ID(),
			CloudProviderID: item.CloudProvider().ID(),
			Name:            item.Name(),
			DisplayName:     item.DisplayName(),
			SupportsMultiAZ: item.SupportsMultiAZ(),
		}
		items = append(items, r)
		return true
	})
	list.Items = items
	return &list, nil
}

// ensure OCMProvider implements Provider interface
var _ Provider = &OCMProvider{}

func newOCMProvider(ocmClient ocm.ClusterManagementClient, clusterBuilder ClusterBuilder, ocmConfig *ocm.OCMConfig) *OCMProvider {
	return &OCMProvider{
		ocmClient:      ocmClient,
		clusterBuilder: clusterBuilder,
		ocmConfig:      ocmConfig,
	}
}

func (o *OCMProvider) addOpenIDIdentityProvider(clusterSpec *types.ClusterSpec, openIdIdpInfo types.OpenIDIdentityProviderInfo) (string, error) {
	provider, buildErr := buildIdentityProvider(openIdIdpInfo)
	if buildErr != nil {
		return "", errors.WithStack(buildErr)
	}
	createdIdentityProvider, createIdentityProviderErr := o.ocmClient.CreateIdentityProvider(clusterSpec.InternalID, provider)
	if createIdentityProviderErr != nil {
		// check to see if identity provider with name 'Kafka_SRE' already exists, if so use it.
		if strings.Contains(createIdentityProviderErr.Error(), ipdAlreadyCreatedErrorToCheck) {
			identityProvidersList, identityProviderListErr := o.ocmClient.GetIdentityProviderList(clusterSpec.InternalID)
			if identityProviderListErr != nil {
				return "", errors.WithStack(identityProviderListErr)
			}

			for _, identityProvider := range identityProvidersList.Slice() {
				if identityProvider.Name() == openIdIdpInfo.Name {
					return identityProvider.ID(), nil
				}
			}
		}
		return "", errors.WithStack(createIdentityProviderErr)
	}
	return createdIdentityProvider.ID(), nil
}

func (o *OCMProvider) createSyncSet(clusterID string, resourceSet types.ResourceSet) (*clustersmgmtv1.Syncset, error) {
	syncset, sysnsetBuilderErr := clustersmgmtv1.NewSyncset().ID(resourceSet.Name).Resources(resourceSet.Resources...).Build()

	if sysnsetBuilderErr != nil {
		return nil, errors.WithStack(sysnsetBuilderErr)
	}

	return o.ocmClient.CreateSyncSet(clusterID, syncset)
}

func (o *OCMProvider) updateSyncSet(clusterID string, resourceSet types.ResourceSet, existingSyncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {
	syncset, sysnsetBuilderErr := clustersmgmtv1.NewSyncset().Resources(resourceSet.Resources...).Build()
	if sysnsetBuilderErr != nil {
		return nil, errors.WithStack(sysnsetBuilderErr)
	}
	if syncsetResourcesChanged(existingSyncset, syncset) {
		glog.V(5).Infof("SyncSet for cluster %s is changed, will update", clusterID)
		return o.ocmClient.UpdateSyncSet(clusterID, resourceSet.Name, syncset)
	}
	glog.V(10).Infof("SyncSet for cluster %s is not changed, no update needed", clusterID)
	return syncset, nil
}

func syncsetResourcesChanged(existing *clustersmgmtv1.Syncset, new *clustersmgmtv1.Syncset) bool {
	if len(existing.Resources()) != len(new.Resources()) {
		return true
	}
	// Here we will convert values in the Resources slice to the same type, and then compare the values.
	// This is needed because when you use ocm.GetSyncset(), the Resources in the returned object only contains a slice of map[string]interface{} objects, because it can't convert them to concrete typed objects.
	// So the compare if there changes, we need to make sure they are the same type first.
	// If the type conversion doesn't work, or the converted values doesn't match, then they are not equal.
	// This assumes that the order of objects in the Resources slice are the same in the exiting and new Syncset (which is the case as the OCM API returns the syncset resources in the same order as they posted)
	for i, r := range new.Resources() {
		obj := reflect.New(reflect.TypeOf(r).Elem()).Interface()
		// Here we convert the unstructured type to the concrete type, as there is a bug in OperatorGroup type to convert it to the unstructured type
		err := runtime.DefaultUnstructuredConverter.FromUnstructured(existing.Resources()[i].(map[string]interface{}), obj)
		// if we can't do the type conversion, it likely means the resource has changed
		if err != nil {
			return true
		}
		if !reflect.DeepEqual(obj, r) {
			return true
		}
	}

	return false
}

func buildIdentityProvider(idpInfo types.OpenIDIdentityProviderInfo) (*clustersmgmtv1.IdentityProvider, error) {
	openIdentityBuilder := clustersmgmtv1.NewOpenIDIdentityProvider().
		ClientID(idpInfo.ClientID).
		ClientSecret(idpInfo.ClientSecret).
		Claims(clustersmgmtv1.NewOpenIDClaims().
			Email("email").
			PreferredUsername("preferred_username").
			Name("last_name", "preferred_username")).
		Issuer(idpInfo.Issuer)

	identityProviderBuilder := clustersmgmtv1.NewIdentityProvider().
		Type("OpenIDIdentityProvider").
		MappingMethod(clustersmgmtv1.IdentityProviderMappingMethodClaim).
		OpenID(openIdentityBuilder).
		Name(idpInfo.Name)

	identityProvider, idpBuildErr := identityProviderBuilder.Build()
	if idpBuildErr != nil {
		return nil, idpBuildErr
	}

	return identityProvider, nil
}

func (o *OCMProvider) GetMachinePool(clusterID string, id string) (*types.MachinePoolInfo, error) {
	ocmMachinePool, err := o.ocmClient.GetMachinePool(clusterID, id)
	if err != nil {
		return nil, err
	}

	if ocmMachinePool == nil {
		return nil, nil
	}

	var nodeTaints []types.ClusterNodeTaint
	for _, ocmNodeTaint := range ocmMachinePool.Taints() {
		nodeTaint := types.ClusterNodeTaint{
			Effect: ocmNodeTaint.Effect(),
			Key:    ocmNodeTaint.Key(),
			Value:  ocmNodeTaint.Value(),
		}
		nodeTaints = append(nodeTaints, nodeTaint)
	}
	res := &types.MachinePoolInfo{
		ID:                 ocmMachinePool.ID(),
		InstanceSize:       ocmMachinePool.InstanceType(),
		MultiAZ:            len(ocmMachinePool.AvailabilityZones()) > 1,
		AutoScalingEnabled: !ocmMachinePool.Autoscaling().Empty(),
		AutoScaling: types.MachinePoolAutoScaling{
			MinNodes: ocmMachinePool.Autoscaling().MinReplicas(),
			MaxNodes: ocmMachinePool.Autoscaling().MaxReplicas(),
		},
		NodeLabels: ocmMachinePool.Labels(),
		NodeTaints: nodeTaints,
	}

	return res, nil
}

func (o *OCMProvider) CreateMachinePool(request *types.MachinePoolRequest) (*types.MachinePoolRequest, error) {
	machinePoolBuilder := clustersmgmtv1.NewMachinePool()
	// At the moment of writing this (2022-06-20) OCM's maximum length for
	// Machine Pool IDs is 30 characters
	machinePoolBuilder.ID(request.ID)
	machinePoolBuilder.InstanceType(request.InstanceSize)

	if request.AutoScalingEnabled {
		autoScalingMaxNodes := request.AutoScaling.MaxNodes
		autoScalingMinNodes := request.AutoScaling.MinNodes

		if request.MultiAZ {
			autoScalingMinNodes = shared.RoundUp(request.AutoScaling.MinNodes, ocmMultiAZClusterNodeScalingMultiple)
			autoScalingMaxNodes = shared.RoundUp(request.AutoScaling.MaxNodes, ocmMultiAZClusterNodeScalingMultiple)
		}
		if autoScalingMinNodes > autoScalingMaxNodes {
			return nil, fmt.Errorf("error creating MachinePool '%s' for cluster id '%s': minimum number of nodes cannot be more than maximum number of nodes", request.ID, request.ClusterID)
		}
		autoScalingBuilder := clustersmgmtv1.NewMachinePoolAutoscaling()
		autoScalingBuilder.MinReplicas(autoScalingMinNodes)
		autoScalingBuilder.MaxReplicas(autoScalingMaxNodes)
		autoScalingBuilder.ID(request.ID)
		machinePoolBuilder.Autoscaling(autoScalingBuilder)

	}
	machinePoolBuilder.Labels(request.NodeLabels)
	var taintsBuilders []*clustersmgmtv1.TaintBuilder
	for _, nodeTaint := range request.NodeTaints {
		taintBuilder := clustersmgmtv1.NewTaint()
		taintBuilder.Key(nodeTaint.Key)
		taintBuilder.Value(nodeTaint.Value)
		taintBuilder.Effect(nodeTaint.Effect)
		taintsBuilders = append(taintsBuilders, taintBuilder)
	}
	machinePoolBuilder.Taints(taintsBuilders...)
	machinePool, err := machinePoolBuilder.Build()
	if err != nil {
		return nil, err
	}

	_, err = o.ocmClient.CreateMachinePool(request.ClusterID, machinePool)
	if err != nil {
		return nil, err
	}

	return request, err
}

// GetClusterResourceQuotaCosts returns a list of quota cost information related to ocm resources used for the provisioning and
// terraforming of data plane clusters for the authenticated user.
//
// Returns a nil slice when no ocm resource quota is assigned to the user
func (o *OCMProvider) GetClusterResourceQuotaCosts() ([]types.QuotaCost, error) {
	var quotaCostList []types.QuotaCost

	account, err := o.ocmClient.GetCurrentAccount()
	if err != nil {
		return quotaCostList, err
	}
	orgID, ok := account.Organization().GetID()
	if !ok {
		return quotaCostList, errors.New("failed to get quota cost: organisation id for the current authenticated user can't be found")
	}

	strimziOperatorAddonName := fmt.Sprintf("%s-%s", AMSQuotaAddonResourceNamePrefix, o.ocmConfig.StrimziOperatorAddonID)
	fleetshardOperatorAddonName := fmt.Sprintf("%s-%s", AMSQuotaAddonResourceNamePrefix, o.ocmConfig.KasFleetshardAddonID)

	filters := []ocm.QuotaCostRelatedResourceFilter{
		// Filter for Strimzi operator add-on
		{
			ResourceName: &strimziOperatorAddonName,
		},
		// Filter for Fleetshard operator add-on
		{
			ResourceName: &fleetshardOperatorAddonName,
		},
		// Filter for osd compute nodes
		{
			ResourceType: &AMSQuotaComputeNodeResourceType,
			Product:      &AMSQuotaOSDProductName,
		},
		// Filter for osd cluster
		{
			ResourceType: &AMSQuotaClusterResourceType,
			Product:      &AMSQuotaOSDProductName,
		},
	}

	ocmQuotaCostList, err := o.ocmClient.GetQuotaCosts(orgID, true, false, filters...)
	if err != nil {
		return quotaCostList, err
	}

	for _, qc := range ocmQuotaCostList {
		quotaCostList = append(quotaCostList, types.QuotaCost{
			ID:         qc.QuotaID(),
			MaxAllowed: qc.Allowed(),
			Consumed:   qc.Consumed(),
		})
	}
	return quotaCostList, nil
}
