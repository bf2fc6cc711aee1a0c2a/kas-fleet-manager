package ocm

import (
	"fmt"
	"net/http"
	"time"

	"github.com/patrickmn/go-cache"
	pkgerrors "github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	sdkClient "github.com/openshift-online/ocm-sdk-go"
	amsv1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/authorizations/v1"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
)

const TERMS_SITECODE = "OCM"
const TERMS_EVENTCODE_ONLINE_SERVICE = "onlineService"
const TERMS_EVENTCODE_REGISTER = "register"

//go:generate moq -out client_moq.go . Client
type Client interface {
	CreateCluster(cluster *clustersmgmtv1.Cluster) (*clustersmgmtv1.Cluster, error)
	GetClusterIngresses(clusterID string) (*clustersmgmtv1.IngressesListResponse, error)
	GetCluster(clusterID string) (*clustersmgmtv1.Cluster, error)
	GetClusterStatus(id string) (*clustersmgmtv1.ClusterStatus, error)
	GetCloudProviders() (*clustersmgmtv1.CloudProviderList, error)
	GetRegions(provider *clustersmgmtv1.CloudProvider) (*clustersmgmtv1.CloudRegionList, error)
	GetAddon(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error)
	CreateAddonWithParams(clusterId string, addonId string, parameters []Parameter) (*clustersmgmtv1.AddOnInstallation, error)
	CreateAddon(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error)
	UpdateAddonParameters(clusterId string, addonId string, parameters []Parameter) (*clustersmgmtv1.AddOnInstallation, error)
	GetClusterDNS(clusterID string) (string, error)
	CreateSyncSet(clusterID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error)
	UpdateSyncSet(clusterID string, syncSetID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error)
	GetSyncSet(clusterID string, syncSetID string) (*clustersmgmtv1.Syncset, error)
	DeleteSyncSet(clusterID string, syncsetID string) (int, error)
	CreateIdentityProvider(clusterID string, identityProvider *clustersmgmtv1.IdentityProvider) (*clustersmgmtv1.IdentityProvider, error)
	GetIdentityProviderList(clusterID string) (*clustersmgmtv1.IdentityProviderList, error)
	DeleteCluster(clusterID string) (int, error)
	ClusterAuthorization(cb *amsv1.ClusterAuthorizationRequest) (*amsv1.ClusterAuthorizationResponse, error)
	DeleteSubscription(id string) (int, error)
	FindSubscriptions(query string) (*amsv1.SubscriptionsListResponse, error)
	GetRequiresTermsAcceptance(username string) (termsRequired bool, redirectUrl string, err error)
	GetOrganisationIdFromExternalId(externalId string) (string, error)
	Connection() *sdkClient.Connection
	GetMachinePool(clusterID string, machinePoolID string) (*clustersmgmtv1.MachinePool, error)
	CreateMachinePool(clusterID string, machinePool *clustersmgmtv1.MachinePool) (*clustersmgmtv1.MachinePool, error)
	// GetQuotaCosts returns a list of quota cost for the given organizationID.
	// Each quota cost contains information on the usage and max allowed ocm resources quota given to the specified oganization.
	//
	// relatedResourceFilters will only be applied when fetchRelatedResources is set to true.
	GetQuotaCosts(organizationID string, fetchRelatedResources, fetchCloudAccounts bool, filters ...QuotaCostRelatedResourceFilter) ([]*amsv1.QuotaCost, error)
	GetQuotaCostsForProduct(organizationID, resourceName, product string) ([]*amsv1.QuotaCost, error)
	// GetCurrentAccount returns the account information of the current authenticated user
	GetCurrentAccount() (*amsv1.Account, error)
}

var _ Client = &client{}

type client struct {
	connection *sdkClient.Connection
	cache      *cache.Cache
}

type AMSClient Client
type ClusterManagementClient Client

