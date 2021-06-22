package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	userv1 "github.com/openshift/api/user/v1"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type OSDClusterConfig struct {
	IngressControllerReplicas    int    `json:"ingress_controller_replicas"`
	OpenshiftVersion             string `json:"cluster_openshift_version"`
	ComputeMachineType           string `json:"cluster_compute_machine_type"`
	StrimziOperatorVersion       string `json:"strimzi_operator_version"`
	ImagePullDockerConfigContent string `json:"image_pull_docker_config_content"`
	ImagePullDockerConfigFile    string `json:"image_pull_docker_config_file"`
	// Possible values are:
	// 'manual' to use OSD Cluster configuration file,
	// 'auto' to use dynamic scaling
	// 'none' to disabled scaling all together, useful in testing
	DataPlaneClusterScalingType           string `json:"dataplane_cluster_scaling_type"`
	DataPlaneClusterConfigFile            string `json:"dataplane_cluster_config_file"`
	ReadOnlyUserList                      userv1.OptionalNames
	ReadOnlyUserListFile                  string
	KafkaSREUsers                         userv1.OptionalNames
	KafkaSREUsersFile                     string
	ClusterConfig                         *ClusterConfig `json:"clusters_config"`
	EnableReadyDataPlaneClustersReconcile bool           `json:"enable_ready_dataplane_clusters_reconcile"`
	Kubeconfig                            string         `json:"kubeconfig"`
	RawKubernetesConfig                   clientcmdapi.Config
}

const (
	// ManualScaling is the manual DataPlaneClusterScalingType via the configuration file
	ManualScaling string = "manual"
	// AutoScaling is the automatic DataPlaneClusterScalingType depending on cluster capacity as reported by the Agent Operator
	AutoScaling string = "auto"
	// NoScaling disables cluster scaling. This is useful in testing
	NoScaling string = "none"
)

func getDefaultKubeconfig() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".kube", "config")
}

func NewOSDClusterConfig() *OSDClusterConfig {
	return &OSDClusterConfig{
		OpenshiftVersion:                      "",
		ComputeMachineType:                    "m5.4xlarge",
		StrimziOperatorVersion:                "v0.21.3",
		ImagePullDockerConfigContent:          "",
		ImagePullDockerConfigFile:             "secrets/image-pull.dockerconfigjson",
		IngressControllerReplicas:             9,
		DataPlaneClusterConfigFile:            "config/dataplane-cluster-configuration.yaml",
		ReadOnlyUserListFile:                  "config/read-only-user-list.yaml",
		KafkaSREUsersFile:                     "config/kafka-sre-user-list.yaml",
		DataPlaneClusterScalingType:           ManualScaling,
		ClusterConfig:                         &ClusterConfig{},
		EnableReadyDataPlaneClustersReconcile: true,
		Kubeconfig:                            getDefaultKubeconfig(),
	}
}

//manual cluster configuration
type ManualCluster struct {
	Name               string                  `yaml:"name"`
	ClusterId          string                  `yaml:"cluster_id"`
	CloudProvider      string                  `yaml:"cloud_provider"`
	Region             string                  `yaml:"region"`
	MultiAZ            bool                    `yaml:"multi_az"`
	Schedulable        bool                    `yaml:"schedulable"`
	KafkaInstanceLimit int                     `yaml:"kafka_instance_limit"`
	Status             api.ClusterStatus       `yaml:"status"`
	ProviderType       api.ClusterProviderType `yaml:"provider_type"`
	ClusterDNS         string                  `yaml:"cluster_dns"`
}

func (c *ManualCluster) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type t ManualCluster
	temp := t{
		Status:       api.ClusterProvisioning,
		ProviderType: api.ClusterProviderOCM,
		ClusterDNS:   "",
	}
	err := unmarshal(&temp)
	if err != nil {
		return err
	}
	*c = ManualCluster(temp)
	if c.ClusterId == "" {
		return fmt.Errorf("cluster_id is empty")
	}

	if c.ProviderType == api.ClusterProviderStandalone {
		if c.ClusterDNS == "" {
			return errors.Errorf("Standalone cluster with id %s does not have the cluster dns field provided", c.ClusterId)
		}

		if c.Name == "" {
			return errors.Errorf("Standalone cluster with id %s does not have the name field provided", c.ClusterId)
		}

		c.Status = api.ClusterProvisioning // force to cluster provisioning status as we do not want to call StandaloneProvider to create the cluster.
	}

	return nil
}

type ClusterList []ManualCluster

type ClusterConfig struct {
	clusterList      ClusterList
	clusterConfigMap map[string]ManualCluster
}

func NewClusterConfig(clusters ClusterList) *ClusterConfig {
	clusterMap := make(map[string]ManualCluster)
	for _, c := range clusters {
		clusterMap[c.ClusterId] = c
	}
	return &ClusterConfig{
		clusterList:      clusters,
		clusterConfigMap: clusterMap,
	}
}

func (conf *ClusterConfig) IsNumberOfKafkaWithinClusterLimit(clusterId string, count int) bool {
	if _, exist := conf.clusterConfigMap[clusterId]; exist {
		limit := conf.clusterConfigMap[clusterId].KafkaInstanceLimit
		return limit == -1 || count <= limit
	}
	return true
}

