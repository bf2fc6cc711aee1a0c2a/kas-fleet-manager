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
	SubscriptionId      string `json:"subscription_id"`
	Owner               string `json:"owner"` // TODO: ocm owner?
	BootstrapServerHost string `json:"bootstrap_server_host"`
	OrganisationId      string `json:"organisation_id"`
	FailedReason        string `json:"failed_reason"`
	// PlacementId field should be updated every time when a KafkaRequest is assigned to an OSD cluster (even if it's the same one again)
	PlacementId string `json:"placement_id"`
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
	return scope.SetColumn("ID", NewID())
}
