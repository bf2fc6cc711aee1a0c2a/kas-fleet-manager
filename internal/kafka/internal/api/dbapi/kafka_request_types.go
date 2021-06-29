package dbapi

import (
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
	Version     string `json:"version"`
	// the quota service type for the kafka, e.g. ams, allow-list
	QuotaType string `json:"quota_type"`
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

func (kafkaRequest *KafkaRequest) BeforeCreate(scope *gorm.DB) error {
	// To allow the id set on the KafkaRequest object to be used. This is useful for testing purposes.
	id := kafkaRequest.ID
	if id == "" {
		kafkaRequest.ID = api.NewID()
	}
	return nil
}
