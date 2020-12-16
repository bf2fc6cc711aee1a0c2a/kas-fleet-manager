package api

import (
	"github.com/jinzhu/gorm"
	"time"
)

type LeaderLease struct {
	Meta
	Leader    string
	LeaseType string
	Expires   *time.Time
}

type LeaderLeaseList []*LeaderLease
type LeaderLeaseIndex map[string]*LeaderLease

func (l LeaderLeaseList) Index() LeaderLeaseIndex {
	index := LeaderLeaseIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

func (org *LeaderLease) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("ID", NewID())
}
