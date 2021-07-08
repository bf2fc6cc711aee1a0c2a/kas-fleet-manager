package dbapi

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"gorm.io/gorm"
)

type KafkaRequest struct {
	api.Meta
	Region              string `json:"region"`
	ClusterID           string `json:"cluster_id" gorm:"index"`
	CloudProvider       string `json:"cloud_provider"`
	MultiAZ             bool   `json:"multi_az"`
	Name                string `json:"name" gorm:"index"`
	Status              string `json:"status" gorm:"index"`
	SsoClientID         string `json:"sso_client_id"`
	SsoClientSecret     string `json:"sso_client_secret"`
	SubscriptionId      string `json:"subscription_id"`
	Owner               string `json:"owner" gorm:"index"` // TODO: ocm owner?
	OwnerAccountId      string `json:"owner_account_id"`
	BootstrapServerHost string `json:"bootstrap_server_host"`
	OrganisationId      string `json:"organisation_id" gorm:"index"`
	FailedReason        string `json:"failed_reason"`
	// PlacementId field should be updated every time when a KafkaRequest is assigned to an OSD cluster (even if it's the same one again)
	PlacementId string `json:"placement_id"`

	DesiredKafkaVersion   string `json:"desired_kafka_version"`
	ActualKafkaVersion    string `json:"actual_kafka_version"`
	DesiredStrimziVersion string `json:"desired_strimzi_version"`
	ActualStrimziVersion  string `json:"actual_strimzi_version"`
	KafkaUpgrading        bool   `json:"kafka_upgrading"`
	StrimziUpgrading      bool   `json:"strimzi_upgrading"`
	// The type of kafka instance (eval or standard)
	InstanceType string `json:"instance_type"`
	// the quota service type for the kafka, e.g. ams, allow-list
	QuotaType string `json:"quota_type"`
	// Routes routes mapping for the kafka instance. It is an array and each item in the array contains a domain value and the corresponding route url
	Routes api.JSON `json:"routes"`
	// RoutesCreated if the routes mapping have been created in the DNS provider like Route53. Use a separate field to make it easier to query.
	RoutesCreated bool `json:"routes_created"`
}

type KafkaList []*KafkaRequest
type KafkaIndex map[string]*KafkaRequest

func (l KafkaList) Index() KafkaIndex {
	index := KafkaIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (k *KafkaRequest) BeforeCreate(scope *gorm.DB) error {
	// To allow the id set on the KafkaRequest object to be used. This is useful for testing purposes.
	id := k.ID
	if id == "" {
		k.ID = api.NewID()
	}
	return nil
}

func (k *KafkaRequest) GetRoutes() ([]DataPlaneKafkaRoute, error) {
	var routes []DataPlaneKafkaRoute
	if k.Routes == nil {
		return routes, nil
	}
	if err := json.Unmarshal(k.Routes, &routes); err != nil {
		return nil, err
	} else {
		return routes, nil
	}
}

func (k *KafkaRequest) GetDefaultRoutes(clusterDNS string, numberOfBrokers int) []DataPlaneKafkaRoute {
	clusterDNS = strings.Replace(clusterDNS, constants.DefaultIngressDnsNamePrefix, constants.ManagedKafkaIngressDnsNamePrefix, 1)
	clusterIngress := fmt.Sprintf("elb.%s", clusterDNS)

	routes := []DataPlaneKafkaRoute{
		{
			Domain: k.BootstrapServerHost,
			Router: clusterIngress,
		},
		{
			Domain: fmt.Sprintf("admin-server-%s", k.BootstrapServerHost),
			Router: clusterIngress,
		},
	}

	for i := 0; i < numberOfBrokers; i++ {
		r := DataPlaneKafkaRoute{
			Domain: fmt.Sprintf("broker-%d-%s", i, k.BootstrapServerHost),
			Router: clusterIngress,
		}
		routes = append(routes, r)
	}
	return routes
}

func (k *KafkaRequest) SetRoutes(routes []DataPlaneKafkaRoute) error {
	if r, err := json.Marshal(routes); err != nil {
		return err
	} else {
		k.Routes = r
		return nil
	}
}
