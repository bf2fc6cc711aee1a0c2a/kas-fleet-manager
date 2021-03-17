package presenters

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
)

func ConvertConnector(from openapi.Connector) (*api.Connector, *errors.ServiceError) {

	spec, err := json.Marshal(from.ConnectorSpec)
	if err != nil {
		return nil, errors.BadRequest("invalid connector spec: %v", err)
	}

	return &api.Connector{
		Meta: api.Meta{
			ID: from.Id,
		},
		ConnectorTypeId: from.ConnectorTypeId,
		ConnectorSpec:   spec,
		TargetKind:      from.DeploymentLocation.Kind,
		AddonClusterId:  from.DeploymentLocation.ClusterId,
		Region:          from.DeploymentLocation.Region,
		CloudProvider:   from.DeploymentLocation.CloudProvider,
		MultiAZ:         from.DeploymentLocation.MultiAz,
		Name:            from.Metadata.Name,
		Status:          from.Status,
		Owner:           from.Metadata.Owner,
		KafkaID:         from.Metadata.KafkaId,
		Version:         from.Metadata.ResourceVersion,
	}, nil
}

func PresentConnector(from *api.Connector) (openapi.Connector, *errors.ServiceError) {
	spec := map[string]interface{}{}
	err := json.Unmarshal([]byte(from.ConnectorSpec), &spec)
	if err != nil {
		return openapi.Connector{}, errors.BadRequest("invalid connector spec: %v", err)
	}

	reference := PresentReference(from.ID, from)
	return openapi.Connector{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Metadata: openapi.ConnectorAllOfMetadata{
			Owner:           from.Owner,
			KafkaId:         from.KafkaID,
			Name:            from.Name,
			CreatedAt:       from.CreatedAt,
			UpdatedAt:       from.UpdatedAt,
			ResourceVersion: from.Version,
		},
		DeploymentLocation: openapi.ClusterTarget{
			Kind:          from.TargetKind,
			ClusterId:     from.AddonClusterId,
			CloudProvider: from.CloudProvider,
			Region:        from.Region,
			MultiAz:       from.MultiAZ,
		},
		ConnectorTypeId: from.ConnectorTypeId,
		ConnectorSpec:   spec,
		Status:          from.Status,
	}, nil
}

func ConvertConnectorDeployment(from openapi.ConnectorDeployment) (api.ConnectorDeployment, *errors.ServiceError) {

	conditions, err := json.Marshal(from.Status.Conditions)
	if err != nil {
		return api.ConnectorDeployment{}, errors.BadRequest("invalid status conditions: %v", err)
	}
	spec, err := ConvertConnectorDeploymentSpec(from.Spec)
	if err != nil {
		return api.ConnectorDeployment{}, errors.BadRequest("invalid spec: %v", err)
	}

	return api.ConnectorDeployment{
		Meta: api.Meta{
			ID: from.Id,
		},
		Version: from.Metadata.ResourceVersion,
		Spec:    spec,
		Status: api.ConnectorDeploymentStatus{
			Phase:      from.Status.Phase,
			Conditions: conditions,
		},
	}, nil
}

func PresentConnectorDeployment(from api.ConnectorDeployment) (openapi.ConnectorDeployment, *errors.ServiceError) {
	var conditions []openapi.MetaV1Condition
	if from.Status.Conditions != nil {
		err := json.Unmarshal([]byte(from.Status.Conditions), &conditions)
		if err != nil {
			return openapi.ConnectorDeployment{}, errors.BadRequest("invalid status conditions: %v", err)
		}
	}

	spec, err := PresentConnectorDeploymentSpec(from.Spec)
	if err != nil {
		return openapi.ConnectorDeployment{}, errors.BadRequest("invalid spec: %v", err)
	}

	reference := PresentReference(from.ID, from)
	return openapi.ConnectorDeployment{
		Id:   reference.Id,
		Kind: reference.Kind,
		Href: reference.Href,
		Metadata: openapi.ConnectorDeploymentAllOfMetadata{
			CreatedAt:       from.CreatedAt,
			UpdatedAt:       from.UpdatedAt,
			ResourceVersion: from.Version,
		},
		Spec: spec,
		Status: openapi.ConnectorDeploymentStatus{
			Phase:      from.Status.Phase,
			Conditions: conditions,
		},
	}, nil
}

