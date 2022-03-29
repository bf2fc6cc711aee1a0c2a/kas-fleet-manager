package api

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"

	kasfleetmanagererrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type ClusterStatus string
type ClusterProviderType string
type ClusterInstanceTypeSupport string

func (k ClusterStatus) String() string {
	return string(k)
}

func (k ClusterInstanceTypeSupport) String() string {
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

// CompareTo - Compare this status with the given status returning an int. The result will be 0 if k==k1, -1 if k < k1, and +1 if k > k1
func (k ClusterStatus) CompareTo(k1 ClusterStatus) int {
	ordinalK := ordinals[k.String()]
	ordinalK1 := ordinals[k1.String()]

	switch {
	case ordinalK == ordinalK1:
		return 0
	case ordinalK > ordinalK1:
		return 1
	default:
		return -1
	}
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

	DeveloperTypeSupport   ClusterInstanceTypeSupport = "developer"
	StandardTypeSupport    ClusterInstanceTypeSupport = "standard"
	AllInstanceTypeSupport ClusterInstanceTypeSupport = "standard,developer"
)

// ordinals - Used to decide if a status comes after or before a given state
var ordinals = map[string]int{
	ClusterAccepted.String():                        0,
	ClusterProvisioning.String():                    10,
	ClusterProvisioned.String():                     20,
	ClusterWaitingForKasFleetShardOperator.String(): 30,
	ClusterReady.String():                           40,
	ClusterComputeNodeScalingUp.String():            50,
	ClusterDeprovisioning.String():                  60,
	ClusterCleanup.String():                         70,
	ClusterFailed.String():                          80,
}

// This represents the valid statuses of a dataplane cluster
var StatusForValidCluster = []string{string(ClusterProvisioning), string(ClusterProvisioned), string(ClusterReady),
	string(ClusterAccepted), string(ClusterWaitingForKasFleetShardOperator), string(ClusterComputeNodeScalingUp)}

// ClusterDeletionStatuses are statuses of clusters under deletion
var ClusterDeletionStatuses = []string{ClusterCleanup.String(), ClusterDeprovisioning.String()}

type Cluster struct {
	Meta
	CloudProvider      string        `json:"cloud_provider"`
	ClusterID          string        `json:"cluster_id" gorm:"uniqueIndex"`
	ExternalID         string        `json:"external_id"`
	MultiAZ            bool          `json:"multi_az"`
	Region             string        `json:"region"`
	Status             ClusterStatus `json:"status" gorm:"index"`
	StatusDetails      string        `json:"status_details" gorm:"-"`
	IdentityProviderID string        `json:"identity_provider_id"`
	ClusterDNS         string        `json:"cluster_dns"`
	ClientID           string        `json:"client_id"`
	ClientSecret       string        `json:"client_secret"`
	// the provider type for the cluster, e.g. OCM, AWS, GCP, Standalone etc
	ProviderType ClusterProviderType `json:"provider_type"`
	// store the provider-specific information that can be used to managed the openshift/k8s cluster
	ProviderSpec JSON `json:"provider_spec"`
	// store the specs of the openshift/k8s cluster which can be used to access the cluster
	ClusterSpec JSON `json:"cluster_spec"`
	// List of available strimzi versions in the cluster. Content MUST be stored
	// with the versions sorted in ascending order as a JSON. See
	// StrimziVersionNumberPartRegex for details on the expected strimzi version
	// format. See the StrimziVersion data type for the format of JSON stored. Use the
	// `SetAvailableStrimziVersions` helper method to ensure the correct order is set.
	// Latest position in the list is considered the newest available version.
	AvailableStrimziVersions JSON `json:"available_strimzi_versions"`
	// SupportedInstanceType holds information on what kind of instances types can be provisioned on this cluster.
	// A cluster can support two kinds of instance types: 'developer', 'standard' or both in this case it will be a comma separated list of instance types e.g 'standard,developer'.
	SupportedInstanceType string `json:"supported_instance_type"`
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

	if cluster.SupportedInstanceType == "" {
		cluster.SupportedInstanceType = AllInstanceTypeSupport.String()
	}

	return nil
}

type StrimziVersion struct {
	Version          string            `json:"version"`
	Ready            bool              `json:"ready"`
	KafkaVersions    []KafkaVersion    `json:"kafkaVersions"`
	KafkaIBPVersions []KafkaIBPVersion `json:"kafkaIBPVersions"`
}

type KafkaVersion struct {
	Version string `json:"version"`
}

func (s *KafkaVersion) Compare(other KafkaVersion) (int, error) {
	return buildAwareSemanticVersioningCompare(s.Version, other.Version)
}

type KafkaIBPVersion struct {
	Version string `json:"version"`
}

func (s *KafkaIBPVersion) Compare(other KafkaIBPVersion) (int, error) {
	return buildAwareSemanticVersioningCompare(s.Version, other.Version)
}

// StrimziVersionNumberPartRegex contains the regular expression needed to
// extract the semver version number for a StrimziVersion. StrimziVersion
// follows the format of: prefix_string-X.Y.Z-B where X,Y,Z,B are numbers
var StrimziVersionNumberPartRegex = regexp.MustCompile(`\d+\.\d+\.\d+-\d+$`)

// Compare returns compares s.Version with other.Version comparing the version
// number suffix specified there using StrimziVersionNumberPartRegex to extract
// the version number. If s.Version is smaller than other.Version a -1 is returned.
// If s.Version is equal than other.Version 0 is returned. If s.Version is greater
// than other.Version 1 is returned. If there is an error during the comparison
// an error is returned
func (s *StrimziVersion) Compare(other StrimziVersion) (int, error) {
	v1VersionNumber := StrimziVersionNumberPartRegex.FindString(s.Version)
	if v1VersionNumber == "" {
		return 0, fmt.Errorf("'%s' does not follow expected Strimzi Version format", s.Version)
	}

	v2VersionNumber := StrimziVersionNumberPartRegex.FindString(other.Version)
	if v2VersionNumber == "" {
		return 0, fmt.Errorf("'%s' does not follow expected Strimzi Version format", s.Version)
	}

	return buildAwareSemanticVersioningCompare(v1VersionNumber, v2VersionNumber)
}

func CompareBuildAwareSemanticVersions(v1, v2 string) (int, error) {
	return buildAwareSemanticVersioningCompare(v1, v2)
}

func CompareSemanticVersionsMajorAndMinor(current, desired string) (int, error) {
	return checkIfMinorDowngrade(current, desired)
}

func (s *StrimziVersion) DeepCopy() *StrimziVersion {
	var res StrimziVersion = *s
	res.KafkaVersions = nil
	res.KafkaIBPVersions = nil

	if s.KafkaVersions != nil {
		kafkaVersionsCopy := make([]KafkaVersion, len(s.KafkaVersions))
		copy(kafkaVersionsCopy, s.KafkaVersions)
		res.KafkaVersions = kafkaVersionsCopy
	}
	if s.KafkaIBPVersions != nil {
		kafkaIBPVersionsCopy := make([]KafkaIBPVersion, len(s.KafkaIBPVersions))
		copy(kafkaIBPVersionsCopy, s.KafkaIBPVersions)
		res.KafkaIBPVersions = kafkaIBPVersionsCopy
	}

	return &res
}

// GetAvailableAndReadyStrimziVersions returns the cluster's list of available
// and ready versions or an error. An empty list is returned if there are no
// available and ready versions
func (cluster *Cluster) GetAvailableAndReadyStrimziVersions() ([]StrimziVersion, error) {
	strimziVersions, err := cluster.GetAvailableStrimziVersions()
	if err != nil {
		return nil, err
	}

	res := []StrimziVersion{}
	for _, val := range strimziVersions {
		if val.Ready {
			res = append(res, val)
		}
	}
	return res, nil
}

// GetAvailableStrimziVersions returns the cluster's list of available strimzi
// versions or an error. An empty list is returned if there are no versions.
// This returns the available versions in the cluster independently on whether
// they are ready or not. If you want to only get the available and ready
// versions use the GetAvailableAndReadyStrimziVersions method
func (cluster *Cluster) GetAvailableStrimziVersions() ([]StrimziVersion, error) {
	versions := []StrimziVersion{}
	if cluster.AvailableStrimziVersions == nil {
		return versions, nil
	}

	err := json.Unmarshal(cluster.AvailableStrimziVersions, &versions)
	if err != nil {
		return nil, err
	}

	return versions, nil
}

// StrimziVersionsDeepSort returns a sorted copy of the provided StrimziVersions
// in the versions slice. The following elements are sorted in ascending order:
// - The strimzi versions
// - For each strimzi version, their Kafka Versions
// - For each strimzi version, their Kafka IBP Versions
func StrimziVersionsDeepSort(versions []StrimziVersion) ([]StrimziVersion, error) {
	if versions == nil {
		return versions, nil
	}
	if len(versions) == 0 {
		return []StrimziVersion{}, nil
	}

	var versionsToSet []StrimziVersion
	for idx := range versions {
		version := &versions[idx]
		copiedStrimziVersion := version.DeepCopy()
		versionsToSet = append(versionsToSet, *copiedStrimziVersion)
	}

	var errors kasfleetmanagererrors.ErrorList

	sort.Slice(versionsToSet, func(i, j int) bool {
		res, err := versionsToSet[i].Compare(versionsToSet[j])
		if err != nil {
			errors = append(errors, err)
		}
		return res == -1
	})

	if errors != nil {
		return nil, errors
	}

	for idx := range versionsToSet {

		// Sort KafkaVersions
		sort.Slice(versionsToSet[idx].KafkaVersions, func(i, j int) bool {
			res, err := versionsToSet[idx].KafkaVersions[i].Compare(versionsToSet[idx].KafkaVersions[j])
			if err != nil {
				errors = append(errors, err)
			}
			return res == -1
		})

		if errors != nil {
			return nil, errors
		}

		// Sort KafkaIBPVersions
		sort.Slice(versionsToSet[idx].KafkaIBPVersions, func(i, j int) bool {
			res, err := versionsToSet[idx].KafkaIBPVersions[i].Compare(versionsToSet[idx].KafkaIBPVersions[j])
			if err != nil {
				errors = append(errors, err)
			}
			return res == -1
		})

		if errors != nil {
			return nil, errors
		}
	}

	return versionsToSet, nil
}

// SetAvailableStrimziVersions sets the cluster's list of available strimzi
// versions. The list of versions is always stored in version ascending order,
// with all versions deeply sorted (strimzi versions, kafka versions, kafka ibp
// versions ...). If availableStrimziVersions is nil an empty list is set. See
// StrimziVersionNumberPartRegex for details on the expected strimzi version
// format
func (cluster *Cluster) SetAvailableStrimziVersions(availableStrimziVersions []StrimziVersion) error {
	sortedVersions, err := StrimziVersionsDeepSort(availableStrimziVersions)
	if err != nil {
		return err
	}
	if sortedVersions == nil {
		sortedVersions = []StrimziVersion{}
	}

	if v, err := json.Marshal(sortedVersions); err != nil {
		return err
	} else {
		cluster.AvailableStrimziVersions = v
		return nil
	}
}
