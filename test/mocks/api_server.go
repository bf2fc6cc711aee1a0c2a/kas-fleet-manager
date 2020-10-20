package mocks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/mux"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ocmErrors "gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"time"
)

const (
	// EndpointPathClusters ocm clusters management service clusters endpoint
	EndpointPathClusters = "/api/clusters_mgmt/v1/clusters"
	// EndpointPathClusters ocm clusters management service clusters endpoint
	EndpointPathCluster = "/api/clusters_mgmt/v1/clusters/{id}"
	// EndpointPathSyncsets ocm clusters management service syncset endpoint
	EndpointPathSyncsets = "/api/clusters_mgmt/v1/clusters/{id}/external_configuration/syncsets"
	// EndpointPathSyncsetsDelete ocm clusters management service syncset endpoint delete
	EndpointPathSyncsetsDelete = "/api/clusters_mgmt/v1/clusters/{id}/external_configuration/syncsets/{syncsetID}"
	// EndpointPathIngresses ocm cluster management ingress endpoint
	EndpointPathIngresses = "/api/clusters_mgmt/v1/clusters/{id}/ingresses"
	// EndpointPathCloudProviders ocm cluster management cloud providers endpoint
	EndpointPathCloudProviders = "/api/clusters_mgmt/v1/cloud_providers"
	// EndpointPathCloudProvider ocm cluster management cloud provider endpoint
	EndpointPathCloudProvider = "/api/clusters_mgmt/v1/cloud_providers/{id}"
	// EndpointPathCloudProviderRegions ocm cluster management cloud provider regions endpoint
	EndpointPathCloudProviderRegions = "/api/clusters_mgmt/v1/cloud_providers/{id}/regions"
	// EndpointPathCloudProviderRegion ocm cluster management cloud provider region endpoint
	EndpointPathCloudProviderRegion = "/api/clusters_mgmt/v1/cloud_providers/{providerID}/regions/{regionID}"
	// EndpointPathClusterStatus ocm cluster management cluster status endpoint
	EndpointPathClusterStatus = "/api/clusters_mgmt/v1/clusters/{id}/status"
	// EndpointPathClusterAddons ocm cluster management cluster addons endpoint
	EndpointPathClusterAddons = "/api/clusters_mgmt/v1/clusters/{id}/addons"
)

// variables for endpoints
var (
	EndpointClusterGet              = Endpoint{EndpointPathCluster, http.MethodGet}
	EndpointKafkaDelete             = Endpoint{EndpointPathSyncsetsDelete, http.MethodDelete}
	EndpointClustersGet             = Endpoint{EndpointPathClusters, http.MethodGet}
	EndpointClustersPost            = Endpoint{EndpointPathClusters, http.MethodPost}
	EndpointClusterSyncsetPost      = Endpoint{EndpointPathSyncsets, http.MethodPost}
	EndpointClusterIngressGet       = Endpoint{EndpointPathIngresses, http.MethodGet}
	EndpointCloudProvidersGet       = Endpoint{EndpointPathCloudProviders, http.MethodGet}
	EndpointCloudProviderGet        = Endpoint{EndpointPathCloudProvider, http.MethodGet}
	EndpointCloudProviderRegionsGet = Endpoint{EndpointPathCloudProviderRegions, http.MethodGet}
	EndpointCloudProviderRegionGet  = Endpoint{EndpointPathCloudProviderRegion, http.MethodGet}
	EndpointClusterStatusGet        = Endpoint{EndpointPathClusterStatus, http.MethodGet}
	EndpointClusterAddonsGet        = Endpoint{EndpointPathClusterAddons, http.MethodGet}
	EndpointClusterAddonPost        = Endpoint{EndpointPathClusterAddons, http.MethodPost}
)

