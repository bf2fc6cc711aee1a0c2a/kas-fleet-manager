package dbapi

import "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

type ConnectorType struct {
	api.Meta
	Version     string
	Name        string
	Description string
	// A json schema that can be used to validate a connectors connector_spec field.
	JsonSchema map[string]interface{}

	Channels []string
	// URL to an icon of the connector.
	IconHref string
	// labels used to categorize the connector
	Labels []string
}

type ConnectorTypeList []*ConnectorType
type ConnectorTypeIndex map[string]*ConnectorType

func (l ConnectorTypeList) Index() ConnectorTypeIndex {
	index := ConnectorTypeIndex{}
	for _, o := range l {
		index[o.ID] = o
	}
	return index
}

type ConnectorShardMetadata struct {
	ID              int64
	ConnectorTypeId string
	Channel         string
	ShardMetadata   api.JSON `gorm:"type:jsonb"`
	LatestId        *int64
}
