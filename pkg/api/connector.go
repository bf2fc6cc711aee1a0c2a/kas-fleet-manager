package api

type ConnectorStatus = string

const (
	ConnectorStatusAssigning ConnectorStatus = "assigning" // set by kas-fleet-manager - user request
	ConnectorStatusAssigned  ConnectorStatus = "assigned"  // set by kas-fleet-manager - worker
	ConnectorStatusReady     ConnectorStatus = "ready"     // set by the agent
	ConnectorStatusFailed    ConnectorStatus = "failed"    // set by the agent
	ConnectorStatusDeleting  ConnectorStatus = "deleting"  // set by the kas-fleet-manager - user request
	ConnectorStatusDeleted   ConnectorStatus = "deleted"   // set by the agent
)

var AgentSetConnectorStatus = []ConnectorStatus{
	ConnectorStatusReady,
	ConnectorStatusFailed,
	ConnectorStatusDeleted,
}

type TargetKind = string

const (
	AddonTargetKind         TargetKind = "addon"
	CloudProviderTargetKind TargetKind = "cloud_provider"
)

var AllTargetKind = []TargetKind{
	AddonTargetKind,
	CloudProviderTargetKind,
}

type Connector struct {
	Meta
	ConnectorTypeId string          `json:"connector_type_id,omitempty"`
	ConnectorSpec   string          `json:"connector_spec"`
	Region          string          `json:"region"`
	ClusterID       string          `json:"cluster_id"`
	CloudProvider   string          `json:"cloud_provider"`
	MultiAZ         bool            `json:"multi_az"`
	Name            string          `json:"name"`
	Status          ConnectorStatus `json:"status"`
	Owner           string          `json:"owner"`
	OrganisationId  string          `json:"organisation_id"`
	KafkaID         string          `json:"kafka_id"`
	Version         int64           `json:"version"`
	TargetKind      TargetKind      `json:"target_kind"`
	AddonGroup      string          `json:"addon_group"`
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
