package mocks

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

var (
	observabilityAccessToken = "observability access token"
	observabilityChannel     = "observability channel"
	observabilityRepository  = "observability repository"
	observabilityTag         = "observability tag"
	versions                 = []string{"v1.0.0-1", "v1.0.0-2"}
	v1                       = "v1.0.0-1"
	v2                       = "v1.0.0-2"
	kafkaVersions            = []api.KafkaVersion{{Version: v1}, {Version: v2}}
	ibpVersions              = []api.KafkaIBPVersion{{Version: v1}, {Version: v2}}
)

func BuildValidDataPlaneClusterUpdateStatusRequest(modifyFn func(statusRequest *private.DataPlaneClusterUpdateStatusRequest)) *private.DataPlaneClusterUpdateStatusRequest {
	statusRequest := &private.DataPlaneClusterUpdateStatusRequest{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
	}
	if modifyFn != nil {
		modifyFn(statusRequest)
	}
	return statusRequest
}

func BuildDataPlaneClusterUpdateStatusRequestStrimzi(modifyFn func(versions *[]private.DataPlaneClusterUpdateStatusRequestStrimzi)) *[]private.DataPlaneClusterUpdateStatusRequestStrimzi {
	versions := &[]private.DataPlaneClusterUpdateStatusRequestStrimzi{
		{Version: "v1.0.0-0", Ready: true, KafkaVersions: versions, KafkaIbpVersions: versions},
		{Version: "v2.0.0-0", Ready: false},
		{Version: "v3.0.0-0", Ready: true},
	}
	if modifyFn != nil {
		modifyFn(versions)
	}
	return versions
}

func BuildApiStrimziVersions(modifyFn func(versions *[]api.StrimziVersion)) *[]api.StrimziVersion {
	versions := &[]api.StrimziVersion{
		{Version: "v1.0.0-0", Ready: true, KafkaVersions: kafkaVersions, KafkaIBPVersions: ibpVersions},
		{Version: "v2.0.0-0", Ready: false},
		{Version: "v3.0.0-0", Ready: true},
	}
	if modifyFn != nil {
		modifyFn(versions)
	}
	return versions
}

func BuildDataPlaneClusterConfig(modifyFn func(config *dbapi.DataPlaneClusterConfig)) *dbapi.DataPlaneClusterConfig {
	config := &dbapi.DataPlaneClusterConfig{
		Observability: dbapi.DataPlaneClusterConfigObservability{
			AccessToken: observabilityAccessToken,
			Channel:     observabilityChannel,
			Repository:  observabilityRepository,
			Tag:         observabilityTag,
		},
	}
	if modifyFn != nil {
		modifyFn(config)
	}
	return config
}

func BuildDataplaneClusterAgentConfig(modifyFn func(config private.DataplaneClusterAgentConfig)) private.DataplaneClusterAgentConfig {
	config := private.DataplaneClusterAgentConfig{
		Spec: private.DataplaneClusterAgentConfigSpec{
			Observability: private.DataplaneClusterAgentConfigSpecObservability{
				AccessToken: &observabilityAccessToken,
				Channel:     observabilityChannel,
				Repository:  observabilityRepository,
				Tag:         observabilityTag,
			},
		},
	}
	if modifyFn != nil {
		modifyFn(config)
	}
	return config
}

func BuildDataPlaneClusterUpdateStatusRequest(modifyFn func(request private.DataPlaneClusterUpdateStatusRequest)) private.DataPlaneClusterUpdateStatusRequest {
	config := private.DataPlaneClusterUpdateStatusRequest{
		Strimzi: []private.DataPlaneClusterUpdateStatusRequestStrimzi{
			{Version: "v1.0.0-0", Ready: true, KafkaVersions: versions, KafkaIbpVersions: versions},
			{Version: "v2.0.0-0", Ready: false},
			{Version: "v3.0.0-0", Ready: true},
		},
	}
	if modifyFn != nil {
		modifyFn(config)
	}
	return config
}
