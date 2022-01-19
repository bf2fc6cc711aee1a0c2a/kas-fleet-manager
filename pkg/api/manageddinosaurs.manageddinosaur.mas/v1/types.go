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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Capacity struct {
	IngressEgressThroughputPerSec string `json:"ingressEgressThroughputPerSec"`
	TotalMaxConnections           int    `json:"totalMaxConnections"`
	MaxDataRetentionSize          string `json:"maxDataRetentionSize"`
	MaxPartitions                 int    `json:"maxPartitions"`
	MaxDataRetentionPeriod        string `json:"maxDataRetentionPeriod"`
	MaxConnectionAttemptsPerSec   int    `json:"maxConnectionAttemptsPerSec"`
}

type VersionsSpec struct {
	Dinosaur         string `json:"dinosaur"`
	DinosaurOperator string `json:"dinosaurOperator"`
}

type ManagedDinosaurStatus struct {
	Conditions []metav1.Condition `json:"conditions"`
	Capacity   Capacity           `json:"capacity"`
	Versions   VersionsSpec       `json:"versions"`
}

// Spec
type OAuthSpec struct {
	ClientId               string  `json:"clientId"`
	ClientSecret           string  `json:"clientSecret"`
	TokenEndpointURI       string  `json:"tokenEndpointURI"`
	JwksEndpointURI        string  `json:"jwksEndpointURI"`
	ValidIssuerEndpointURI string  `json:"validIssuerEndpointURI"`
	UserNameClaim          string  `json:"userNameClaim"`
	TlsTrustedCertificate  *string `json:"tlsTrustedCertificate,omitempty"`
	CustomClaimCheck       string  `json:"customClaimCheck"`
	FallBackUserNameClaim  string  `json:"fallbackUserNameClaim"`
	MaximumSessionLifetime int64   `json:"maximumSessionLifetime"`
}

type TlsSpec struct {
	Cert string `json:"cert"`
	Key  string `json:"key"`
}

type EndpointSpec struct {
	Host string   `json:"host"`
	Tls  *TlsSpec `json:"tls,omitempty"`
}

type ServiceAccount struct {
	Name      string `json:"name"`
	Principal string `json:"principal"`
	Password  string `json:"password"`
}

type ManagedDinosaurSpec struct {
	Capacity        Capacity         `json:"capacity"`
	OAuth           OAuthSpec        `json:"oauth"`
	Endpoint        EndpointSpec     `json:"endpoint"`
	Versions        VersionsSpec     `json:"versions"`
	Deleted         bool             `json:"deleted"`
	Owners          []string         `json:"owners"`
	ServiceAccounts []ServiceAccount `json:"service_accounts"`
}

type ManagedDinosaur struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Id     string                `json:"id,omitempty"`
	Spec   ManagedDinosaurSpec   `json:"spec,omitempty"`
	Status ManagedDinosaurStatus `json:"status,omitempty"`
}
