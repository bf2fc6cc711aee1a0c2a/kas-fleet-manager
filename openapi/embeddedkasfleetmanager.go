package openapi

import (
	_ "embed"
)

//go:embed kas-fleet-manager.yaml
var kasFleetManagerOpenAPIContents []byte

func KASFleetManagerOpenAPIYAMLBytes() []byte {
	return kasFleetManagerOpenAPIContents
}
