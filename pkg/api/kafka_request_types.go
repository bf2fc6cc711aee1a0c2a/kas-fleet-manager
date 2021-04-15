package api

import "github.com/jinzhu/gorm"

type KafkaRequest struct {
	Meta
	Region              string `json:"region"`
	ClusterID           string `json:"cluster_id"`
	CloudProvider       string `json:"cloud_provider"`
	MultiAZ             bool   `json:"multi_az"`
	Name                string `json:"name"`
	Status              string `json:"status"`
	SsoClientID         string `json:"sso_client_id"`
	SsoClientSecret     string `json:"sso_client_secret"`
	SubscriptionId      string `json:"subscription_id"`
	Owner               string `json:"owner"` // TODO: ocm owner?
	OwnerAccountId      string `json:"owner_account_id"`
	BootstrapServerHost string `json:"bootstrap_server_host"`
	OrganisationId      string `json:"organisation_id"`
	FailedReason        string `json:"failed_reason"`
	// PlacementId field should be updated every time when a KafkaRequest is assigned to an OSD cluster (even if it's the same one again)
	PlacementId string `json:"placement_id"`
	Version     string `json:"version"`
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

func (org *KafkaRequest) BeforeCreate(scope *gorm.Scope) error {
	// To allow the id set on the KafkaRequest object to be used. This is useful for testing purposes.
	id := org.ID
	if id == "" {
		id = NewID()
	}
	return scope.SetColumn("ID", id)
}
