package api

import (
	"github.com/jinzhu/gorm"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
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
	State         clustersmgmtv1.ClusterState
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
	return scope.SetColumn("ID", NewID())
}
