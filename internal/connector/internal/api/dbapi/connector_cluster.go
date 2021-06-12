package dbapi

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

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
	Owner          string                 `json:"owner"`
	OrganisationId string                 `json:"organisation_id"`
	Name           string                 `json:"name"`
	Status         ConnectorClusterStatus `json:"status" gorm:"embedded;embeddedPrefix:status_"`
}

type ConnectorClusterStatus struct {
	Phase ConnectorClusterPhaseEnum `json:"phase,omitempty"`
	// the version of the agent
	Version    string        `json:"version,omitempty"`
	Conditions ConditionList `json:"conditions,omitempty" gorm:"type:text"`
	Operators  OperatorList  `json:"operators,omitempty" gorm:"type:text"`
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
	Type               string `json:"type,omitempty"`
	Reason             string `json:"reason,omitempty"`
	Message            string `json:"message,omitempty"`
	Status             string `json:"status,omitempty"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
}

type OperatorStatus struct {
	// the id of the operator
	Id string `json:"id,omitempty"`
	// the type of the operator
	Type string `json:"type,omitempty"`
	// the version of the operator
	Version string `json:"version,omitempty"`
	// the namespace to which the operator has been installed
	Namespace string `json:"namespace,omitempty"`
	// the status of the operator
	Status string `json:"status,omitempty"`
}

func (org *ConnectorCluster) BeforeCreate(tx *gorm.DB) error {
	org.ID = api.NewID()
	return nil
}

type ConnectorClusterList []ConnectorCluster
