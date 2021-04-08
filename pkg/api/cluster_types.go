package api

import (
	"github.com/jinzhu/gorm"
)

type ClusterStatus string

func (k ClusterStatus) String() string {
	return string(k)
}

const (
	// The create cluster request has been recorder
	ClusterAccepted ClusterStatus = "cluster_accepted"
	// ClusterProvisioning the underlying ocm cluster is provisioning
	ClusterProvisioning ClusterStatus = "cluster_provisioning"
	// ClusterProvisioned the underlying ocm cluster is provisioned
	ClusterProvisioned ClusterStatus = "cluster_provisioned"
	// ClusterFailed the cluster failed to become ready
	ClusterFailed ClusterStatus = "failed"
	// ManagedKafkaAddonID the ID of the managed Kafka addon
	ManagedKafkaAddonID = "managed-kafka"
	// ClusterReady the cluster is terraformed and ready for kafka instances
	ClusterReady ClusterStatus = "ready"
	// ClusterDeprovisioning the cluster is empty and can be deprovisioned
	ClusterDeprovisioning ClusterStatus = "deprovisioning"
	// ClusterWaitingForKasFleetShardOperator the cluster is waiting for the KAS fleetshard operator to be ready
	ClusterWaitingForKasFleetShardOperator ClusterStatus = "waiting_for_kas_fleetshard_operator"
	// ClusterFull the cluster is full and cannot accept more Kafka clusters
	ClusterFull ClusterStatus = "full"
	// ClusterComputeNodeScalingUp the cluster is in the process of scaling up a compute node
	ClusterComputeNodeScalingUp ClusterStatus = "compute_node_scaling_up"

	// KasFleetshardOperatorAddonId the ID of the kas-fleetshard-operator addon
	KasFleetshardOperatorAddonId = "kas-fleetshard-operator"
)

// This represents the valid statuses of a OSD cluster
var StatusForValidCluster = []string{string(ClusterProvisioning), string(ClusterProvisioned), string(ClusterReady),
	string(ClusterAccepted), string(ClusterWaitingForKasFleetShardOperator), string(ClusterComputeNodeScalingUp)}

type Cluster struct {
	Meta
	CloudProvider      string        `json:"cloud_provider"`
	ClusterID          string        `json:"cluster_id"`
	ExternalID         string        `json:"external_id"`
	MultiAZ            bool          `json:"multi_az"`
	Region             string        `json:"region"`
	BYOC               bool          `json:"byoc"`
	Managed            bool          `json:"managed"`
	Status             ClusterStatus `json:"status"`
	IdentityProviderID string        `json:"identity_provider_id"`
}

type ClusterList []*Cluster
type ClusterIndex map[string]*Cluster

func (c ClusterList) Index() ClusterIndex {
	index := ClusterIndex{}
	for _, o := range c {
		index[o.ID] = o
	}
	return index
}

func (org *Cluster) BeforeCreate(scope *gorm.Scope) error {
	if org.Status == "" {
		if err := scope.SetColumn("status", ClusterAccepted); err != nil {
			return err
		}
	}

	id := org.ID
	if id == "" {
		id = NewID()
	}

	return scope.SetColumn("ID", id)
}
