package api

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/jinzhu/gorm"
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
	Meta
	Owner          string                 `json:"owner"`
	OrganisationId string                 `json:"organisation_id"`
	Name           string                 `json:"name"`
	Status         ConnectorClusterStatus `json:"status" gorm:"embedded;embedded_prefix:status_"`
}

type ConnectorClusterStatus struct {
	Phase ConnectorClusterPhaseEnum `json:"phase,omitempty"`
	// the version of the agent
	Version    string        `json:"version,omitempty"`
	Conditions ConditionList `json:"conditions,omitempty"`
	Operators  OperatorList  `json:"operators,omitempty"`
}

type ConditionList []Condition

func (j *ConditionList) Scan(value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to unmarshal json value: %v", value)
	}

	result := ConditionList{}
	err := json.Unmarshal([]byte(s), &result)
	*j = ConditionList(result)
	return err
}

func (j ConditionList) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.Marshal(j)
}

type OperatorList []Operators

func (j *OperatorList) Scan(value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("failed to unmarshal json value: %v", value)
	}

	result := OperatorList{}
	err := json.Unmarshal([]byte(s), &result)
	*j = OperatorList(result)
	return err
}

func (j OperatorList) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.Marshal(j)
}

type Condition struct {
	Type               string `json:"type,omitempty"`
	Reason             string `json:"reason,omitempty"`
	Message            string `json:"message,omitempty"`
	Status             string `json:"status,omitempty"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
}

type Operators struct {
	// the id of the operator
	Id string `json:"id,omitempty"`
	// the version of the operator
	Version string `json:"version,omitempty"`
	// the namespace to which the operator has been installed
	Namespace string `json:"namespace,omitempty"`
	// the status of the operator
	Status string `json:"status,omitempty"`
}

func (org *ConnectorCluster) BeforeCreate(scope *gorm.Scope) error {
	return scope.SetColumn("ID", NewID())
}

type ConnectorClusterList []ConnectorCluster