func NewOCMConnection(ocmConfig *OCMConfig, BaseUrl string) (*sdkClient.Connection, func(), error) {
	if ocmConfig.EnableMock && ocmConfig.MockMode != MockModeEmulateServer {
		return nil, func() {}, nil
	}

	builder := sdkClient.NewConnectionBuilder().
		URL(BaseUrl).
		MetricsSubsystem("api_outbound")

	if !ocmConfig.EnableMock {
		// Create a logger that has the debug level enabled:
		logger, err := sdkClient.NewGoLoggerBuilder().
			Debug(ocmConfig.Debug).
			Build()
		if err != nil {
			return nil, nil, err
		}
		builder = builder.Logger(logger)
	}

	if ocmConfig.ClientID != "" && ocmConfig.ClientSecret != "" {
		builder = builder.Client(ocmConfig.ClientID, ocmConfig.ClientSecret)
	} else if ocmConfig.SelfToken != "" {
		builder = builder.Tokens(ocmConfig.SelfToken)
	} else {
		return nil, nil, fmt.Errorf("can't build OCM client connection,no Client/Secret or Token has been provided")
	}

	connection, err := builder.Build()
	if err != nil {
		return nil, nil, err
	}
	return connection, func() {
		_ = connection.Close()
	}, nil

}

func NewClient(connection *sdkClient.Connection) Client {
	return &client{
		connection: connection,
		cache:      cache.New(168*time.Hour, 1*time.Hour),
	}
}

func (c *client) Connection() *sdkClient.Connection {
	return c.connection
}

func (c *client) Close() {
	if c.connection != nil {
		_ = c.connection.Close()
	}
}

func (c *client) CreateCluster(cluster *clustersmgmtv1.Cluster) (*clustersmgmtv1.Cluster, error) {

	clusterResource := c.connection.ClustersMgmt().V1().Clusters()
	response, err := clusterResource.Add().Body(cluster).Send()
	if err != nil {
		return &clustersmgmtv1.Cluster{}, errors.New(errors.ErrorGeneral, err.Error())
	}
	createdCluster := response.Body()

	return createdCluster, nil
}

func (c *client) GetOrganisationIdFromExternalId(externalId string) (string, error) {
	orgId, cached := c.cache.Get(externalId)
	if cached {
		orgId, ok := orgId.(string)
		if ok && orgId != "" {
			return orgId, nil
		}
	}

	res, err := c.connection.AccountsMgmt().V1().Organizations().List().Search(fmt.Sprintf("external_id='%s'", externalId)).Send()
	if err != nil {
		return "", err
	}

	items := res.Items()
	if items.Len() < 1 {
		// should never happen...
		return "", errors.New(errors.ErrorGeneral, "organisation with external_id '%s' can't be found", externalId)
	}

	organisationId := items.Get(0).ID()
	c.cache.Set(externalId, organisationId, cache.DefaultExpiration)
	return organisationId, nil
}

func (c *client) GetRequiresTermsAcceptance(username string) (termsRequired bool, redirectUrl string, err error) {
	// Check for Appendix 4 Terms
	request, err := v1.NewTermsReviewRequest().AccountUsername(username).SiteCode(TERMS_SITECODE).EventCode(TERMS_EVENTCODE_REGISTER).Build()
	if err != nil {
		return false, "", err
	}
	selfTermsReview := c.connection.Authorizations().V1().TermsReview()
	postResp, err := selfTermsReview.Post().Request(request).Send()
	if err != nil {
		return false, "", err
	}
	response, ok := postResp.GetResponse()
	if !ok {
		return false, "", fmt.Errorf("empty response from authorization post request")
	}

	redirectUrl, _ = response.GetRedirectUrl()

	return response.TermsRequired(), redirectUrl, nil
}

