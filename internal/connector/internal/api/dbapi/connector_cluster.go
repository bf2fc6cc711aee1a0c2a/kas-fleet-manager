package dbapi

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"

	"gorm.io/gorm"
)

type ConnectorClusterPhaseEnum = string

const (
	// ConnectorClusterPhaseUnconnected - cluster status when first created
	ConnectorClusterPhaseUnconnected ConnectorClusterPhaseEnum = "unconnected"
	// ConnectorClusterPhaseReady- cluster status when it operational
	ConnectorClusterPhaseReady ConnectorClusterPhaseEnum = "ready"
	// ConnectorClusterPhaseFull- cluster status when it full and cannot accept anymore deployments
	ConnectorClusterPhaseFull ConnectorClusterPhaseEnum = "full"
	// ConnectorClusterPhaseFailed- cluster status when it has failed
	ConnectorClusterPhaseFailed ConnectorClusterPhaseEnum = "failed"
	// ConnectorClusterPhaseFailed- cluster status when it has been deleted
	ConnectorClusterPhaseDeleted ConnectorClusterPhaseEnum = "deleted"
)

var AllConnectorClusterStatus = []ConnectorClusterPhaseEnum{
	ConnectorClusterPhaseUnconnected,
	ConnectorClusterPhaseReady,
	ConnectorClusterPhaseFull,
	ConnectorClusterPhaseFailed,
	ConnectorClusterPhaseDeleted,
}

type ConnectorCluster struct {
	api.Meta
	Owner          string
	OrganisationId string
	Name           string
	Status         ConnectorClusterStatus `gorm:"embedded;embeddedPrefix:status_"`
}

type ConnectorClusterStatus struct {
	Phase ConnectorClusterPhaseEnum
	// the version of the agent
	Version    string
	Conditions ConditionList `gorm:"type:jsonb"`
	Operators  OperatorList  `gorm:"type:jsonb"`
}

type ConditionList []Condition

func (c *ConditionList) Scan(value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to unmarshal json value: %v", value)
	}

	result := ConditionList{}
	err := json.Unmarshal([]byte(s), &result)
	*c = ConditionList(result)
	return err
}

func (c ConditionList) Value() (driver.Value, error) {
	if len(c) == 0 {
		return nil, nil
	}
	return json.Marshal(c)
}

type OperatorList []OperatorStatus

func (o *OperatorList) Scan(value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to unmarshal json value: %v", value)
	}

	result := OperatorList{}
	err := json.Unmarshal([]byte(s), &result)
	*o = OperatorList(result)
	return err
}

func (o OperatorList) Value() (driver.Value, error) {
	if len(o) == 0 {
		return nil, nil
	}
	return json.Marshal(o)
}

type Condition struct {
	Type               string
	Reason             string
	Message            string
	Status             string
	LastTransitionTime string
}

type OperatorStatus struct {
	// the id of the operator
	Id string
	// the type of the operator
	Type string
	// the version of the operator
	Version string
	// the namespace to which the operator has been installed
	Namespace string
	// the status of the operator
	Status string
}

func (org *ConnectorCluster) BeforeCreate(tx *gorm.DB) error {
	org.ID = api.NewID()
	return nil
}

type ConnectorClusterList []ConnectorCluster
