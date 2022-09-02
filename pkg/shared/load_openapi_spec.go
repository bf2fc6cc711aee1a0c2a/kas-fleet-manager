package shared

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/ghodss/yaml"
)

func LoadOpenAPISpecFromYAML(openapiYAMLBytes []byte) (data []byte, err error) {
	data, err = yaml.YAMLToJSON(openapiYAMLBytes)
	if err != nil {
		err = errors.GeneralError("can't convert OpenAPI specification from from YAML to JSON")
		return
	}
	return
}