func ConvertConnectorDeploymentSpec(from openapi.ConnectorDeploymentSpec) (api.ConnectorDeploymentSpec, *errors.ServiceError) {

	operatorIds, err := json.Marshal(from.OperatorIds)
	if err != nil {
		return api.ConnectorDeploymentSpec{}, errors.BadRequest("operator_ids: %v", err)
	}
	resources, err := json.Marshal(from.Resources)
	if err != nil {
		return api.ConnectorDeploymentSpec{}, errors.BadRequest("resources: %v", err)
	}
	statusExtractors, err := json.Marshal(from.StatusExtractors)
	if err != nil {
		return api.ConnectorDeploymentSpec{}, errors.BadRequest("status_extractors: %v", err)
	}

	return api.ConnectorDeploymentSpec{
		ConnectorId:      from.ConnectorId,
		OperatorsIds:     operatorIds,
		Resources:        resources,
		StatusExtractors: statusExtractors,
	}, nil
}

func PresentConnectorDeploymentSpec(from api.ConnectorDeploymentSpec) (openapi.ConnectorDeploymentSpec, *errors.ServiceError) {

	var operatorIds []string
	if from.OperatorsIds != nil {
		err := json.Unmarshal([]byte(from.OperatorsIds), &operatorIds)
		if err != nil {
			return openapi.ConnectorDeploymentSpec{}, errors.BadRequest("invalid spec operatorIds: %v", err)
		}
	}

	var resources []map[string]interface{}
	if from.Resources != nil {
		err := json.Unmarshal([]byte(from.Resources), &resources)
		if err != nil {
			return openapi.ConnectorDeploymentSpec{}, errors.BadRequest("invalid spec resources: %v", err)
		}
	}

	var statusExtractors []openapi.ConnectorDeploymentSpecStatusExtractors
	if from.StatusExtractors != nil {
		err := json.Unmarshal([]byte(from.StatusExtractors), &statusExtractors)
		if err != nil {
			return openapi.ConnectorDeploymentSpec{}, errors.BadRequest("invalid spec resources: %v", err)
		}
	}

	return openapi.ConnectorDeploymentSpec{
		ConnectorId:      from.ConnectorId,
		OperatorIds:      operatorIds,
		Resources:        resources,
		StatusExtractors: statusExtractors,
	}, nil
}

func ConvertStatusExtractors(in []openapi.ConnectorDeploymentSpecStatusExtractors) []api.ConnectorDeploymentSpecStatusExtractors {
	out := make([]api.ConnectorDeploymentSpecStatusExtractors, len(in))
	for i, v := range in {
		out[i] = api.ConnectorDeploymentSpecStatusExtractors{
			ApiVersion:    v.ApiVersion,
			Kind:          v.Kind,
			Name:          v.Name,
			JsonPath:      v.JsonPath,
			ConditionType: v.ConditionType,
		}
	}
	return out
}

func PresentStatusExtractors(in []api.ConnectorDeploymentSpecStatusExtractors) []openapi.ConnectorDeploymentSpecStatusExtractors {
	out := make([]openapi.ConnectorDeploymentSpecStatusExtractors, len(in))
	for i, v := range in {
		out[i] = openapi.ConnectorDeploymentSpecStatusExtractors{
			ApiVersion:    v.ApiVersion,
			Kind:          v.Kind,
			Name:          v.Name,
			JsonPath:      v.JsonPath,
			ConditionType: v.ConditionType,
		}
	}
	return out
}
