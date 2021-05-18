package api

import "net/url"

type ConnectorType struct {
	Meta
	Version     string `json:"version"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	// A json schema that can be used to validate a connectors connector_spec field.
	JsonSchema   map[string]interface{} `json:"json_schema,omitempty"`
	ExtensionURL *url.URL               `json:"-"`

	Channels []string `json:"channels,omitempty"`
	// URL to an icon of the connector.
	IconHref string `json:"icon_href,omitempty"`
	// labels used to categorize the connector
	Labels []string `json:"labels,omitempty"`
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
