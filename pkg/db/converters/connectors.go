package converters

import (
	"encoding/json"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
)

func ConvertConnectors(request *api.Connector) (result []map[string]interface{}) {
	result = append(result, mustConvertUsingJsonTags(request))
	return
}

// ConvertConnectorsList converts a ConnectorsList to the response type expected by mocket
func ConvertConnectorsList(list api.ConnectorList) (result []map[string]interface{}) {
	for _, item := range list {
		result = append(result, mustConvertUsingJsonTags(item))
	}
	return
}

// mustConvertUsingJsonTags converts the request to a json map of fields to values.  panics
// if the request cannot be converted.
func mustConvertUsingJsonTags(request interface{}) map[string]interface{} {
	data, err := json.Marshal(request)
	if err != nil {
		panic("could not convert to json")
	}
	result := map[string]interface{}{}
	err = json.Unmarshal(data, &result)
	if err != nil {
		panic("could not convert from json")
	}
	return result
}
