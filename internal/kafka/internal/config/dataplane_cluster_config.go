package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	userv1 "github.com/openshift/api/user/v1"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

type DataplaneClusterConfig struct {
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
	RawKubernetesConfig                   *clientcmdapi.Config
	StrimziOperatorOLMConfig              OperatorInstallationConfig `json:"strimzi_operator_olm_config"`
	KasFleetshardOperatorOLMConfig        OperatorInstallationConfig `json:"kas_fleetshard_operator_olm_config"`
}

type OperatorInstallationConfig struct {
	Namespace              string `json:"namespace"`
	IndexImage             string `json:"index_image"`
	CatalogSourceNamespace string `json:"catalog_source_namespace"`
	Package                string `json:"package"`
	SubscriptionChannel    string `json:"subscription_channel"`
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

func NewDataplaneClusterConfig() *DataplaneClusterConfig {
	return &DataplaneClusterConfig{
		OpenshiftVersion:                      "",
		ComputeMachineType:                    "m5.2xlarge",
		StrimziOperatorVersion:                "",
		ImagePullDockerConfigContent:          "",
		ImagePullDockerConfigFile:             "secrets/image-pull.dockerconfigjson",
		DataPlaneClusterConfigFile:            "config/dataplane-cluster-configuration.yaml",
		ReadOnlyUserListFile:                  "config/read-only-user-list.yaml",
		KafkaSREUsersFile:                     "config/kafka-sre-user-list.yaml",
		DataPlaneClusterScalingType:           ManualScaling,
		ClusterConfig:                         &ClusterConfig{},
		EnableReadyDataPlaneClustersReconcile: true,
		Kubeconfig:                            getDefaultKubeconfig(),
		StrimziOperatorOLMConfig: OperatorInstallationConfig{
			IndexImage:             "quay.io/osd-addons/managed-kafka:production-82b42db",
			CatalogSourceNamespace: "openshift-marketplace",
			Namespace:              constants.StrimziOperatorNamespace,
			SubscriptionChannel:    "alpha",
			Package:                "managed-kafka",
		},
		KasFleetshardOperatorOLMConfig: OperatorInstallationConfig{
			IndexImage:             "quay.io/osd-addons/kas-fleetshard-operator:production-82b42db",
			CatalogSourceNamespace: "openshift-marketplace",
			Namespace:              constants.KASFleetShardOperatorNamespace,
			SubscriptionChannel:    "alpha",
			Package:                "kas-fleetshard-operator",
		},
	}
}

//manual cluster configuration
type ManualCluster struct {
	Name                  string                  `yaml:"name"`
	ClusterId             string                  `yaml:"cluster_id"`
	CloudProvider         string                  `yaml:"cloud_provider"`
	Region                string                  `yaml:"region"`
	MultiAZ               bool                    `yaml:"multi_az"`
	Schedulable           bool                    `yaml:"schedulable"`
	KafkaInstanceLimit    int                     `yaml:"kafka_instance_limit"`
	Status                api.ClusterStatus       `yaml:"status"`
	ProviderType          api.ClusterProviderType `yaml:"provider_type"`
	ClusterDNS            string                  `yaml:"cluster_dns"`
	SupportedInstanceType string                  `yaml:"supported_instance_type"`
}

func (c *ManualCluster) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type t ManualCluster
	temp := t{
		Status:                api.ClusterProvisioning,
		ProviderType:          api.ClusterProviderOCM,
		ClusterDNS:            "",
		SupportedInstanceType: api.AllInstanceTypeSupport.String(), // by default support both instance type
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

	if c.SupportedInstanceType == "" {
		c.SupportedInstanceType = api.AllInstanceTypeSupport.String()
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

func (conf *ClusterConfig) GetCapacityForRegion(region string) int {
	var capacity = 0
	for _, cluster := range conf.clusterList {
		if cluster.Region == region {
			capacity += cluster.KafkaInstanceLimit
		}
	}
	return capacity
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

func (conf *ClusterConfig) GetClusterSupportedInstanceType(clusterId string) (string, bool) {
	manualCluster, exist := conf.clusterConfigMap[clusterId]
	return manualCluster.SupportedInstanceType, exist
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

func (c *DataplaneClusterConfig) IsDataPlaneManualScalingEnabled() bool {
	return c.DataPlaneClusterScalingType == ManualScaling
}

func (c *DataplaneClusterConfig) IsDataPlaneAutoScalingEnabled() bool {
	return c.DataPlaneClusterScalingType == AutoScaling
}

func (c *DataplaneClusterConfig) IsReadyDataPlaneClustersReconcileEnabled() bool {
	return c.EnableReadyDataPlaneClustersReconcile
}

func (c *DataplaneClusterConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.OpenshiftVersion, "cluster-openshift-version", c.OpenshiftVersion, "The version of openshift installed on the cluster. An empty string indicates that the latest stable version should be used")
	fs.StringVar(&c.ComputeMachineType, "cluster-compute-machine-type", c.ComputeMachineType, "The compute machine type")
	fs.StringVar(&c.StrimziOperatorVersion, "strimzi-operator-version", c.StrimziOperatorVersion, "The version of the Strimzi operator to install. If set, DesiredStrimziVersion in kafka_request will updated with this value. Otherwise value reported from kas-fleetshard operator will be used")
	fs.StringVar(&c.ImagePullDockerConfigFile, "image-pull-docker-config-file", c.ImagePullDockerConfigFile, "The file that contains the docker config content for pulling MK operator images on clusters")
	fs.StringVar(&c.DataPlaneClusterConfigFile, "dataplane-cluster-config-file", c.DataPlaneClusterConfigFile, "File contains properties for manually configuring OSD cluster.")
	fs.StringVar(&c.DataPlaneClusterScalingType, "dataplane-cluster-scaling-type", c.DataPlaneClusterScalingType, "Set to use cluster configuration to configure clusters. Its value should be either 'none' for no scaling, 'manual' or 'auto'.")
	fs.StringVar(&c.ReadOnlyUserListFile, "read-only-user-list-file", c.ReadOnlyUserListFile, "File contains a list of users with read-only permissions to data plane clusters")
	fs.StringVar(&c.KafkaSREUsersFile, "kafka-sre-user-list-file", c.KafkaSREUsersFile, "File contains a list of kafka-sre users with cluster-admin permissions to data plane clusters")
	fs.BoolVar(&c.EnableReadyDataPlaneClustersReconcile, "enable-ready-dataplane-clusters-reconcile", c.EnableReadyDataPlaneClustersReconcile, "Enables reconciliation for data plane clusters in the 'Ready' state")
	fs.StringVar(&c.Kubeconfig, "kubeconfig", c.Kubeconfig, "A path to kubeconfig file used for communication with standalone clusters")
	fs.StringVar(&c.StrimziOperatorOLMConfig.CatalogSourceNamespace, "strimzi-operator-cs-namespace", c.StrimziOperatorOLMConfig.CatalogSourceNamespace, "Strimzi operator catalog source namespace.")
	fs.StringVar(&c.StrimziOperatorOLMConfig.IndexImage, "strimzi-operator-index-image", c.StrimziOperatorOLMConfig.IndexImage, "Strimzi operator index image")
	fs.StringVar(&c.StrimziOperatorOLMConfig.Namespace, "strimzi-operator-namespace", c.StrimziOperatorOLMConfig.Namespace, "Strimzi operator namespace")
	fs.StringVar(&c.StrimziOperatorOLMConfig.Package, "strimzi-operator-package", c.StrimziOperatorOLMConfig.Package, "Strimzi operator package")
	fs.StringVar(&c.StrimziOperatorOLMConfig.SubscriptionChannel, "strimzi-operator-sub-channel", c.StrimziOperatorOLMConfig.SubscriptionChannel, "Strimzi operator subscription channel")
	fs.StringVar(&c.KasFleetshardOperatorOLMConfig.CatalogSourceNamespace, "kas-fleetshard-operator-cs-namespace", c.KasFleetshardOperatorOLMConfig.CatalogSourceNamespace, "kas-fleetshard operator catalog source namespace.")
	fs.StringVar(&c.KasFleetshardOperatorOLMConfig.IndexImage, "kas-fleetshard-operator-index-image", c.KasFleetshardOperatorOLMConfig.IndexImage, "kas-fleetshard operator index image")
	fs.StringVar(&c.KasFleetshardOperatorOLMConfig.Namespace, "kas-fleetshard-operator-namespace", c.KasFleetshardOperatorOLMConfig.Namespace, "kas-fleetshard operator namespace")
	fs.StringVar(&c.KasFleetshardOperatorOLMConfig.Package, "kas-fleetshard-operator-package", c.KasFleetshardOperatorOLMConfig.Package, "kas-fleetshard operator package")
	fs.StringVar(&c.KasFleetshardOperatorOLMConfig.SubscriptionChannel, "kas-fleetshard-operator-sub-channel", c.KasFleetshardOperatorOLMConfig.SubscriptionChannel, "kas-fleetshard operator subscription channel")
}

func (c *DataplaneClusterConfig) ReadFiles() error {
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

		// read kubeconfig and validate standalone clusters are in kubeconfig context
		for _, cluster := range c.ClusterConfig.clusterList {
			if cluster.ProviderType != api.ClusterProviderStandalone {
				continue
			}
			// make sure we only read kubeconfig once
			if c.RawKubernetesConfig == nil {
				err = c.readKubeconfig()
				if err != nil {
					return err
				}
			}
			validationErr := validateClusterIsInKubeconfigContext(*c.RawKubernetesConfig, cluster)
			if validationErr != nil {
				return validationErr
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

func (c *DataplaneClusterConfig) readKubeconfig() error {
	_, err := os.Stat(c.Kubeconfig)
	if err != nil {
		if os.IsNotExist(err) {
			return errors.Errorf("The kubeconfig file %s does not exist", c.Kubeconfig)
		}
		return err
	}
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{Precedence: []string{c.Kubeconfig}},
		&clientcmd.ConfigOverrides{})
	rawConfig, err := config.RawConfig()
	if err != nil {
		return err
	}
	c.RawKubernetesConfig = &rawConfig
	return nil
}

func validateClusterIsInKubeconfigContext(rawConfig clientcmdapi.Config, cluster ManualCluster) error {
	if _, found := rawConfig.Contexts[cluster.Name]; found {
		return nil
	}
	return errors.Errorf("standalone cluster with ID: %s, and Name %s not in kubeconfig context", cluster.ClusterId, cluster.Name)
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

func (c *DataplaneClusterConfig) FindClusterNameByClusterId(clusterId string) string {
	for _, cluster := range c.ClusterConfig.clusterList {
		if cluster.ClusterId == clusterId {
			return cluster.Name
		}
	}
	return ""
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
