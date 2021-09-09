/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Rack definition for configuring rack awareness for Dinosaur brokers.
type Rack struct {
	TopologyKey string `json:"topologyKey"`
}

// Pod template for the Zookeeper or Dinosaur pods.
type PodTemplate struct {
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
}

// Rule definition of a Prometheus JMX Exporter rule for filtering metrics
type Rule struct {
	Pattern string            `json:"pattern,omitempty"`
	Name    string            `json:"name,omitempty"`
	Type    string            `json:"type,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
}

// Metrics definition of the Prometheus JMX Exporter configuration
type Metrics struct {
	LowercaseOutputName bool   `json:"lowercaseOutputName,omitempty"`
	Rules               []Rule `json:"rules,omitempty"`
}

// CertAndKeySecretSource reference to the Secret which holds the certificate and private key pair.
// The certificate can optionally contain the whole chain.
type CertAndKeySecretSource struct {
	Certificate string `json:"certificate"`
	Key         string `json:"key"`
	SecretName  string `json:"secretName"`
}

// GenericSecretSource reference to the Secret which holds a secret.
type GenericSecretSource struct {
	Key        string `json:"key"`
	SecretName string `json:"secretName"`
}

// CertSecretSource reference to the Secret which holds a certificate.
type CertSecretSource struct {
	Certificate string `json:"certificate"`
	SecretName  string `json:"secretName"`
}

// StorageType type of possible storages.
type StorageType string

// StorageType constants.
const (
	Ephemeral       StorageType = "ephemeral"
	PersistentClaim StorageType = "persistent-claim"
	Jbod            StorageType = "jbod"
)

// EphemeralStorage storage of ephemeral type.
type EphemeralStorage struct {
	SizeLimit string `json:"sizeLimit,omitempty"`
}

// PersistentClaimStorageOverride overrides for individual brokers.
type PersistentClaimStorageOverride struct {
	Class  string `json:"class"`
	Broker int    `json:"broker"`
}

// PersistentClaimStorage storage of persistent-claim type.
type PersistentClaimStorage struct {
	Size        string                           `json:"size,omitempty"`
	Selector    map[string]string                `json:"selector,omitempty"`
	DeleteClaim *bool                            `json:"deleteClaim,omitempty"`
	Class       string                           `json:"class,omitempty"`
	Overrides   []PersistentClaimStorageOverride `json:"overrides,omitempty"`
}

// JbodVolume volume in a jbod storage.
type JbodVolume struct {
	Type StorageType `json:"type"`
	ID   *int        `json:"id,omitempty"`
	EphemeralStorage
	PersistentClaimStorage
}

// JbodStorage storage of jbod type.
// See https://strimzi.io/docs/operators/latest/using.html#jbod_configuration for more details.
type JbodStorage struct {
	Volumes []JbodVolume `json:"volumes,omitempty"`
}

// Storage configuration (disk). Cannot be updated.
// The type depends on the value of the Type property within the given object, which must be one of [ephemeral, persistent-claim, jbod].
type Storage struct {
	Type StorageType `json:"type"`
	ID   *int        `json:"id,omitempty"`
	EphemeralStorage
	PersistentClaimStorage
	JbodStorage
}

// DinosaurAuthorizationType type of possible authorization mechanisms.
type DinosaurAuthorizationType string

// DinosaurAuthorizationType constants.
const (
	Simple   DinosaurAuthorizationType = "simple"
	OPA      DinosaurAuthorizationType = "opa"
	Keycloak DinosaurAuthorizationType = "keycloak"
)

// DinosaurAuthorization authorization configuration for Dinosaur brokers.
// The type depends on the value of the Type property within the given object, which must be one of [simple, opa, keycloak].
type DinosaurAuthorization struct {
	Type DinosaurAuthorizationType `json:"type"`
	DinosaurAuthorizationSimple
	DinosaurAuthorizationOPA
	DinosaurAuthorizationKeycloak
}

// DinosaurAuthorizationSimple authorization configuration for Simple
type DinosaurAuthorizationSimple struct{}

// DinosaurAuthorizationOPA authorization configuration for OPA
type DinosaurAuthorizationOPA struct{}

// DinosaurAuthorizationKeycloak authorization configuration for Keycloak
type DinosaurAuthorizationKeycloak struct {
	ClientID                       string             `json:"clientId,omitempty"`
	TokenEndpointURI               string             `json:"tokenEndpointUri,omitempty"`
	TLSTrustedCertificates         []CertSecretSource `json:"tlsTrustedCertificates,omitempty"`
	DisableTLSHostnameVerification bool               `json:"disableTlsHostnameVerification,omitempty"`
	DelegateToDinosaurAcls         bool               `json:"delegateToDinosaurAcls,omitempty"`
	SuperUsers                     []string           `json:"superUsers,omitempty"`
}

// DinosaurListenerAuthenticationType type of possible authentication mechanisms.
type DinosaurListenerAuthenticationType string

// DinosaurListenerAuthenticationType constants.
const (
	TLS         DinosaurListenerAuthenticationType = "tls"
	ScramSha512 DinosaurListenerAuthenticationType = "scram-sha-512"
	OAuth       DinosaurListenerAuthenticationType = "oauth"
)

// DinosaurListenerAuthentication authentication configuration for Dinosaur brokers.
// The type depends on the value of the Type property within the given object, which must be one of [tls, scram-sha-512, oauth].
type DinosaurListenerAuthentication struct {
	Type DinosaurListenerAuthenticationType `json:"type"`
	DinosaurListenerAuthenticationTLS
	DinosaurListenerAuthenticationScramSha512
	DinosaurListenerAuthenticationOAuth
}

// DinosaurListenerAuthenticationTLS authentication configuration for TLS
type DinosaurListenerAuthenticationTLS struct{}

// DinosaurListenerAuthenticationScramSha512 authentication configuration for SCRAM-SHA-512
type DinosaurListenerAuthenticationScramSha512 struct{}

// DinosaurListenerAuthenticationOAuth authentication configuration for OAuth
type DinosaurListenerAuthenticationOAuth struct {
	AccessTokenIsJwt               bool                `json:"accessTokenIsJwt,omitempty"`
	CheckAccessTokenType           bool                `json:"checkAccessTokenType,omitempty"`
	CheckIssuer                    bool                `json:"checkIssuer,omitempty"`
	ClientID                       string              `json:"clientId,omitempty"`
	ClientSecret                   GenericSecretSource `json:"clientSecret,omitempty"`
	CustomClaimCheck               string              `json:"customClaimCheck,omitempty"`
	DisableTLSHostnameVerification bool                `json:"disableTlsHostnameVerification,omitempty"`
	EnableECDSA                    bool                `json:"enableECDSA,omitempty"`
	FallbackUserNameClaim          string              `json:"fallbackUserNameClaim,omitempty"`
	FallbackUserNamePrefix         string              `json:"fallbackUserNamePrefix,omitempty"`
	IntrospectionEndpointURI       string              `json:"introspectionEndpointUri,omitempty"`
	JwksEndpointURI                string              `json:"jwksEndpointUri,omitempty"`
	JwksExpirySeconds              int                 `json:"jwksExpirySeconds,omitempty"`
	JwksRefreshSeconds             int                 `json:"jwksRefreshSeconds,omitempty"`
	TLSTrustedCertificates         []CertSecretSource  `json:"tlsTrustedCertificates,omitempty"`
	UserInfoEndpointURI            string              `json:"userInfoEndpointUri,omitempty"`
	UserNameClaim                  string              `json:"userNameClaim,omitempty"`
	ValidIssuerURI                 string              `json:"validIssuerUri,omitempty"`
	ValidTokenType                 string              `json:"validTokenType,omitempty"`
	EnablePlain                    bool                `json:"enablePlain,omitempty"`
	TokenEndpointURI               string              `json:"tokenEndpointUri,omitempty"`
}

// DinosaurListenerExternalType type of possible external listeners.
type DinosaurListenerExternalType string

// DinosaurListenerType type of possible listeners.
type DinosaurListenerType string

// DinosaurListenerType constants.
const (
	Internal     DinosaurListenerType = "internal"
	Route        DinosaurListenerType = "route"
	LoadBalancer DinosaurListenerType = "loadbalancer"
	NodePort     DinosaurListenerType = "nodeport"
	Ingress      DinosaurListenerType = "ingress"
)

// ExternalTrafficPolicy specifies whether the service routes external traffic to node-local or cluster-wide endpoints.
type ExternalTrafficPolicy string

// ZookeeperTemplate definition for the template of ZooKeeper cluster resources.
type ZookeeperTemplate struct {
	Pod *PodTemplate `json:"pod,omitempty"`
}

// DinosaurTemplate definition for the template of Dinosaur cluster resources.
type DinosaurTemplate struct {
	Pod *PodTemplate `json:"pod,omitempty"`
}

// ExternalTrafficPolicy constants.
const (
	Local   ExternalTrafficPolicy = "local"
	Cluster ExternalTrafficPolicy = "cluster"
)

// GenericDinosaurListenerConfigurationBootstrap defines bootstrap configuration for Dinosaur listeners.
type GenericDinosaurListenerConfigurationBootstrap struct {
	AlternativeNames []string          `json:"alternativeNames,omitempty"`
	Host             string            `json:"host,omitempty"`
	NodePort         int               `json:"nodePort,omitempty"`
	LoadBalancerIP   string            `json:"loadBalancerIP,omitempty"`
	Annotations      map[string]string `json:"annotations,omitempty"`
}

// GenericDinosaurListenerConfigurationBroker defines per-broker configuration for Dinosaur listeners.
type GenericDinosaurListenerConfigurationBroker struct {
	Broker         int               `json:"broker"`
	AdvertisedHost string            `json:"advertisedHost,omitempty"`
	AdvertisedPort int               `json:"advertisedPort,omitempty"`
	Host           string            `json:"host,omitempty"`
	NodePort       int               `json:"nodePort,omitempty"`
	LoadBalancerIP string            `json:"loadBalancerIP,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
}

