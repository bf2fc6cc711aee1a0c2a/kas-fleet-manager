package config

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
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
	DataPlaneClusterScalingType           string         `json:"dataplane_cluster_scaling_type"`
	DataPlaneClusterConfigFile            string         `json:"dataplane_cluster_config_file"`
	ClusterConfig                         *ClusterConfig `json:"clusters_config"`
	EnableReadyDataPlaneClustersReconcile bool           `json:"enable_ready_dataplane_clusters_reconcile"`
}

type DynamicScalingConfig struct {
	Enabled bool `json:"enabled"`
}

const (
	// ManualScaling is the manual DataPlaneClusterScalingType via the configuration file
	ManualScaling string = "manual"
	// AutoScaling is the automatic DataPlaneClusterScalingType depending on cluster capacity as reported by the Agent Operator
	AutoScaling string = "auto"
	// NoScaling disables cluster scaling. This is useful in testing
	NoScaling string = "none"
)

func NewOSDClusterConfig() *OSDClusterConfig {
	return &OSDClusterConfig{
		OpenshiftVersion:                      "",
		ComputeMachineType:                    "m5.4xlarge",
		StrimziOperatorVersion:                "v0.21.3",
		ImagePullDockerConfigContent:          "",
		ImagePullDockerConfigFile:             "secrets/image-pull.dockerconfigjson",
		IngressControllerReplicas:             9,
		DataPlaneClusterConfigFile:            "config/dataplane-cluster-configuration.yaml",
		DataPlaneClusterScalingType:           ManualScaling,
		ClusterConfig:                         &ClusterConfig{},
		EnableReadyDataPlaneClustersReconcile: true,
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

func (s *OSDClusterConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&s.OpenshiftVersion, "cluster-openshift-version", s.OpenshiftVersion, "The version of openshift installed on the cluster. An empty string indicates that the latest stable version should be used")
	fs.StringVar(&s.ComputeMachineType, "cluster-compute-machine-type", s.ComputeMachineType, "The compute machine type")
	fs.StringVar(&s.StrimziOperatorVersion, "strimzi-operator-version", s.StrimziOperatorVersion, "The version of the Strimzi operator to install")
	fs.StringVar(&s.ImagePullDockerConfigFile, "image-pull-docker-config-file", s.ImagePullDockerConfigFile, "The file that contains the docker config content for pulling MK operator images on clusters")
	fs.IntVar(&s.IngressControllerReplicas, "ingress-controller-replicas", s.IngressControllerReplicas, "The number of replicas for the IngressController")
	fs.StringVar(&s.DataPlaneClusterConfigFile, "dataplane-cluster-config-file", s.DataPlaneClusterConfigFile, "File contains properties for manually configuring OSD cluster.")
	fs.StringVar(&s.DataPlaneClusterScalingType, "dataplane-cluster-scaling-type", s.DataPlaneClusterScalingType, "Set to use cluster configuration to configure clusters. Its value should be either 'none' for no scaling, 'manual' or 'auto'.")
	fs.BoolVar(&s.EnableReadyDataPlaneClustersReconcile, "enable-ready-dataplane-clusters-reconcile", s.EnableReadyDataPlaneClustersReconcile, "Enables reconciliation for data plane clusters in the 'Ready' state")
}

func (s *OSDClusterConfig) ReadFiles() error {
	if s.ImagePullDockerConfigContent == "" && s.ImagePullDockerConfigFile != "" {
		err := readFileValueString(s.ImagePullDockerConfigFile, &s.ImagePullDockerConfigContent)
		if err != nil {
			return err
		}
	}
	if s.IsDataPlaneManualScalingEnabled() {
		list, err := readDataPlaneClusterConfig(s.DataPlaneClusterConfigFile)
		if err == nil {
			s.ClusterConfig = NewClusterConfig(list)
		} else {
			return err
		}
	}

	return nil
}

func readDataPlaneClusterConfig(file string) (ClusterList, error) {
	fileContents, err := readFile(file)
	if err != nil {
		return nil, err
	}

	c := struct {
		ClusterList ClusterList `yaml:"clusters"`
	}{}

	if err = yaml.UnmarshalStrict([]byte(fileContents), &c); err != nil {
		return nil, err
	} else {
		return c.ClusterList, nil
	}
}