// GetClusterIngresses sends a GET request to ocm to retrieve the ingresses of an OSD cluster
func (c *client) GetClusterIngresses(clusterID string) (*clustersmgmtv1.IngressesListResponse, error) {
	clusterIngresses := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).Ingresses()
	ingressList, err := clusterIngresses.List().Send()
	if err != nil {
		return nil, err
	}

	return ingressList, nil
}

func (c client) GetCluster(clusterID string) (*clustersmgmtv1.Cluster, error) {
	resp, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).Get().Send()
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c client) GetClusterStatus(id string) (*clustersmgmtv1.ClusterStatus, error) {
	resp, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(id).Status().Get().Send()
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c *client) GetCloudProviders() (*clustersmgmtv1.CloudProviderList, error) {
	providersCollection := c.connection.ClustersMgmt().V1().CloudProviders()
	providersResponse, err := providersCollection.List().Send()
	if err != nil {
		return nil, pkgerrors.Wrap(err, "error retrieving cloud provider list")
	}
	cloudProviderList := providersResponse.Items()
	return cloudProviderList, nil
}

func (c *client) GetRegions(provider *clustersmgmtv1.CloudProvider) (*clustersmgmtv1.CloudRegionList, error) {
	regionsCollection := c.connection.ClustersMgmt().V1().CloudProviders().CloudProvider(provider.ID()).Regions()
	regionsResponse, err := regionsCollection.List().Send()
	if err != nil {
		return nil, pkgerrors.Wrap(err, "error retrieving cloud region list")
	}

	regionList := regionsResponse.Items()
	return regionList, nil
}

func (c client) CreateAddonWithParams(clusterId string, addonId string, params []Parameter) (*clustersmgmtv1.AddOnInstallation, error) {
	addon := clustersmgmtv1.NewAddOn().ID(addonId)
	addonParameters := newAddonParameterListBuilder(params)
	addonInstallationBuilder := clustersmgmtv1.NewAddOnInstallation().Addon(addon)
	if addonParameters != nil {
		addonInstallationBuilder = addonInstallationBuilder.Parameters(addonParameters)
	}
	addonInstallation, err := addonInstallationBuilder.Build()
	if err != nil {
		return nil, err
	}
	resp, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterId).Addons().Add().Body(addonInstallation).Send()
	if err != nil {
		return nil, err
	}
	return resp.Body(), nil
}

func (c client) CreateAddon(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
	return c.CreateAddonWithParams(clusterId, addonId, []Parameter{})
}

func (c client) GetAddon(clusterId string, addonId string) (*clustersmgmtv1.AddOnInstallation, error) {
	resp, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterId).Addons().List().Send()
	if err != nil {
		return nil, err
	}

	addon := &clustersmgmtv1.AddOnInstallation{}
	resp.Items().Each(func(addOnInstallation *clustersmgmtv1.AddOnInstallation) bool {
		if addOnInstallation.ID() == addonId {
			addon = addOnInstallation
			return false
		}
		return true
	})

	return addon, nil
}

func (c client) UpdateAddonParameters(clusterId string, addonInstallationId string, parameters []Parameter) (*clustersmgmtv1.AddOnInstallation, error) {
	addonInstallationResp, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterId).Addons().Addoninstallation(addonInstallationId).Get().Send()
	if err != nil {
		return nil, err
	}
	if existingParameters, ok := addonInstallationResp.Body().GetParameters(); ok {
		if sameParameters(existingParameters, parameters) {
			return addonInstallationResp.Body(), nil
		}
	}
	addonInstallationBuilder := clustersmgmtv1.NewAddOnInstallation()
	updatedParamsListBuilder := newAddonParameterListBuilder(parameters)
	if updatedParamsListBuilder != nil {
		addonInstallation, err := addonInstallationBuilder.Parameters(updatedParamsListBuilder).Build()
		if err != nil {
			return nil, err
		}
		resp, err := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterId).Addons().Addoninstallation(addonInstallationId).Update().Body(addonInstallation).Send()
		if err != nil {
			return nil, err
		}
		return resp.Body(), nil
	}
	return addonInstallationResp.Body(), nil
}

