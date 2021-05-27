package api

import (
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type ClusterStatus string
type ClusterProviderType string

func (k ClusterStatus) String() string {
	return string(k)
}

func (k *ClusterStatus) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	switch s {
	case ClusterProvisioning.String():
		*k = ClusterProvisioning
	case ClusterProvisioned.String():
		*k = ClusterProvisioned
	case ClusterReady.String():
		*k = ClusterReady
	default:
		return errors.Errorf("invalid value %s", s)
	}
	return nil
}

func (p ClusterProviderType) String() string {
	return string(p)
}

func (p *ClusterProviderType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	switch s {
	case ClusterProviderOCM.String():
		*p = ClusterProviderOCM
	case ClusterProviderAwsEKS.String():
		*p = ClusterProviderAwsEKS
	case ClusterProviderStandalone.String():
		*p = ClusterProviderStandalone
	default:
		return errors.Errorf("invalid value %s", s)
	}
	return nil
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
	// ClusterReady the cluster is terraformed and ready for kafka instances
	ClusterReady ClusterStatus = "ready"
	// ClusterDeprovisioning the cluster is empty and can be deprovisioned
	ClusterDeprovisioning ClusterStatus = "deprovisioning"
	// ClusterCleanup the cluster external resources are being removed
	ClusterCleanup ClusterStatus = "cleanup"
	// ClusterWaitingForKasFleetShardOperator the cluster is waiting for the KAS fleetshard operator to be ready
	ClusterWaitingForKasFleetShardOperator ClusterStatus = "waiting_for_kas_fleetshard_operator"
	// ClusterFull the cluster is full and cannot accept more Kafka clusters
	ClusterFull ClusterStatus = "full"
	// ClusterComputeNodeScalingUp the cluster is in the process of scaling up a compute node
	ClusterComputeNodeScalingUp ClusterStatus = "compute_node_scaling_up"

	ClusterProviderOCM        ClusterProviderType = "ocm"
	ClusterProviderAwsEKS     ClusterProviderType = "aws_eks"
	ClusterProviderStandalone ClusterProviderType = "standalone"
)

// This represents the valid statuses of a OSD cluster
var StatusForValidCluster = []string{string(ClusterProvisioning), string(ClusterProvisioned), string(ClusterReady),
	string(ClusterAccepted), string(ClusterWaitingForKasFleetShardOperator), string(ClusterComputeNodeScalingUp)}

type Cluster struct {
	Meta
	CloudProvider      string        `json:"cloud_provider"`
	ClusterID          string        `json:"cluster_id" gorm:"uniqueIndex"`
	ExternalID         string        `json:"external_id"`
	MultiAZ            bool          `json:"multi_az"`
	Region             string        `json:"region"`
	Status             ClusterStatus `json:"status" gorm:"index"`
	IdentityProviderID string        `json:"identity_provider_id"`
	ClusterDNS         string        `json:"cluster_dns"`
	// the provider type for the cluster, e.g. OCM, AWS, GCP, Standalone etc
	ProviderType ClusterProviderType `json:"provider_type"`
	// store the provider-specific information that can be used to managed the openshift/k8s cluster
	ProviderSpec JSON `json:"provider_spec"`
	// store the specs of the openshift/k8s cluster which can be used to access the cluster
	ClusterSpec JSON `json:"cluster_spec"`
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

func (cluster *Cluster) BeforeCreate(tx *gorm.DB) error {
	if cluster.Status == "" {
		cluster.Status = ClusterAccepted
	}

	if cluster.ID == "" {
		cluster.ID = NewID()
	}

	return nil
}
