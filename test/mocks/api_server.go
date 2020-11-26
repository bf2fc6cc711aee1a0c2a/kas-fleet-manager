package mocks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"

	"k8s.io/apimachinery/pkg/util/wait"

	"time"

	"github.com/gorilla/mux"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ocmErrors "gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

const (
	// EndpointPathClusters ocm clusters management service clusters endpoint
	EndpointPathClusters = "/api/clusters_mgmt/v1/clusters"
	// EndpointPathCluster ocm clusters management service clusters endpoint
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
	// EndpointPathMachinePools ocm cluster management machine pools endpoint
	EndpointPathMachinePools = "/api/clusters_mgmt/v1/clusters/{id}/machine_pools"
	// EndpointPathMachinePool ocm cluster management machine pool endpoint
	EndpointPathMachinePool = "/api/clusters_mgmt/v1/clusters/{id}/machine_pools/{machinePoolId}"

	// Default values for getX functions

	// MockClusterID default mock cluster id used in the mock ocm server
	MockClusterID = "2aad9fc1-c40e-471f-8d57-fdaecc7d3686"
	// MockCloudProviderID default mock provider ID
	MockCloudProviderID = "aws"
	// MockClusterExternalID default mock cluster external ID
	MockClusterExternalID = "2aad9fc1-c40e-471f-8d57-fdaecc7d3686"
	// MockClusterState default mock cluster state
	MockClusterState = clustersmgmtv1.ClusterStateReady
	// MockCloudProviderDisplayName default mock provider display name
	MockCloudProviderDisplayName = "AWS"
	// MockCloudRegionID default mock cluster region
	MockCloudRegionID = "us-east-1"
	// MockCloudRegionDisplayName default mock cloud region display name
	MockCloudRegionDisplayName = "US East, N. Virginia"
	// MockSyncsetID default mock syncset id used in the mock ocm server
	MockSyncsetID = "ext-8a41f783-b5e4-4692-a7cd-c0b9c8eeede9"
	// MockIngressID default mock ingress id used in the mock ocm server
	MockIngressID = "s1h5"
	// MockIngressDNS default mock ingress dns used in the mock ocm server
	MockIngressDNS = "apps.ms-btq2d1h8d3b1.b3k3.s1.devshift.org"
	// MockIngressHref default mock ingress HREF used in the mock ocm server
	MockIngressHref = "/api/clusters_mgmt/v1/clusters/000/ingresses/i8y1"
	// MockIngressListening default mock ingress listening used in the mock ocm server
	MockIngressListening = clustersmgmtv1.ListeningMethodExternal
	// MockClusterAddonID default mock cluster addon ID
	MockClusterAddonID = "managed-kafka"
	// MockClusterAddonState default mock cluster addon state
	MockClusterAddonState = clustersmgmtv1.AddOnInstallationStateReady
	// MockClusterAddonDescription default mock cluster addon description
	MockClusterAddonDescription = "InstallWaiting"
	// MockMachinePoolID default machine pool ID
	MockMachinePoolID = "managed"
	// MockMachinePoolReplicas default number of machine pool replicas
	MockMachinePoolReplicas = 3
	// MockOpenshiftVersion default cluster openshift version
	MockOpenshiftVersion = "openshift-v4.6.1"
	//MockMultiAZ default value
	MockMultiAZ = true
)