// NodeAddressType defines which address type should be used as the node address.
type NodeAddressType string

// NodeAddressType constants.
const (
	ExternalDNS NodeAddressType = "ExternalDNS"
	ExternalIP  NodeAddressType = "ExternalIP"
	InternalDNS NodeAddressType = "InternalDNS"
	InternalIP  NodeAddressType = "InternalIP"
	Hostname    NodeAddressType = "Hostname"
)

// GenericDinosaurListenerConfiguration defines some generic configuration for Dinosaur listeners
type GenericDinosaurListenerConfiguration struct {
	BrokerCertChainAndKey        *CertAndKeySecretSource                        `json:"brokerCertChainAndKey,omitempty"`
	ExternalTrafficPolicy        ExternalTrafficPolicy                          `json:"externalTrafficPolicy,omitempty"`
	LoadBalancerSourceRanges     []string                                       `json:"loadBalancerSourceRanges,omitempty"`
	Bootstrap                    *GenericDinosaurListenerConfigurationBootstrap `json:"bootstrap,omitempty"`
	Brokers                      []GenericDinosaurListenerConfigurationBroker   `json:"brokers,omitempty"`
	Class                        string                                         `json:"class,omitempty"`
	PreferredNodePortAddressType NodeAddressType                                `json:"preferredNodePortAddressType,omitempty"`
	UseServiceDNSDomain          bool                                           `json:"useServiceDnsDomain,omitempty"`
}

