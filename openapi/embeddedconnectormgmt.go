package openapi

import (
	_ "embed"
)

//go:embed connector_mgmt.yaml
var connectorMgmtOpenAPIContents []byte

func ConnectorMgmtOpenAPIYAMLBytes() []byte {
	return connectorMgmtOpenAPIContents
}