func (c *client) GetClusterDNS(clusterID string) (string, error) {
	if clusterID == "" {
		return "", errors.Validation("clusterID cannot be empty")
	}
	ingresses, err := c.GetClusterIngresses(clusterID)
	if err != nil {
		return "", err
	}

	var clusterDNS string
	ingresses.Items().Each(func(ingress *clustersmgmtv1.Ingress) bool {
		if ingress.Default() {
			clusterDNS = ingress.DNSName()
			return false
		}
		return true
	})

	if clusterDNS == "" {
		return "", errors.NotFound("cluster %s: DNS is empty", clusterID)
	}

	return clusterDNS, nil
}

func (c client) CreateSyncSet(clusterID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {

	clustersResource := c.connection.ClustersMgmt().V1().Clusters()
	response, syncsetErr := clustersResource.Cluster(clusterID).
		ExternalConfiguration().
		Syncsets().
		Add().
		Body(syncset).
		Send()
	var err error
	if syncsetErr != nil {
		err = errors.NewErrorFromHTTPStatusCode(response.Status(), "ocm client failed to create syncset: %s", syncsetErr)
	}
	return response.Body(), err
}

func (c client) UpdateSyncSet(clusterID string, syncSetID string, syncset *clustersmgmtv1.Syncset) (*clustersmgmtv1.Syncset, error) {

	clustersResource := c.connection.ClustersMgmt().V1().Clusters()
	response, syncsetErr := clustersResource.Cluster(clusterID).
		ExternalConfiguration().
		Syncsets().
		Syncset(syncSetID).
		Update().
		Body(syncset).
		Send()

	var err error
	if syncsetErr != nil {
		err = errors.NewErrorFromHTTPStatusCode(response.Status(), "ocm client failed to update syncset '%s': %s", syncSetID, syncsetErr)
	}
	return response.Body(), err
}

func (c client) CreateIdentityProvider(clusterID string, identityProvider *clustersmgmtv1.IdentityProvider) (*clustersmgmtv1.IdentityProvider, error) {

	clustersResource := c.connection.ClustersMgmt().V1().Clusters()
	response, identityProviderErr := clustersResource.Cluster(clusterID).
		IdentityProviders().
		Add().
		Body(identityProvider).
		Send()
	var err error
	if identityProviderErr != nil {
		err = errors.NewErrorFromHTTPStatusCode(response.Status(), "ocm client failed to create identity provider: %s", identityProviderErr)
	}
	return response.Body(), err
}

func (c client) GetIdentityProviderList(clusterID string) (*clustersmgmtv1.IdentityProviderList, error) {
	clusterResource := c.connection.ClustersMgmt().V1().Clusters()
	response, getIDPErr := clusterResource.Cluster(clusterID).
		IdentityProviders().
		List().
		Send()

	if getIDPErr != nil {
		return nil, errors.NewErrorFromHTTPStatusCode(response.Status(), "ocm client failed to get list of identity providers, err: %s", getIDPErr.Error())
	}
	return response.Items(), nil
}

func (c client) GetSyncSet(clusterID string, syncSetID string) (*clustersmgmtv1.Syncset, error) {
	clustersResource := c.connection.ClustersMgmt().V1().Clusters()
	response, syncsetErr := clustersResource.Cluster(clusterID).
		ExternalConfiguration().
		Syncsets().
		Syncset(syncSetID).
		Get().
		Send()

	var err error
	if syncsetErr != nil {
		err = errors.NewErrorFromHTTPStatusCode(response.Status(), "ocm client failed to get syncset '%s': %s", syncSetID, syncsetErr)
	}
	return response.Body(), err
}

// Status returns the response status code.
func (c client) DeleteSyncSet(clusterID string, syncsetID string) (int, error) {
	clustersResource := c.connection.ClustersMgmt().V1().Clusters()
	response, syncsetErr := clustersResource.Cluster(clusterID).
		ExternalConfiguration().
		Syncsets().
		Syncset(syncsetID).
		Delete().
		Send()
	return response.Status(), syncsetErr
}

func newAddonParameterListBuilder(params []Parameter) *clustersmgmtv1.AddOnInstallationParameterListBuilder {
	if len(params) > 0 {
		var items []*clustersmgmtv1.AddOnInstallationParameterBuilder
		for _, p := range params {
			pb := clustersmgmtv1.NewAddOnInstallationParameter().ID(p.Id).Value(p.Value)
			items = append(items, pb)
		}
		return clustersmgmtv1.NewAddOnInstallationParameterList().Items(items...)
	}
	return nil
}

func sameParameters(parameterList *clustersmgmtv1.AddOnInstallationParameterList, params []Parameter) bool {
	if parameterList.Len() != len(params) {
		return false
	}
	paramsMap := map[string]string{}
	for _, p := range params {
		paramsMap[p.Id] = p.Value
	}
	match := true
	parameterList.Each(func(item *clustersmgmtv1.AddOnInstallationParameter) bool {
		if paramsMap[item.ID()] != item.Value() {
			match = false
			return false
		}
		return true
	})
	return match
}

func (c client) DeleteCluster(clusterID string) (int, error) {
	clustersResource := c.connection.ClustersMgmt().V1().Clusters()
	response, deleteClusterError := clustersResource.Cluster(clusterID).Delete().Send()

	var err error
	if deleteClusterError != nil {
		err = errors.NewErrorFromHTTPStatusCode(response.Status(), "OCM client failed to delete cluster '%s': %s", clusterID, deleteClusterError)
	}
	return response.Status(), err
}

func (c client) ClusterAuthorization(cb *amsv1.ClusterAuthorizationRequest) (*amsv1.ClusterAuthorizationResponse, error) {
	r, err := c.connection.AccountsMgmt().V1().
		ClusterAuthorizations().
		Post().Request(cb).Send()
	if err != nil && r.Status() != http.StatusTooManyRequests {
		err = errors.NewErrorFromHTTPStatusCode(r.Status(), "OCM client failed to create cluster authorization")
		return nil, err
	}
	resp, _ := r.GetResponse()
	return resp, nil
}

func (c client) DeleteSubscription(id string) (int, error) {
	r := c.connection.AccountsMgmt().V1().Subscriptions().Subscription(id).Delete()
	resp, err := r.Send()
	return resp.Status(), err
}

func (c client) FindSubscriptions(query string) (*amsv1.SubscriptionsListResponse, error) {
	r, err := c.connection.AccountsMgmt().V1().Subscriptions().List().Search(query).Send()
	if err != nil {
		return nil, err
	}
	return r, nil
}

// GetMachinePool returns the machinePoolID associated to clusterID.
// If the cluster does not exist the returned MachinePool is nil
func (c *client) GetMachinePool(clusterID string, machinePoolID string) (*clustersmgmtv1.MachinePool, error) {
	machinePoolsClient := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID).MachinePools()
	resp, err := machinePoolsClient.MachinePool(machinePoolID).Get().Send()
	if resp.Status() == http.StatusNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return resp.Body(), nil
}

