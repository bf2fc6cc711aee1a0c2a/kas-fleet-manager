package api

import (
	"github.com/jinzhu/gorm"
)

type ConnectorClusterStatus = string

const (
	// ConnectorClusterStatusUnconnected - cluster status when first created
	ConnectorClusterStatusUnconnected ConnectorClusterStatus = "unconnected"
	// ConnectorClusterStatusReady- cluster status when it operational
	ConnectorClusterStatusReady ConnectorClusterStatus = "ready"
	// ConnectorClusterStatusFull- cluster status when it full and cannot accept anymore deployments
	ConnectorClusterStatusFull ConnectorClusterStatus = "full"
	// ConnectorClusterStatusFailed- cluster status when it has failed
	ConnectorClusterStatusFailed ConnectorClusterStatus = "failed"
	// ConnectorClusterStatusFailed- cluster status when it has been deleted
	ConnectorClusterStatusDeleted ConnectorClusterStatus = "deleted"
)

var AllConnectorClusterStatus = []ConnectorClusterStatus{
	ConnectorClusterStatusUnconnected,
	ConnectorClusterStatusReady,
	ConnectorClusterStatusFull,
	ConnectorClusterStatusFailed,
	ConnectorClusterStatusDeleted,
}

type ConnectorCluster struct {
	Meta
	Owner      string                 `json:"owner"`
	Name       string                 `json:"name"`
	AddonGroup string                 `json:"addon_group"`
	Status     ConnectorClusterStatus `json:"status"`
}

func (org *ConnectorCluster) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("ID", NewID())
}

type ConnectorClusterList []*ConnectorCluster
type ConnectorClusterIndex map[string]*ConnectorCluster

func (l ConnectorClusterList) Index() ConnectorClusterIndex {
	index := ConnectorClusterIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}