// variables for endpoints
var (
	EndpointClusterGet              = Endpoint{EndpointPathCluster, http.MethodGet}
	EndpointKafkaDelete             = Endpoint{EndpointPathSyncsetsDelete, http.MethodDelete}
	EndpointClustersGet             = Endpoint{EndpointPathClusters, http.MethodGet}
	EndpointClustersPost            = Endpoint{EndpointPathClusters, http.MethodPost}
	EndpointClustersPatch           = Endpoint{EndpointPathCluster, http.MethodPatch}
	EndpointClusterSyncsetPost      = Endpoint{EndpointPathSyncsets, http.MethodPost}
	EndpointClusterIngressGet       = Endpoint{EndpointPathIngresses, http.MethodGet}
	EndpointCloudProvidersGet       = Endpoint{EndpointPathCloudProviders, http.MethodGet}
	EndpointCloudProviderGet        = Endpoint{EndpointPathCloudProvider, http.MethodGet}
	EndpointCloudProviderRegionsGet = Endpoint{EndpointPathCloudProviderRegions, http.MethodGet}
	EndpointCloudProviderRegionGet  = Endpoint{EndpointPathCloudProviderRegion, http.MethodGet}
	EndpointClusterStatusGet        = Endpoint{EndpointPathClusterStatus, http.MethodGet}
	EndpointClusterAddonsGet        = Endpoint{EndpointPathClusterAddons, http.MethodGet}
	EndpointClusterAddonPost        = Endpoint{EndpointPathClusterAddons, http.MethodPost}
	EndpointMachinePoolsGet         = Endpoint{EndpointPathMachinePools, http.MethodGet}
	EndpointMachinePoolPost         = Endpoint{EndpointPathMachinePools, http.MethodPost}
	EndpointMachinePoolPatch        = Endpoint{EndpointPathMachinePool, http.MethodPatch}
	EndpointMachinePoolGet          = Endpoint{EndpointPathMachinePool, http.MethodGet}
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
	MockMachinePoolList              *clustersmgmtv1.MachinePoolList
	MockMachinePool                  *clustersmgmtv1.MachinePool
	MockCluster                      *clustersmgmtv1.Cluster
)

// routerSwapper is an http.Handler that allows you to swap mux routers.
type routerSwapper struct {
	mu     sync.Mutex
	router *mux.Router
}

// Swap changes the old router with the new one.
func (rs *routerSwapper) Swap(newRouter *mux.Router) {
	rs.mu.Lock()
	rs.router = newRouter
	rs.mu.Unlock()
}

var router *mux.Router

// rSwapper is required if any change to the Router for mocked OCM server is needed
var rSwapper *routerSwapper

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

// SetClusterGetResponse set a mock response cluster or error for the POST /api/clusters_mgmt/v1/clusters endpoint
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

// SetClustersPatchResponse set a mock response cluster or error for the GET /api/clusters_mgmt/v1/clusters endpoint
func (b *MockConfigurableServerBuilder) SetClustersPatchResponse(cluster *clustersmgmtv1.Cluster, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClustersPatch] = buildMockRequestHandler(cluster, err)
}

// SetClusterSyncsetPostResponse set a mock response syncset or error for the POST /api/clusters_mgmt/v1/clusters/{id}/syncsets endpoint
func (b *MockConfigurableServerBuilder) SetClusterSyncsetPostResponse(syncset *clustersmgmtv1.Syncset, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterSyncsetPost] = buildMockRequestHandler(syncset, err)
}

// SetClusterIngressGetResponse set a mock response ingress or error for the GET /api/clusters_mgmt/v1/clusters/{id}/ingresses endpoint
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

// SetClusterAddonPostResponse set a mock response addon or error for POST /api/clusters_mgmt/v1/clusters/{id}/addons
func (b *MockConfigurableServerBuilder) SetClusterAddonPostResponse(addon *clustersmgmtv1.AddOnInstallation, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointClusterAddonPost] = buildMockRequestHandler(addon, err)
}

// SetMachinePoolsGetResponse set a mock response machine pool or error for Get /api/clusters_mgmt/v1/clusters/{id}/machine_pools
func (b *MockConfigurableServerBuilder) SetMachinePoolsGetResponse(mp *clustersmgmtv1.MachinePoolList, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointMachinePoolsGet] = buildMockRequestHandler(mp, err)
}

// SetMachinePoolGetResponse set a mock response machine pool list or error for Get /api/clusters_mgmt/v1/clusters/{id}/machine_pools/{machinePoolId}
func (b *MockConfigurableServerBuilder) SetMachinePoolGetResponse(mp *clustersmgmtv1.MachinePoolList, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointMachinePoolGet] = buildMockRequestHandler(mp, err)
}