// variables for mocked ocm types
//
// these are the default types that will be returned by the emulated ocm api
// to override these values, do not set them directly e.g. mocks.MockSyncset = ...
// instead use the Set*Response functions provided by MockConfigurableServerBuilder e.g. SetClusterGetResponse(...)
var (
	MockSyncset                      *clustersmgmtv1.Syncset
	MockIngressList                  *clustersmgmtv1.IngressList
	MockCloudProvider                *clustersmgmtv1.CloudProvider
	MockCloudProviderList            *clustersmgmtv1.CloudProviderList
	MockCloudProviderRegion          *clustersmgmtv1.CloudRegion
	MockCloudProviderRegionList      *clustersmgmtv1.CloudRegionList
	MockClusterStatus                *clustersmgmtv1.ClusterStatus
	MockClusterAddonInstallation     *clustersmgmtv1.AddOnInstallation
	MockClusterAddonInstallationList *clustersmgmtv1.AddOnInstallationList
	MockCluster                      *clustersmgmtv1.Cluster
)

// Endpoint is a wrapper around an endpoint and the method used to interact with that endpoint e.g. GET /clusters
type Endpoint struct {
	Path   string
	Method string
}

// HandlerRegister is a cache that maps Endpoints to their handlers
type HandlerRegister map[Endpoint]func(w http.ResponseWriter, r *http.Request)

// MockConfigurableServerBuilder allows mock ocm api servers to be built
type MockConfigurableServerBuilder struct {
	// handlerRegister cache of endpoints and handlers to be used when the mock ocm api server is built
	handlerRegister HandlerRegister
}

// NewMockConfigurableServerBuilder returns a new builder that can be used to define a mock ocm api server
func NewMockConfigurableServerBuilder() *MockConfigurableServerBuilder {
	// get the default endpoint handlers that'll be used if they're not overridden
	handlerRegister, err := getDefaultHandlerRegister()
	if err != nil {
		panic(err)
	}
	return &MockConfigurableServerBuilder{
		handlerRegister: handlerRegister,
	}
}

// SetClustersPostResponse set a mock response cluster or error for the POST /api/clusters_mgmt/v1/clusters endpoint
func (b *MockConfigurableServerBuilder) SetClusterGetResponse(cluster *clustersmgmtv1.Cluster, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterGet] = buildMockRequestHandler(cluster, err)
}

// SetKafkaDeleteResponse set a mock response cluster or error for the DELETE /api/clusters_mgmt/v1/clusters/{id}/external_configuration/syncsets/{syncsetID} endpoint
func (b *MockConfigurableServerBuilder) SetKafkaDeleteResponse(syncset *clustersmgmtv1.Syncset, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointKafkaDelete] = buildMockRequestHandler(syncset, err)
}

// SetClustersPostResponse set a mock response cluster or error for the POST /api/clusters_mgmt/v1/clusters endpoint
func (b *MockConfigurableServerBuilder) SetClustersPostResponse(cluster *clustersmgmtv1.Cluster, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClustersPost] = buildMockRequestHandler(cluster, err)
}

// SetClustersGetResponse set a mock response cluster or error for the GET /api/clusters_mgmt/v1/clusters endpoint
func (b *MockConfigurableServerBuilder) SetClustersGetResponse(cluster *clustersmgmtv1.Cluster, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClustersGet] = buildMockRequestHandler(cluster, err)
}

// SetClusterSyncsetPostResponse set a mock response syncset or error for the POST /api/clusters_mgmt/v1/clusters/{id}/syncsets endpoint
func (b *MockConfigurableServerBuilder) SetClusterSyncsetPostResponse(syncset *clustersmgmtv1.Syncset, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterSyncsetPost] = buildMockRequestHandler(syncset, err)
}

// SetIngressGetResponse set a mock response ingress or error for the GET /api/clusters_mgmt/v1/clusters/{id}/ingresses endpoint
func (b *MockConfigurableServerBuilder) SetClusterIngressGetResponse(ingress *clustersmgmtv1.Ingress, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterIngressGet] = buildMockRequestHandler(ingress, err)
}

