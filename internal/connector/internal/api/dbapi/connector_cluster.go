package dbapi

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"gorm.io/gorm"
)

type ConnectorClusterPhaseEnum string

const (
	// ConnectorClusterPhaseDisconnected - cluster status when first created
	ConnectorClusterPhaseDisconnected ConnectorClusterPhaseEnum = "disconnected"
	// ConnectorClusterPhaseReady - cluster status when it operational
	ConnectorClusterPhaseReady ConnectorClusterPhaseEnum = "ready"
	// ConnectorClusterPhaseDeleting - cluster status when in the process of being deleted
	ConnectorClusterPhaseDeleting ConnectorClusterPhaseEnum = "deleting"
)

var AgentRequestConnectorClusterStatus = []string{
	string(ConnectorClusterPhaseReady),
}

type ConnectorCluster struct {
	db.Model
	Owner          string
	OrganisationId string
	Name           string
	ClientId       string
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

	return json.Unmarshal([]byte(s), c)
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

	return json.Unmarshal([]byte(s), o)
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