// SetMachinePoolPostResponse set a mock response for Post /api/clusters_mgmt/v1/clusters/{id}/machine_pools
func (b *MockConfigurableServerBuilder) SetMachinePoolPostResponse(mp *clustersmgmtv1.MachinePool, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointMachinePoolPost] = buildMockRequestHandler(mp, err)
}

// SetMachinePoolPatchResponse set a mock response for Patch /api/clusters_mgmt/v1/clusters/{id}/machine_pools/{machinePoolId}
func (b *MockConfigurableServerBuilder) SetMachinePoolPatchResponse(mp *clustersmgmtv1.MachinePool, err *ocmErrors.ServiceError) {
	b.handlerRegister[EndpointMachinePoolPatch] = buildMockRequestHandler(mp, err)
}

// Build builds the mock ocm api server using the endpoint handlers that have been set in the builder
func (b *MockConfigurableServerBuilder) Build() *httptest.Server {
	router = mux.NewRouter()
	rSwapper = &routerSwapper{sync.Mutex{}, router}

	// set up handlers from the builder
	for endpoint, handleFn := range b.handlerRegister {
		router.HandleFunc(endpoint.Path, handleFn).Methods(endpoint.Method)
	}
	server := httptest.NewUnstartedServer(rSwapper)
	l, err := net.Listen("tcp", "127.0.0.1:9876")
	if err != nil {
		log.Fatal(err)
	}
	server.Listener = l
	server.Start()
	err = wait.PollImmediate(time.Second, 10*time.Second, func() (done bool, err error) {
		_, err = http.Get("http://127.0.0.1:9876/api/clusters_mgmt/v1/cloud_providers/aws/regions")
		return err == nil, nil
	})
	if err != nil {
		log.Fatal("Timed out waiting for mock server to start.")
		panic(err)
	}
	return server
}

// SwapRouterResponse and update the router to handle this response
func (b *MockConfigurableServerBuilder) SwapRouterResponse(path string, method string, successType interface{}, serviceErr *ocmErrors.ServiceError) {
	b.handlerRegister[Endpoint{
		Path:   path,
		Method: method,
	}] = buildMockRequestHandler(successType, serviceErr)

	router = mux.NewRouter()
	for endpoint, handleFn := range b.handlerRegister {
		router.HandleFunc(endpoint.Path, handleFn).Methods(endpoint.Method)
	}

	rSwapper.Swap(router)
}

// ServeHTTP makes the routerSwapper to implement the http.Handler interface
// so that routerSwapper can be used by httptest.NewServer()
func (rs *routerSwapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rs.mu.Lock()
	router := rs.router
	rs.mu.Unlock()
	router.ServeHTTP(w, r)
}