func (conf *ClusterConfig) IsClusterSchedulable(clusterId string) bool {
	if _, exist := conf.clusterConfigMap[clusterId]; exist {
		return conf.clusterConfigMap[clusterId].Schedulable
	}
	return true
}

func (conf *ClusterConfig) ExcessClusters(clusterList map[string]api.Cluster) []string {
	var res []string

	for clusterId, v := range clusterList {
		if _, exist := conf.clusterConfigMap[clusterId]; !exist {
			res = append(res, v.ClusterID)
		}
	}
	return res
}

func (conf *ClusterConfig) MissingClusters(clusterMap map[string]api.Cluster) []ManualCluster {
	var res []ManualCluster

	//ensure the order
	for _, p := range conf.clusterList {
		if _, exists := clusterMap[p.ClusterId]; !exists {
			res = append(res, p)
		}
	}
	return res
}

func (c *OSDClusterConfig) IsDataPlaneManualScalingEnabled() bool {
	return c.DataPlaneClusterScalingType == ManualScaling
}

func (c *OSDClusterConfig) IsDataPlaneAutoScalingEnabled() bool {
	return c.DataPlaneClusterScalingType == AutoScaling
}

func (c *OSDClusterConfig) IsReadyDataPlaneClustersReconcileEnabled() bool {
	return c.EnableReadyDataPlaneClustersReconcile
}

func (c *OSDClusterConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.OpenshiftVersion, "cluster-openshift-version", c.OpenshiftVersion, "The version of openshift installed on the cluster. An empty string indicates that the latest stable version should be used")
	fs.StringVar(&c.ComputeMachineType, "cluster-compute-machine-type", c.ComputeMachineType, "The compute machine type")
	fs.StringVar(&c.StrimziOperatorVersion, "strimzi-operator-version", c.StrimziOperatorVersion, "The version of the Strimzi operator to install")
	fs.StringVar(&c.ImagePullDockerConfigFile, "image-pull-docker-config-file", c.ImagePullDockerConfigFile, "The file that contains the docker config content for pulling MK operator images on clusters")
	fs.IntVar(&c.IngressControllerReplicas, "ingress-controller-replicas", c.IngressControllerReplicas, "The number of replicas for the IngressController")
	fs.StringVar(&c.DataPlaneClusterConfigFile, "dataplane-cluster-config-file", c.DataPlaneClusterConfigFile, "File contains properties for manually configuring OSD cluster.")
	fs.StringVar(&c.DataPlaneClusterScalingType, "dataplane-cluster-scaling-type", c.DataPlaneClusterScalingType, "Set to use cluster configuration to configure clusters. Its value should be either 'none' for no scaling, 'manual' or 'auto'.")
	fs.StringVar(&c.ReadOnlyUserListFile, "read-only-user-list-file", c.ReadOnlyUserListFile, "File contains a list of users with read-only permissions to data plane clusters")
	fs.StringVar(&c.KafkaSREUsersFile, "kafka-sre-user-list-file", c.KafkaSREUsersFile, "File contains a list of kafka-sre users with cluster-admin permissions to data plane clusters")
	fs.BoolVar(&c.EnableReadyDataPlaneClustersReconcile, "enable-ready-dataplane-clusters-reconcile", c.EnableReadyDataPlaneClustersReconcile, "Enables reconciliation for data plane clusters in the 'Ready' state")
	fs.StringVar(&c.Kubeconfig, "kubeconfig", c.Kubeconfig, "A path to kubeconfig file used for communication with standalone clusters")
}

func (c *OSDClusterConfig) ReadFiles() error {
	if c.ImagePullDockerConfigContent == "" && c.ImagePullDockerConfigFile != "" {
		err := shared.ReadFileValueString(c.ImagePullDockerConfigFile, &c.ImagePullDockerConfigContent)
		if err != nil {
			return err
		}
	}

	if c.IsDataPlaneManualScalingEnabled() {
		list, err := readDataPlaneClusterConfig(c.DataPlaneClusterConfigFile)
		if err == nil {
			c.ClusterConfig = NewClusterConfig(list)
		} else {
			return err
		}
		for _, cluster := range c.ClusterConfig.clusterList {
			if cluster.ProviderType == api.ClusterProviderStandalone {
				config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
					&clientcmd.ClientConfigLoadingRules{Precedence: []string{c.Kubeconfig}},
					&clientcmd.ConfigOverrides{})
				c.RawKubernetesConfig, err = config.RawConfig()
				if err != nil {
					return err
				}
				break
			}
		}
	}

	err := readOnlyUserListFile(c.ReadOnlyUserListFile, &c.ReadOnlyUserList)
	if err != nil {
		return err
	}

	err = readKafkaSREUserFile(c.KafkaSREUsersFile, &c.KafkaSREUsers)
	if err != nil {
		return err
	}

	return nil
}

func readDataPlaneClusterConfig(file string) (ClusterList, error) {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return nil, err
	}

	c := struct {
		ClusterList ClusterList `yaml:"clusters"`
	}{}

	if err = yaml.Unmarshal([]byte(fileContents), &c); err != nil {
		return nil, err
	} else {
		return c.ClusterList, nil
	}
}

// Read the read-only users in the file into the read-only user list config
func readOnlyUserListFile(file string, val *userv1.OptionalNames) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}

// Read the kafka-sre users from the file into the kafka-sre user list config
func readKafkaSREUserFile(file string, val *userv1.OptionalNames) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}