// CreateMachinePool creates the provided MachinePool in OCM.
// The created MachinePool or an error is returned
func (c *client) CreateMachinePool(clusterID string, machinePool *clustersmgmtv1.MachinePool) (*clustersmgmtv1.MachinePool, error) {
	clusterClient := c.connection.ClustersMgmt().V1().Clusters().Cluster(clusterID)
	machinePoolsClient := clusterClient.MachinePools()
	response, err := machinePoolsClient.Add().Body(machinePool).Send()
	if err != nil {
		return nil, errors.New(errors.ErrorGeneral, err.Error())
	}
	createdMachinePool := response.Body()

	return createdMachinePool, nil
}

// QuotaCostRelatedResourceFilter represents the properties of the related resource, associated
// to each quota cost, that can be used to filter the result of the get quota costs request.
// Any property set to nil will not be applied as a filter.
type QuotaCostRelatedResourceFilter struct {
	ResourceName *string
	ResourceType *string
	Product      *string
}

// IsMatch returns true if all the given properties of the filter matches that of the given related resource.
// If a filter property was not specified, the check for that property will always return true
func (qcf *QuotaCostRelatedResourceFilter) IsMatch(relatedResource *amsv1.RelatedResource) bool {
	resourceNameMatches := (qcf.ResourceName == nil || relatedResource.ResourceName() == *qcf.ResourceName)
	resourceTypeMatches := (qcf.ResourceType == nil || relatedResource.ResourceType() == *qcf.ResourceType)
	productMatches := (qcf.Product == nil || relatedResource.Product() == *qcf.Product)

	return resourceNameMatches && resourceTypeMatches && productMatches
}

