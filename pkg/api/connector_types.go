package api

import "github.com/jinzhu/gorm"

type Connector struct {
	Meta
	ConnectorTypeId string `json:"connector_type_id,omitempty"`
	ConnectorSpec   string `json:"connector_spec"`
	Region          string `json:"region"`
	ClusterID       string `json:"cluster_id"`
	CloudProvider   string `json:"cloud_provider"`
	MultiAZ         bool   `json:"multi_az"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	Owner           string `json:"owner"`
	KafkaID         string `json:"kafka_id"`
}

func (org *Connector) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("ID", NewID())
}

type ConnectorList []*Connector
type ConnectorIndex map[string]*Connector

func (l ConnectorList) Index() ConnectorIndex {
	index := ConnectorIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}
