package api

import "github.com/jinzhu/gorm"

type KafkaRequest struct {
	Meta
	Region              string `json:"region"`
	ClusterID           string `json:"clusterID"`
	CloudProvider       string `json:"cloud_provider"`
	MultiAZ             bool   `json:"multi_az"`
	Name                string `json:"name"`
	Status              string `json:"status"`
	Owner               string `json:"owner"` // TODO: ocm owner?
	BootstrapServerHost string `json:"bootstrap_server_host"`
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