// SetCloudProvidersGetResponse set a mock response provider list or error for GET /api/clusters_mgmt/v1/cloud_providers
func (b *MockConfigurableServerBuilder) SetCloudProvidersGetResponse(providers *clustersmgmtv1.CloudProviderList, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointCloudProvidersGet] = buildMockRequestHandler(providers, err)
}

// SetCloudRegionsGetResponse set a mock response region list or error for GET /api/clusters_mgmt/v1/cloud_providers/{id}/regions
func (b *MockConfigurableServerBuilder) SetCloudRegionsGetResponse(regions *clustersmgmtv1.CloudRegionList, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointCloudProviderRegionsGet] = buildMockRequestHandler(regions, err)
}

// SetCloudRegionGetResponse set a mock response region or error for GET /api/clusters_mgmt/v1/cloud_providers/{id}/regions/{regionId}
func (b *MockConfigurableServerBuilder) SetCloudRegionGetResponse(region *clustersmgmtv1.CloudRegion, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointCloudProviderRegionGet] = buildMockRequestHandler(region, err)
}

// SetClusterStatusGetResponse set a mock response cluster status or error for GET /api/clusters_mgmt/v1/clusters/{id}/status
func (b *MockConfigurableServerBuilder) SetClusterStatusGetResponse(status *clustersmgmtv1.ClusterStatus, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterStatusGet] = buildMockRequestHandler(status, err)
}

// SetClusterAddonsGetResponse set a mock response addon list or error for GET /api/clusters_mgmt/v1/clusters/{id}/addons
func (b *MockConfigurableServerBuilder) SetClusterAddonsGetResponse(addons *clustersmgmtv1.AddOnInstallationList, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterAddonsGet] = buildMockRequestHandler(addons, err)
}

// SetClusterAddonGetResponse set a mock response addon or error for POST /api/clusters_mgmt/v1/clusters/{id}/addons
func (b *MockConfigurableServerBuilder) SetClusterAddonPostResponse(addon *clustersmgmtv1.AddOnInstallation, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterAddonPost] = buildMockRequestHandler(addon, err)
}

// Build builds the mock ocm api server using the endpoint handlers that have been set in the builder
func (b *MockConfigurableServerBuilder) Build() *httptest.Server {
	router := mux.NewRouter()

	// set up handlers from the builder
	for endpoint, handleFn := range b.handlerRegister {
		router.HandleFunc(endpoint.Path, handleFn).Methods(endpoint.Method)
	}
	server := httptest.NewUnstartedServer(router)
	l, err := net.Listen("tcp", "127.0.0.1:9876")
	if err != nil {
		log.Fatal(err)
	}
	server.Listener = l
	server.Start()
	for i := 0; i < 5; i++ {
		res, err := http.Get("http://127.0.0.1:9876/api/clusters_mgmt/v1/cloud_providers/aws/regions")
		fmt.Println("Response: " + res.Status)
		if err != nil {
			fmt.Println("Error" + err.Error())
		}
		time.Sleep(time.Second)
	}
	return server
}

// getDefaultHandlerRegister returns a set of default endpoints and handlers used in the mock ocm api server
func getDefaultHandlerRegister() (HandlerRegister, error) {
	// define a list of default endpoints and handlers in the mock ocm api server, when new endpoints are used in the
	// managed-services-api service, a default ocm response should also be added here
	return HandlerRegister{
		EndpointClusterGet:              buildMockRequestHandler(MockCluster, nil),
		EndpointKafkaDelete:             buildMockRequestHandler(MockSyncset, nil),
		EndpointClustersGet:             buildMockRequestHandler(MockCluster, nil),
		EndpointClustersPost:            buildMockRequestHandler(MockCluster, nil),
		EndpointClusterSyncsetPost:      buildMockRequestHandler(MockSyncset, nil),
		EndpointClusterIngressGet:       buildMockRequestHandler(MockIngressList, nil),
		EndpointCloudProvidersGet:       buildMockRequestHandler(MockCloudProviderList, nil),
		EndpointCloudProviderGet:        buildMockRequestHandler(MockCloudProvider, nil),
		EndpointCloudProviderRegionsGet: buildMockRequestHandler(MockCloudProviderRegionList, nil),
		EndpointCloudProviderRegionGet:  buildMockRequestHandler(MockCloudProviderRegion, nil),
		EndpointClusterStatusGet:        buildMockRequestHandler(MockClusterStatus, nil),
		EndpointClusterAddonsGet:        buildMockRequestHandler(MockClusterAddonInstallationList, nil),
		EndpointClusterAddonPost:        buildMockRequestHandler(MockClusterAddonInstallation, nil),
	}, nil
}

