package api

import "github.com/jinzhu/gorm"

type Kafka struct {
	Meta
	Region        string `json:"region"`
	ClusterID     string `json:"clusterID"`
	CloudProvider string `json:"cloud_provider"`
	Name          string `json:"cluster_name"`
	Status        string `json:"status"`
}

type KafkaList []*Kafka
type KafkaIndex map[string]*Kafka

func (l KafkaList) Index() KafkaIndex {
	index := KafkaIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (org *Kafka) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("ID", NewID())
}
