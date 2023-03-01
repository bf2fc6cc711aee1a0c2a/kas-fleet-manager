package mocks

import (
	"encoding/json"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/clusters/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"gorm.io/gorm"
)

var (
	testClientID                = "some-client-id"
	testClientSecret            = "some-client-secret"
	testRegion                  = "us-west-1"
	testProvider                = "aws"
	testCloudProvider           = "aws"
	testMultiAZ                 = true
	testStatus                  = api.ClusterProvisioned
	TestClusterID               = "123"
	StrimziOperatorVersion      = "strimzi-cluster-operator.from-cluster"
	DefaultKafkaVersion         = "2.7.0"
	AvailableStrimziVersions, _ = json.Marshal([]api.StrimziVersion{
		{
			Version: StrimziOperatorVersion,
			Ready:   true,
			KafkaVersions: []api.KafkaVersion{
				{
					Version: "2.7.0",
				},
				{
					Version: "2.8.0",
				},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{
					Version: "2.7",
				},
				{
					Version: "2.8",
				},
			},
		},
	})

	NoAvailableStrimziVersions, _ = json.Marshal([]api.StrimziVersion{
		{
			Version: StrimziOperatorVersion,
			Ready:   false,
			KafkaVersions: []api.KafkaVersion{
				{
					Version: "2.7.0",
				},
				{
					Version: "2.8.0",
				},
			},
			KafkaIBPVersions: []api.KafkaIBPVersion{
				{
					Version: "2.7",
				},
				{
					Version: "2.8",
				},
			},
		},
	})
)

// build a test cluster
func BuildCluster(modifyFn func(cluster *api.Cluster)) *api.Cluster {
	clusterSpec, _ := json.Marshal(&types.ClusterSpec{})
	providerSpec, _ := json.Marshal("{}")
	cluster := &api.Cluster{
		Region:        testRegion,
		CloudProvider: testProvider,
		MultiAZ:       testMultiAZ,
		ClusterID:     TestClusterID,
		ExternalID:    TestClusterID,
		ClientID:      testClientID,
		ClientSecret:  testClientSecret,
		Status:        testStatus,
		ClusterSpec:   clusterSpec,
		ProviderSpec:  providerSpec,
		ProviderType:  api.ClusterProviderOCM,
		Meta: api.Meta{
			DeletedAt: gorm.DeletedAt{Valid: true},
		},
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
			"cluster_id":     TestClusterID,
			"external_id":    TestClusterID,
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
