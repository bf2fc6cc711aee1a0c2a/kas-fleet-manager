package config

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

type OSDClusterConfig struct {
	IngressControllerReplicas    int                   `json:"ingress_controller_replicas"`
	OpenshiftVersion             string                `json:"cluster_openshift_version"`
	ComputeMachineType           string                `json:"cluster_compute_machine_type"`
	StrimziOperatorVersion       string                `json:"strimzi_operator_version"`
	ImagePullDockerConfigContent string                `json:"image_pull_docker_config_content"`
	ImagePullDockerConfigFile    string                `json:"image_pull_docker_config_file"`
	DynamicScalingConfig         *DynamicScalingConfig `json:"dynamic_scaling_config"`
	//'manual' to use OSD Cluster configuration file, 'auto' to use dynamic scaling and it is TBD
	DataPlaneClusterScalingType string         `json:"dataplane_cluster_scaling_type"`
	DataPlaneClusterConfigFile  string         `json:"dataplane_cluster_config_file"`
	ClusterConfig               *ClusterConfig `json:"clusters_config"`
}

type DynamicScalingConfig struct {
	Enabled bool `json:"enabled"`
}

func NewOSDClusterConfig() *OSDClusterConfig {
	return &OSDClusterConfig{
		OpenshiftVersion:             "",
		ComputeMachineType:           "m5.4xlarge",
		StrimziOperatorVersion:       "v0.21.3",
		ImagePullDockerConfigContent: "",
		ImagePullDockerConfigFile:    "secrets/image-pull.dockerconfigjson",
		IngressControllerReplicas:    9,
		DynamicScalingConfig: &DynamicScalingConfig{
			Enabled: false,
		},
		DataPlaneClusterConfigFile:  "config/dataplane-cluster-configuration.yaml",
		DataPlaneClusterScalingType: "manual",
		ClusterConfig:               &ClusterConfig{},
	}
}

//manual cluster configuration
type ManualCluster struct {
	Name               string `yaml:"name"`
	ClusterId          string `yaml:"cluster_id"`
	CloudProvider      string `yaml:"cloud_provider"`
	Region             string `yaml:"region"`
	MultiAZ            bool   `yaml:"multi_az"`
	Schedulable        bool   `yaml:"schedulable"`
	KafkaInstanceLimit int    `yaml:"kafka_instance_limit"`
}

type ClusterList []ManualCluster

//
type ClusterConfig struct {
	ClusterList      ClusterList `yaml:"clusters"`
	ClusterConfigMap map[string]ManualCluster
}

func (conf *ClusterConfig) IsNumberOfKafkaWithinClusterLimit(clusterId string, count int) bool {
	if _, exist := conf.ClusterConfigMap[clusterId]; exist {
		limit := conf.ClusterConfigMap[clusterId].KafkaInstanceLimit
		return limit == -1 || count <= conf.ClusterConfigMap[clusterId].KafkaInstanceLimit
	}
	return true
}

func (conf *ClusterConfig) IsClusterSchedulable(clusterId string) bool {
	if _, exist := conf.ClusterConfigMap[clusterId]; exist {
		return conf.ClusterConfigMap[clusterId].Schedulable
	}
	return true
}

func (conf *ClusterConfig) ExcessClusters(clusterList map[string]api.Cluster) []string {
	var res []string

	for clusterId, v := range clusterList {
		if _, exist := conf.ClusterConfigMap[clusterId]; !exist {
			res = append(res, v.ClusterID)
		}
	}
	return res
}

func (conf *ClusterConfig) MissingClusters(clusterMap map[string]api.Cluster) []ManualCluster {
	var res []ManualCluster

	//ensure the order
	for _, p := range conf.ClusterList {
		if _, exists := clusterMap[p.ClusterId]; !exists {
			res = append(res, p)
		}
	}
	return res
}

func (c *OSDClusterConfig) IsDataPlaneScalingEnabled() bool {
	return c.DataPlaneClusterScalingType == "manual"
}

func (s *OSDClusterConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.OpenshiftVersion, "cluster-openshift-version", s.OpenshiftVersion, "The version of openshift installed on the cluster. An empty string indicates that the latest stable version should be used")
	fs.StringVar(&s.ComputeMachineType, "cluster-compute-machine-type", s.ComputeMachineType, "The compute machine type")
	fs.StringVar(&s.StrimziOperatorVersion, "strimzi-operator-version", s.StrimziOperatorVersion, "The version of the Strimzi operator to install")
	fs.StringVar(&s.ImagePullDockerConfigFile, "image-pull-docker-config-file", s.ImagePullDockerConfigFile, "The file that contains the docker config content for pulling MK operator images on clusters")
	fs.IntVar(&s.IngressControllerReplicas, "ingress-controller-replicas", s.IngressControllerReplicas, "The number of replicas for the IngressController")
	fs.BoolVar(&s.DynamicScalingConfig.Enabled, "enable-dynamic-scaling", s.DynamicScalingConfig.Enabled, "Enable Dynamic Scaling functionality")
	fs.StringVar(&s.DataPlaneClusterConfigFile, "dataplane-cluster-config-file", s.DataPlaneClusterConfigFile, "File contains properties for manually configuring OSD cluster.")
	fs.StringVar(&s.DataPlaneClusterScalingType, "dataplane-cluster-scaling-type", s.DataPlaneClusterScalingType, "Set to use cluster configuration to configure clusters")
}

func (s *OSDClusterConfig) ReadFiles() error {
	if s.ImagePullDockerConfigContent == "" && s.ImagePullDockerConfigFile != "" {
		err := readFileValueString(s.ImagePullDockerConfigFile, &s.ImagePullDockerConfigContent)
		if err != nil {
			return err
		}
	}
	if s.IsDataPlaneScalingEnabled() {
		err := readDataPlaneClusterConfig(s.DataPlaneClusterConfigFile, s.ClusterConfig)
		if err == nil {
			s.ClusterConfig.ClusterConfigMap = make(map[string]ManualCluster)
			//initialize helper map
			for _, p := range s.ClusterConfig.ClusterList {
				s.ClusterConfig.ClusterConfigMap[p.ClusterId] = p
			}
		}
	}

	return nil
}

func readDataPlaneClusterConfig(file string, val *ClusterConfig) error {
	fileContents, err := readFile(file)
	if err != nil {
		return err
	}

	return yaml.UnmarshalStrict([]byte(fileContents), val)
}
