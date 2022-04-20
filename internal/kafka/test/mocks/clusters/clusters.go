package mocks

import (
	"encoding/json"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
)

var (
	testRegion        = "us-west-1"
	testProvider      = "aws"
	testCloudProvider = "aws"
	testMultiAZ       = true
	testStatus        = api.ClusterProvisioned
	testClusterID     = "123"
)

// build a test cluster
func BuildCluster(modifyFn func(cluster *api.Cluster)) *api.Cluster {
	clusterSpec, _ := json.Marshal(&types.ClusterSpec{})
	providerSpec, _ := json.Marshal("{}")
	cluster := &api.Cluster{
		Region:        testRegion,
		CloudProvider: testProvider,
		MultiAZ:       testMultiAZ,
		ClusterID:     testClusterID,
		ExternalID:    testClusterID,
		Status:        testStatus,
		ClusterSpec:   clusterSpec,
		ProviderSpec:  providerSpec,
	}
	if modifyFn != nil {
		modifyFn(cluster)
	}
	return cluster
}

func BuildClusterMap(modifyFn func(m []map[string]interface{})) []map[string]interface{} {
	clusterSpec := []uint8("{\"internal_id\":\"\",\"external_id\":\"\",\"status\":\"\",\"status_details\":\"\",\"additional_info\":null}")
	providerSpec := []uint8("\"{}\"")
	m := []map[string]interface{}{
		{
			"region":         testRegion,
			"cloud_provider": testCloudProvider,
			"multi_az":       testMultiAZ,
			"status":         testStatus,
			"cluster_id":     testClusterID,
			"external_id":    testClusterID,
			"created_at":     time.Time{},
			"deleted_at":     time.Time{},
			"updated_at":     time.Time{},
			"id":             "",
			"provider_type":  "",
			"cluster_spec":   &clusterSpec,
			"provider_spec":  &providerSpec,
		},
	}
	if modifyFn != nil {
		modifyFn(m)
	}
	return m
}