// GenericDinosaurListener configures a generic listener of Dinosaur brokers.
type GenericDinosaurListener struct {
	Name               string                                `json:"name"`
	Port               int                                   `json:"port"`
	Type               DinosaurListenerType                  `json:"type"`
	TLS                bool                                  `json:"tls"`
	Authentication     *DinosaurListenerAuthentication       `json:"authentication,omitempty"`
	Configuration      *GenericDinosaurListenerConfiguration `json:"configuration,omitempty"`
	NetworkPolicyPeers []networkingv1.NetworkPolicyPeer      `json:"networkPolicyPeers,omitempty"`
}

// DinosaurClusterSpec configuration of the Dinosaur cluster.
type DinosaurClusterSpec struct {
	Replicas      int                          `json:"replicas"`
	Version       string                       `json:"version,omitempty"`
	Config        map[string]string            `json:"config,omitempty"`
	Storage       Storage                      `json:"storage"`
	Listeners     []GenericDinosaurListener    `json:"listeners"`
	Authorization *DinosaurAuthorization       `json:"authorization,omitempty"`
	Metrics       *Metrics                     `json:"metrics,omitempty"`
	Image         *string                      `json:"image,omitempty"`
	Resources     *corev1.ResourceRequirements `json:"resources,omitempty"`
	Template      *DinosaurTemplate            `json:"template,omitempty"`
	JvmOptions    *JvmOptionsSpec              `json:"jvmOptions,omitempty"`
	Rack          *Rack                        `json:"rack,omitempty"`
}

// JVM options passed to containers
type JvmOptionsSpec struct {
	Xms string `json:"-Xms"`
	Xmx string `json:"-Xmx"`
}

// ZookeeperClusterSpec configuration of the ZooKeeper cluster.
type ZookeeperClusterSpec struct {
	Replicas   int                          `json:"replicas"`
	Storage    Storage                      `json:"storage"`
	Metrics    *Metrics                     `json:"metrics,omitempty"`
	Resources  *corev1.ResourceRequirements `json:"resources,omitempty"`
	Template   *ZookeeperTemplate           `json:"template,omitempty"`
	JvmOptions *JvmOptionsSpec              `json:"jvmOptions,omitempty"`
}

// EntityTopicOperatorSpec configuration of the Topic Operator.
type EntityTopicOperatorSpec struct {
}

// EntityUserOperatorSpec configuration of the User Operator.
type EntityUserOperatorSpec struct {
}

// EntityOperatorSpec configuration of the Entity Operator.
type EntityOperatorSpec struct {
	TopicOperator EntityTopicOperatorSpec `json:"topicOperator,omitempty"`
	UserOperator  EntityUserOperatorSpec  `json:"userOperator,omitempty"`
}

// DinosaurExporterSpec configuration of the Dinosaur Exporter
type DinosaurExporterSpec struct {
	TopicRegex string `json:"topicRegex,omitempty"`
	GroupRegex string `json:"groupRegex,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DinosaurSpec defines the desired state of Dinosaur
type DinosaurSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Dinosaur         DinosaurClusterSpec   `json:"dinosaur"`
	Zookeeper        ZookeeperClusterSpec  `json:"zookeeper"`
	EntityOperator   *EntityOperatorSpec   `json:"entityOperator,omitempty"`
	DinosaurExporter *DinosaurExporterSpec `json:"dinosaurExporter,omitempty"`
}

// DinosaurStatus defines the observed state of Dinosaur
type DinosaurStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Dinosaur is the Schema for the dinosaurs API
type Dinosaur struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DinosaurSpec   `json:"spec,omitempty"`
	Status DinosaurStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DinosaurList contains a list of Dinosaur
type DinosaurList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dinosaur `json:"items"`
}