// GetQuotaCosts returns a list of quota cost for the given organizationID.
// Each quota cost contains information on the usage and max allowed ocm resources quota given to the specified oganization.
//
// relatedResourceFilters will only be applied when fetchRelatedResources is set to true.
// When relatedResourceFilters is not specified, all the quotas are returned.
// When relatedResourceFilters is specified, a quota is returned if one of the filters matches the related resources.
func (c client) GetQuotaCosts(organizationID string, fetchRelatedResources, fetchCloudAccounts bool, relatedResourceFilters ...QuotaCostRelatedResourceFilter) ([]*amsv1.QuotaCost, error) {
	organizationClient := c.connection.AccountsMgmt().V1().Organizations()
	quotaCostClient := organizationClient.Organization(organizationID).QuotaCost()

	quotaCostResp, err := quotaCostClient.List().Parameter("fetchRelatedResources", fetchRelatedResources).Parameter("fetchCloudAccounts", fetchCloudAccounts).Send()
	if err != nil {
		return nil, err
	}
	quotaCostList := quotaCostResp.Items()

	if !fetchRelatedResources || len(relatedResourceFilters) == 0 {
		return quotaCostList.Slice(), nil
	}

	var quotaCosts []*amsv1.QuotaCost

	// iterates through all Quota and only return the quota whose resources matches the given filters
	quotaCostList.Each(func(qc *amsv1.QuotaCost) bool {
		relatedResources := qc.RelatedResources()
		quotaMatchesFilters := false
		for _, relatedResource := range relatedResources {
			for _, filter := range relatedResourceFilters {
				if filter.IsMatch(relatedResource) {
					quotaMatchesFilters = true
					break
				}
			}
			if quotaMatchesFilters {
				quotaCosts = append(quotaCosts, qc)
				break
			}
		}
		return true
	})

	return quotaCosts, nil
}

// GetQuotaCostsForProduct gets the AMS QuotaCosts in the given organizationID
// whose relatedResources contains at least a relatedResource that has the
// given resourceName and product
func (c client) GetQuotaCostsForProduct(organizationID, resourceName, product string) ([]*amsv1.QuotaCost, error) {
	quotaCostList, err := c.GetQuotaCosts(organizationID, true, true, QuotaCostRelatedResourceFilter{
		ResourceName: &resourceName,
		Product:      &product,
	})

	if err != nil {
		return nil, err
	}

	return quotaCostList, nil
}

// GetCurrentAccount returns the account information of the current authenticated user
func (c *client) GetCurrentAccount() (*amsv1.Account, error) {
	currentAccountClient := c.connection.AccountsMgmt().V1().CurrentAccount()
	response, err := currentAccountClient.Get().Send()
	if err != nil {
		return nil, err
	}

	currentAccount := response.Body()
	return currentAccount, nil
}
