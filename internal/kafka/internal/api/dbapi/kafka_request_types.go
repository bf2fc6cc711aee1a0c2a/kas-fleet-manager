package dbapi

import (
	"encoding/json"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"gorm.io/gorm"
)

type KafkaRequest struct {
	api.Meta
	Region                           string `json:"region"`
	ClusterID                        string `json:"cluster_id" gorm:"index"`
	CloudProvider                    string `json:"cloud_provider"`
	MultiAZ                          bool   `json:"multi_az"`
	Name                             string `json:"name" gorm:"index"`
	Status                           string `json:"status" gorm:"index"`
	SsoClientID                      string `json:"sso_client_id"`
	SsoClientSecret                  string `json:"sso_client_secret"`
	CanaryServiceAccountClientID     string `json:"canary_service_account_client_id"`
	CanaryServiceAccountClientSecret string `json:"canary_service_account_client_secret"`
	SubscriptionId                   string `json:"subscription_id"`
	Owner                            string `json:"owner" gorm:"index"` // TODO: ocm owner?
	OwnerAccountId                   string `json:"owner_account_id"`
	BootstrapServerHost              string `json:"bootstrap_server_host"`
	OrganisationId                   string `json:"organisation_id" gorm:"index"`
	FailedReason                     string `json:"failed_reason"`
	// PlacementId field should be updated every time when a KafkaRequest is assigned to an OSD cluster (even if it's the same one again)
	PlacementId string `json:"placement_id"`

	DesiredKafkaVersion    string `json:"desired_kafka_version"`
	ActualKafkaVersion     string `json:"actual_kafka_version"`
	DesiredStrimziVersion  string `json:"desired_strimzi_version"`
	ActualStrimziVersion   string `json:"actual_strimzi_version"`
	DesiredKafkaIBPVersion string `json:"desired_kafka_ibp_version"`
	ActualKafkaIBPVersion  string `json:"actual_kafka_ibp_version"`
	KafkaUpgrading         bool   `json:"kafka_upgrading"`
	StrimziUpgrading       bool   `json:"strimzi_upgrading"`
	KafkaIBPUpgrading      bool   `json:"kafka_ibp_upgrading"`
	KafkaStorageSize       string `json:"kafka_storage_size"`
	// The type of kafka instance (developer or standard)
	InstanceType string `json:"instance_type"`
	// the quota service type for the kafka, e.g. ams, quota-management-list
	QuotaType string `json:"quota_type"`
	// Routes routes mapping for the kafka instance. It is an array and each item in the array contains a domain value and the corresponding route url
	Routes api.JSON `json:"routes"`
	// RoutesCreated if the routes mapping have been created in the DNS provider like Route53. Use a separate field to make it easier to query.
	RoutesCreated bool `json:"routes_created"`
	// Namespace is the namespace of the provisioned kafka instance.
	// We store this in the database to ensure that old kafkas whose namespace contained "owner-<kafka-id>" information will continue to work.
	Namespace               string `json:"namespace"`
	ReauthenticationEnabled bool   `json:"reauthentication_enabled"`
	RoutesCreationId        string `json:"routes_creation_id"`
	SizeId                  string `json:"size_id"`
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

func (k *KafkaRequest) SetRoutes(routes []DataPlaneKafkaRoute) error {
	if r, err := json.Marshal(routes); err != nil {
		return err
	} else {
		k.Routes = r
		return nil
	}
}