// buildMockRequestHandler builds a generic handler for all ocm api server responses
// one of successType of serviceErr should be defined
// if serviceErr is defined, it will be provided as an ocm error response
// if successType is defined, it will be provided as an ocm success response
func buildMockRequestHandler(successType interface{}, serviceErr *ocmErrors.ServiceError) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if serviceErr != nil {
			w.WriteHeader(serviceErr.HttpCode)
			if err := marshalOCMType(serviceErr, w); err != nil {
				panic(err)
			}
		} else if successType != nil {
			if err := marshalOCMType(successType, w); err != nil {
				panic(err)
			}
		} else {
			panic("no response was defined")
		}
	}
}

// marshalOCMType marshals known ocm types to a provided io.Writer using the ocm sdk marshallers
func marshalOCMType(t interface{}, w io.Writer) error {
	switch t.(type) {
	// handle cluster types
	case *clustersmgmtv1.Cluster:
		return clustersmgmtv1.MarshalCluster(t.(*clustersmgmtv1.Cluster), w)
	// handle cluster status types
	case *clustersmgmtv1.ClusterStatus:
		return clustersmgmtv1.MarshalClusterStatus(t.(*clustersmgmtv1.ClusterStatus), w)
	// handle syncset types
	case *clustersmgmtv1.Syncset:
		return clustersmgmtv1.MarshalSyncset(t.(*clustersmgmtv1.Syncset), w)
	// handle ingress types
	case *clustersmgmtv1.Ingress:
		return clustersmgmtv1.MarshalIngress(t.(*clustersmgmtv1.Ingress), w)
	case []*clustersmgmtv1.Ingress:
		return clustersmgmtv1.MarshalIngressList(t.([]*clustersmgmtv1.Ingress), w)
	// for any <type>List ocm type we'll need to follow this pattern to ensure the array of objects
	// is wrapped with an OCMList object
	case *clustersmgmtv1.IngressList:
		ocmList, err := NewOCMList().WithItems(t.(*clustersmgmtv1.IngressList).Slice())
		if err != nil {
			return err
		}
		return json.NewEncoder(w).Encode(ocmList)
	// handle cloud provider types
	case *clustersmgmtv1.CloudProvider:
		return clustersmgmtv1.MarshalCloudProvider(t.(*clustersmgmtv1.CloudProvider), w)
	case []*clustersmgmtv1.CloudProvider:
		return clustersmgmtv1.MarshalCloudProviderList(t.([]*clustersmgmtv1.CloudProvider), w)
	case *clustersmgmtv1.CloudProviderList:
		ocmList, err := NewOCMList().WithItems(t.(*clustersmgmtv1.CloudProviderList).Slice())
		if err != nil {
			return err
		}
		return json.NewEncoder(w).Encode(ocmList)
	// handle cloud region types
	case *clustersmgmtv1.CloudRegion:
		return clustersmgmtv1.MarshalCloudRegion(t.(*clustersmgmtv1.CloudRegion), w)
	case []*clustersmgmtv1.CloudRegion:
		return clustersmgmtv1.MarshalCloudRegionList(t.([]*clustersmgmtv1.CloudRegion), w)
	case *clustersmgmtv1.CloudRegionList:
		ocmList, err := NewOCMList().WithItems(t.(*clustersmgmtv1.CloudRegionList).Slice())
		if err != nil {
			return err
		}
		return json.NewEncoder(w).Encode(ocmList)
	// handle cluster addon installations
	case *clustersmgmtv1.AddOnInstallation:
		return clustersmgmtv1.MarshalAddOnInstallation(t.(*clustersmgmtv1.AddOnInstallation), w)
	case []*clustersmgmtv1.AddOnInstallation:
		return clustersmgmtv1.MarshalAddOnInstallationList(t.([]*clustersmgmtv1.AddOnInstallation), w)
	case *clustersmgmtv1.AddOnInstallationList:
		ocmList, err := NewOCMList().WithItems(t.(*clustersmgmtv1.AddOnInstallationList).Slice())
		if err != nil {
			return err
		}
		return json.NewEncoder(w).Encode(ocmList)
	// handle the generic ocm list type
	case *ocmList:
		return json.NewEncoder(w).Encode(t)
	// handle ocm error type
	case *ocmErrors.ServiceError:
		return json.NewEncoder(w).Encode(t.(*ocmErrors.ServiceError).AsOpenapiError(""))
	}
	return errors.New(fmt.Sprintf("could not recognise type %s in ocm type marshaller", reflect.TypeOf(t).String()))
}