// getDefaultHandlerRegister returns a set of default endpoints and handlers used in the mock ocm api server
func getDefaultHandlerRegister() (HandlerRegister, error) {
	// define a list of default endpoints and handlers in the mock ocm api server, when new endpoints are used in the
	// managed-services-api service, a default ocm response should also be added here
	return HandlerRegister{
		EndpointClusterGet:              buildMockRequestHandler(MockCluster, nil),
		EndpointKafkaDelete:             buildMockRequestHandler(MockSyncset, nil),
		EndpointClustersGet:             buildMockRequestHandler(MockCluster, nil),
		EndpointClustersPatch:           buildMockRequestHandler(MockCluster, nil),
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
		EndpointMachinePoolsGet:         buildMockRequestHandler(MockMachinePoolList, nil),
		EndpointMachinePoolGet:          buildMockRequestHandler(MockMachinePool, nil),
		EndpointMachinePoolPatch:        buildMockRequestHandler(MockMachinePool, nil),
		EndpointMachinePoolPost:         buildMockRequestHandler(MockMachinePool, nil),
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
	case *clustersmgmtv1.MachinePool:
		return clustersmgmtv1.MarshalMachinePool(t.(*clustersmgmtv1.MachinePool), w)
	case []*clustersmgmtv1.MachinePool:
		return clustersmgmtv1.MarshalMachinePoolList(t.([]*clustersmgmtv1.MachinePool), w)
	case *clustersmgmtv1.MachinePoolList:
		ocmList, err := NewOCMList().WithItems(t.(*clustersmgmtv1.MachinePoolList).Slice())
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
	var err error
	// mock syncsets
	mockMockSyncsetBuilder := GetMockSyncsetBuilder(nil)
	MockSyncset, err = GetMockSyncset(mockMockSyncsetBuilder)
	if err != nil {
		panic(err)
	}

	// mock ingresses
	MockIngressList, err = GetMockIngressList(nil)
	if err != nil {
		panic(err)
	}

	// mock cloud providers
	MockCloudProvider, err = GetMockCloudProvider(nil)
	if err != nil {
		panic(err)
	}
	MockCloudProviderList, err = GetMockCloudProviderList(nil)
	if err != nil {
		panic(err)
	}

	// mock cloud provider regions/cloud regions
	MockCloudProviderRegion, err = GetMockCloudProviderRegion(nil)
	if err != nil {
		panic(err)
	}
	MockCloudProviderRegionList, err = GetMockCloudProviderRegionList(nil)
	if err != nil {
		panic(err)
	}

	// mock cluster status
	MockClusterStatus, err = GetMockClusterStatus(nil)
	MockClusterAddonInstallation, err = GetMockClusterAddonInstallation(nil)
	if err != nil {
		panic(err)
	}
	MockClusterAddonInstallationList, err = GetMockClusterAddonInstallationList(nil)
	if err != nil {
		panic(err)
	}
	MockCluster, err = GetMockCluster(nil)
	if err != nil {
		panic(err)
	}

	// Mock machine pools
	MockMachinePoolList, err = GetMachinePoolList(nil)
	if err != nil {
		panic(err)
	}
	MockMachinePool, err = GetMockMachinePool(nil)
	if err != nil {
		panic(err)
	}
}

// GetMockSyncsetBuilder for emulated OCM server
func GetMockSyncsetBuilder(modifyFn func(b *clustersmgmtv1.SyncsetBuilder)) *clustersmgmtv1.SyncsetBuilder {
	builder := clustersmgmtv1.NewSyncset().
		ID(MockSyncsetID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/external_configuration/syncsets/%s", MockClusterID, MockSyncsetID))

	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockSyncset for emulated OCM server
func GetMockSyncset(syncsetBuilder *clustersmgmtv1.SyncsetBuilder) (*clustersmgmtv1.Syncset, error) {
	return syncsetBuilder.Build()
}

// GetMockIngressList for emulated OCM server
func GetMockIngressList(modifyFn func(l *v1.IngressList, err error)) (*clustersmgmtv1.IngressList, error) {
	list, err := clustersmgmtv1.NewIngressList().Items(
		clustersmgmtv1.NewIngress().ID(MockIngressID).DNSName(MockIngressDNS).Default(true).Listening(MockIngressListening).HREF(MockIngressHref)).Build()

	if modifyFn != nil {
		modifyFn(list, err)
	}
	return list, err
}

// GetMockCloudProviderBuilder for emulated OCM server
func GetMockCloudProviderBuilder(modifyFn func(builder *clustersmgmtv1.CloudProviderBuilder)) *clustersmgmtv1.CloudProviderBuilder {
	builder := clustersmgmtv1.NewCloudProvider().
		ID(MockCloudProviderID).
		Name(MockCloudProviderID).
		DisplayName(MockCloudProviderDisplayName).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/cloud_providers/%s", MockCloudProviderID))

	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockCloudProvider for emulated OCM server
func GetMockCloudProvider(modifyFn func(*clustersmgmtv1.CloudProvider, error)) (*clustersmgmtv1.CloudProvider, error) {
	cloudProvider, err := GetMockCloudProviderBuilder(nil).Build()
	if modifyFn != nil {
		modifyFn(cloudProvider, err)
	}
	return cloudProvider, err
}

// GetMockCloudProviderList for emulated OCM server
func GetMockCloudProviderList(modifyFn func(*clustersmgmtv1.CloudProviderList, error)) (*clustersmgmtv1.CloudProviderList, error) {
	list, err := clustersmgmtv1.NewCloudProviderList().
		Items(GetMockCloudProviderBuilder(nil)).
		Build()
	if modifyFn != nil {
		modifyFn(list, err)
	}
	return list, err
}

// GetMockCloudProviderRegionBuilder for emulated OCM server
func GetMockCloudProviderRegionBuilder(modifyFn func(*clustersmgmtv1.CloudRegionBuilder)) *clustersmgmtv1.CloudRegionBuilder {
	builder := clustersmgmtv1.NewCloudRegion().
		ID(MockCloudRegionID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/cloud_providers/%s/regions/%s", MockCloudProviderID, MockCloudRegionID)).
		DisplayName(MockCloudRegionDisplayName).
		CloudProvider(GetMockCloudProviderBuilder(nil)).
		Enabled(true).
		SupportsMultiAZ(true)

	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockCloudProviderRegion for emulated OCM server
func GetMockCloudProviderRegion(modifyFn func(*clustersmgmtv1.CloudRegion, error)) (*clustersmgmtv1.CloudRegion, error) {
	cloudRegion, err := GetMockCloudProviderRegionBuilder(nil).Build()
	if modifyFn != nil {
		modifyFn(cloudRegion, err)
	}
	return cloudRegion, err
}

// GetMockCloudProviderRegionList for emulated OCM server
func GetMockCloudProviderRegionList(modifyFn func(*clustersmgmtv1.CloudRegionList, error)) (*clustersmgmtv1.CloudRegionList, error) {
	list, err := clustersmgmtv1.NewCloudRegionList().Items(GetMockCloudProviderRegionBuilder(nil)).Build()
	if modifyFn != nil {
		modifyFn(list, err)
	}
	return list, err
}

// GetMockClusterStatus for emulated OCM server
func GetMockClusterStatus(modifyFn func(*clustersmgmtv1.ClusterStatus, error)) (*clustersmgmtv1.ClusterStatus, error) {
	status, err := clustersmgmtv1.NewClusterStatus().
		ID(MockClusterID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/status", MockClusterID)).
		State(MockClusterState).
		Description("").
		Build()
	if modifyFn != nil {
		modifyFn(status, err)
	}
	return status, err
}

// GetMockClusterAddonBuilder for emulated OCM server
func GetMockClusterAddonBuilder(modifyFn func(*clustersmgmtv1.AddOnBuilder)) *clustersmgmtv1.AddOnBuilder {
	builder := clustersmgmtv1.NewAddOn().
		ID(MockClusterAddonID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/addons/%s", MockClusterAddonID))
	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockClusterAddonInstallationBuilder for emulated OCM server
func GetMockClusterAddonInstallationBuilder(modifyFn func(*clustersmgmtv1.AddOnInstallationBuilder)) *clustersmgmtv1.AddOnInstallationBuilder {
	addonInstallation := clustersmgmtv1.NewAddOnInstallation().
		ID(MockClusterAddonID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/addons/managed-kafka", MockClusterID)).
		Addon(GetMockClusterAddonBuilder(nil)).
		State(MockClusterAddonState).
		StateDescription(MockClusterAddonDescription)
	if modifyFn != nil {
		modifyFn(addonInstallation)
	}
	return addonInstallation
}

// GetMockClusterAddonInstallation for emulated OCM server
func GetMockClusterAddonInstallation(modifyFn func(*clustersmgmtv1.AddOnInstallation, error)) (*clustersmgmtv1.AddOnInstallation, error) {
	addonInstall, err := GetMockClusterAddonInstallationBuilder(nil).Build()
	if modifyFn != nil {
		modifyFn(addonInstall, err)
	}
	return addonInstall, err
}

// GetMockClusterAddonInstallationList for emulated OCM server
func GetMockClusterAddonInstallationList(modifyFn func(*clustersmgmtv1.AddOnInstallationList, error)) (*clustersmgmtv1.AddOnInstallationList, error) {
	list, err := clustersmgmtv1.NewAddOnInstallationList().Items(GetMockClusterAddonInstallationBuilder(nil)).Build()
	if modifyFn != nil {
		modifyFn(list, err)
	}
	return list, err
}

// GetMockClusterNodesBuilder for emulated OCM server
func GetMockClusterNodesBuilder(modifyFn func(*clustersmgmtv1.ClusterNodesBuilder)) *clustersmgmtv1.ClusterNodesBuilder {
	builder := clustersmgmtv1.NewClusterNodes().
		Compute(constants.ClusterNodeScaleIncrement)
	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockClusterBuilder for emulated OCM server
func GetMockClusterBuilder(modifyFn func(*clustersmgmtv1.ClusterBuilder)) *clustersmgmtv1.ClusterBuilder {
	builder := clustersmgmtv1.NewCluster().
		ID(MockClusterID).
		ExternalID(MockClusterExternalID).
		State(MockClusterState).
		MultiAZ(MockMultiAZ).
		Nodes(GetMockClusterNodesBuilder(nil)).
		CloudProvider(GetMockCloudProviderBuilder(nil)).
		Region(GetMockCloudProviderRegionBuilder(nil)).
		Version(GetMockOpenshiftVersionBuilder(nil))
	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMockCluster for emulated OCM server
func GetMockCluster(modifyFn func(*clustersmgmtv1.Cluster, error)) (*clustersmgmtv1.Cluster, error) {
	cluster, err := GetMockClusterBuilder(nil).Build()
	if modifyFn != nil {
		modifyFn(cluster, err)
	}
	return cluster, err
}

// GetMockMachineBuilder for emulated OCM server
func GetMockMachineBuilder(modifyFn func(*clustersmgmtv1.MachinePoolBuilder)) *clustersmgmtv1.MachinePoolBuilder {
	builder := clustersmgmtv1.NewMachinePool().
		ID(MockMachinePoolID).
		HREF(fmt.Sprintf("/api/clusters_mgmt/v1/clusters/%s/machine_pools/%s", MockClusterID, MockMachinePoolID)).
		Replicas(MockMachinePoolReplicas).
		Cluster(GetMockClusterBuilder(nil))
	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}

// GetMachinePoolList for emulated OCM server
func GetMachinePoolList(modifyFn func(*clustersmgmtv1.MachinePoolList, error)) (*clustersmgmtv1.MachinePoolList, error) {
	list, err := clustersmgmtv1.NewMachinePoolList().Items(GetMockMachineBuilder(nil)).Build()
	if modifyFn != nil {
		modifyFn(list, err)
	}
	return list, err
}

// GetMockMachinePool for emulated OCM server
func GetMockMachinePool(modifyFn func(*clustersmgmtv1.MachinePool, error)) (*clustersmgmtv1.MachinePool, error) {
	machinePool, err := GetMockMachineBuilder(nil).Build()
	if modifyFn != nil {
		modifyFn(machinePool, err)
	}
	return machinePool, err
}

// GetMockOpenshiftVersionBuilder for emulated OCM server
func GetMockOpenshiftVersionBuilder(modifyFn func(*clustersmgmtv1.VersionBuilder)) *clustersmgmtv1.VersionBuilder {
	builder := clustersmgmtv1.NewVersion().ID(MockOpenshiftVersion)
	if modifyFn != nil {
		modifyFn(builder)
	}
	return builder
}
