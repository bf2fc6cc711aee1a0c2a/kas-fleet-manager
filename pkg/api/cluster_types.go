package api

import (
	"github.com/jinzhu/gorm"
)

type Cluster struct {
	Meta
	CloudProvider string `json:"cloud_provider"`
	ClusterID     string
	ExternalID    string
	MultiAZ       bool   `json:"multi_az"`
	Region        string `json:"region"`
	BYOC          bool
	Managed       bool
}

type ClusterList []*Cluster
type ClusterIndex map[string]*Cluster

func (c ClusterList) Index() ClusterIndex {
	index := ClusterIndex{}
	for _, o := range c {
		index[o.ID] = o
	}
	return index
}

func (org *Cluster) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("ID", org.ClusterID)
}
