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