// basic wrapper to emulate the the ocm list types as they're private
type ocmList struct {
	HREF  *string         `json:"href"`
	Link  bool            `json:"link"`
	Items json.RawMessage `json:"items"`
}

func NewOCMList() *ocmList {
	return &ocmList{
		HREF:  nil,
		Link:  false,
		Items: nil,
	}
}

func (l *ocmList) WithHREF(href string) *ocmList {
	l.HREF = &href
	return l
}

func (l *ocmList) WithLink(link bool) *ocmList {
	l.Link = link
	return l
}

func (l *ocmList) WithItems(items interface{}) (*ocmList, error) {
	var b bytes.Buffer
	if err := marshalOCMType(items, &b); err != nil {
		return l, err
	}
	l.Items = b.Bytes()
	return l, nil
}

// init the shared mock types, panic if we fail, this should never fail
func init() {
	// mockClusterID default mock cluster id used in the mock ocm server
	mockClusterID := "2aad9fc1-c40e-471f-8d57-fdaecc7d3686"
	// mockCloudProviderID default mock provider ID
	mockCloudProviderID := "aws"
	// mockClusterExternalID default mock cluster external ID
	mockClusterExternalID := "2aad9fc1-c40e-471f-8d57-fdaecc7d3686"
	// mockClusterState default mock cluster state
	mockClusterState := clustersmgmtv1.ClusterStateReady
	// mockCloudProviderDisplayName default mock provider display name
	mockCloudProviderDisplayName := "AWS"
	// mockCloudRegionID default mock cluster region
	mockCloudRegionID := "eu-west-1"
	// mockCloudRegionDisplayName default mock cloud region display name
	mockCloudRegionDisplayName := "EU, Ireland"
	// mockSyncsetID default mock syncset id used in the mock ocm server
	mockSyncsetID := "ext-8a41f783-b5e4-4692-a7cd-c0b9c8eeede9"
	// mockIngressID default mock ingress id used in the mock ocm server
	mockIngressID := "s1h5"
	// mockIngressDNS default mock ingress dns used in the mock ocm server
	mockIngressDNS := "apps.ms-btq2d1h8d3b1.b3k3.s1.devshift.org"
	// mockIngressHref default mock ingress HREF used in the mock ocm server
	mockIngressHref := "/api/clusters_mgmt/v1/clusters/000/ingresses/i8y1"
	// mockIngressListening default mock ingress listening used in the mock ocm server
	mockIngressListening := clustersmgmtv1.ListeningMethodExternal
	// mockClusterAddonID default mock cluster addon ID
	mockClusterAddonID := "managed-kafka"
	// mockClusterAddonState default mock cluster addon state
	mockClusterAddonState := clustersmgmtv1.AddOnInstallationStateReady
	// mockClusterAddonDescription default mock cluster addon description
	mockClusterAddonDescription := "InstallWaiting"

	// mock syncsets
	var err error
	mockMockSyncsetBuilder := clustersmgmtv1.NewSyncset().
		ID(mockSyncsetID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/external_configuration/syncsets/%s", mockClusterID, mockSyncsetID))

	MockSyncset, err = mockMockSyncsetBuilder.Build()
	if err != nil {
		panic(err)
	}

	// mock ingresses

	MockIngressList, err = clustersmgmtv1.NewIngressList().Items(
		clustersmgmtv1.NewIngress().ID(mockIngressID).DNSName(mockIngressDNS).Default(true).Listening(mockIngressListening).HREF(mockIngressHref)).Build()
	if err != nil {
		panic(err)
	}

	// mock cloud providers

	mockCloudProviderBuilder := clustersmgmtv1.NewCloudProvider().
		ID(mockCloudProviderID).
		Name(mockCloudProviderID).
		DisplayName(mockCloudProviderDisplayName).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/cloud_providers/%s", mockCloudProviderID))

	MockCloudProvider, err = mockCloudProviderBuilder.Build()
	if err != nil {
		panic(err)
	}

	MockCloudProviderList, err = clustersmgmtv1.NewCloudProviderList().
		Items(mockCloudProviderBuilder).
		Build()
	if err != nil {
		panic(err)
	}

	// mock cloud provider regions/cloud regions

	mockCloudProviderRegionBuilder := clustersmgmtv1.NewCloudRegion().
		ID(mockCloudRegionID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/cloud_providers/%s/regions/%s", mockCloudProviderID, mockCloudRegionID)).
		DisplayName(mockCloudRegionDisplayName).
		CloudProvider(mockCloudProviderBuilder).
		Enabled(true).
		SupportsMultiAZ(true)

	MockCloudProviderRegion, err = mockCloudProviderRegionBuilder.Build()
	if err != nil {
		panic(err)
	}

	MockCloudProviderRegionList, err = clustersmgmtv1.NewCloudRegionList().Items(mockCloudProviderRegionBuilder).Build()
	if err != nil {
		panic(err)
	}

	// mock cluster status

	MockClusterStatus, err = clustersmgmtv1.NewClusterStatus().
		ID(mockClusterID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/status", mockClusterID)).
		State(mockClusterState).
		Description("").
		Build()

	// mock cluster addons

	mockClusterAddonBuilder := clustersmgmtv1.NewAddOn().
		ID(mockClusterAddonID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/addons/%s", mockClusterAddonID))

	// mock cluster addon installations

	mockClusterAddonInstallationBuilder := clustersmgmtv1.NewAddOnInstallation().
		ID(mockClusterAddonID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/addons/managed-kafka", mockClusterID)).
		Addon(mockClusterAddonBuilder).
		State(mockClusterAddonState).
		StateDescription(mockClusterAddonDescription)

	MockClusterAddonInstallation, err = mockClusterAddonInstallationBuilder.Build()
	if err != nil {
		panic(err)
	}

	MockClusterAddonInstallationList, err = clustersmgmtv1.NewAddOnInstallationList().Items(mockClusterAddonInstallationBuilder).Build()
	if err != nil {
		panic(err)
	}

	// mock cluster

	MockCluster, err = clustersmgmtv1.NewCluster().
		ID(mockClusterID).
		ExternalID(mockClusterExternalID).
		State(mockClusterState).
		CloudProvider(mockCloudProviderBuilder).
		Region(mockCloudProviderRegionBuilder).
		Build()
	if err != nil {
		panic(err)
	}
}
