package mocks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"

	"github.com/gorilla/mux"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	ocmErrors "gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

const (
	// MockConfigurableServerClusterID default mock cluster id used in the mock ocm server
	MockConfigurableServerClusterID = "2aad9fc1-c40e-471f-8d57-fdaecc7d3686"

	// MockConfigurableServerSyncsetID default mock syncset id used in the mock ocm server
	MockConfigurableServerSyncsetID = "ext-8a41f783-b5e4-4692-a7cd-c0b9c8eeede9"

	// MockConfigurableServerIngressID default mock ingress id used in the mock ocm server
	MockConfigurableServerIngressID = "s1h5"

	// MockConfigurableServerIngressDNS default mock ingress dns used in the mock ocm server
	MockConfigurableServerIngressDNS = "apps.ms-btq2d1h8d3b1.b3k3.s1.devshift.org"

	// MockConfigurableServerIngressHref default mock ingress HREF used in the mock ocm server
	MockConfigurableServerIngressHref = "/api/clusters_mgmt/v1/clusters/000/ingresses/i8y1"

	// MockConfigurableServerIngressListening default mock ingress listening used in the mock ocm server
	MockConfigurableServerIngressListening = "external"

	// EndpointPathClusters ocm clusters management service clusters endpoint
	EndpointPathClusters = "/api/clusters_mgmt/v1/clusters"
	// EndpointPathClusters ocm clusters management service clusters endpoint
	EndpointPathCluster = "/api/clusters_mgmt/v1/clusters/000"
	// EndpointPathSyncsets ocm clusters management service syncset endpoint
	EndpointPathSyncsets = "/api/clusters_mgmt/v1/clusters/000/external_configuration/syncsets"
	// EndpointPathIngresses ocm cluster management ingress endpoint
	EndpointPathIngresses = "/api/clusters_mgmt/v1/clusters/000/ingresses"
)

var (
	EndpointClusterGet         = Endpoint{EndpointPathCluster, http.MethodGet}
	EndpointClustersGet        = Endpoint{EndpointPathClusters, http.MethodGet}
	EndpointClustersPost       = Endpoint{EndpointPathClusters, http.MethodPost}
	EndpointClusterSyncsetPost = Endpoint{EndpointPathSyncsets, http.MethodPost}
	EndpointClusterIngressGet  = Endpoint{EndpointPathIngresses, http.MethodGet}
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

// Build builds the mock ocm api server using the endpoint handlers that have been set in the builder
func (b *MockConfigurableServerBuilder) Build() *httptest.Server {
	router := mux.NewRouter()

	// set up handlers from the builder
	for endpoint, handleFn := range b.handlerRegister {
		router.HandleFunc(endpoint.Path, handleFn).Methods(endpoint.Method)
	}
	server := httptest.NewServer(router)
	return server
}

// getDefaultHandlerRegister returns a set of default endpoints and handlers used in the mock ocm api server
func getDefaultHandlerRegister() (HandlerRegister, error) {
	mockCluster, err := clustersmgmtv1.NewCluster().ID(MockConfigurableServerClusterID).Build()
	if err != nil {
		return nil, err
	}
	mockSyncset, err := clustersmgmtv1.NewSyncset().ID(MockConfigurableServerSyncsetID).Build()
	if err != nil {
		return nil, err
	}
	mockIngressList, err := clustersmgmtv1.NewIngressList().Items(
		clustersmgmtv1.NewIngress().ID(MockConfigurableServerIngressID).DNSName(MockConfigurableServerIngressDNS).Default(true).Listening(MockConfigurableServerIngressListening).HREF(MockConfigurableServerIngressHref)).Build()
	if err != nil {
		return nil, err
	}
	// define a list of default endpoints and handlers in the mock ocm api server, when new endpoints are used in the
	// managed-services-api service, a default ocm response should also be added here
	return HandlerRegister{
		EndpointClusterGet:         buildMockRequestHandler(mockCluster, nil),
		EndpointClustersGet:        buildMockRequestHandler(mockCluster, nil),
		EndpointClustersPost:       buildMockRequestHandler(mockCluster, nil),
		EndpointClusterSyncsetPost: buildMockRequestHandler(mockSyncset, nil),
		EndpointClusterIngressGet:  buildMockRequestHandler(mockIngressList, nil),
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
	case *clustersmgmtv1.Cluster:
		return clustersmgmtv1.MarshalCluster(t.(*clustersmgmtv1.Cluster), w)
	case *clustersmgmtv1.Syncset:
		return clustersmgmtv1.MarshalSyncset(t.(*clustersmgmtv1.Syncset), w)
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
	case *ocmList:
		return json.NewEncoder(w).Encode(t)
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
