package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/pkg/errors"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	userv1 "github.com/openshift/api/user/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"

	k8sYaml "sigs.k8s.io/yaml"

	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

const (
	defaultAWSComputeMachineType = "m5.2xlarge"
	defaultGCPComputeMachineType = "custom-8-32768"
)

type DataplaneClusterConfig struct {
	OpenshiftVersion             string
	AWSComputeMachineType        string
	GCPComputeMachineType        string
	ImagePullDockerConfigContent string
	ImagePullDockerConfigFile    string
	// Possible values are:
	// 'manual' to use OSD Cluster configuration file,
	// 'auto' to use dynamic scaling
	// 'none' to disabled scaling all together, useful in testing
	DataPlaneClusterScalingType                   string
	DataPlaneClusterConfigFile                    string
	ReadOnlyUserList                              userv1.OptionalNames
	ReadOnlyUserListFile                          string
	KafkaSREUsers                                 userv1.OptionalNames
	KafkaSREUsersFile                             string
	ClusterConfig                                 *ClusterConfig
	EnableReadyDataPlaneClustersReconcile         bool
	EnableKafkaSreIdentityProviderConfiguration   bool
	EnableDynamicScaleUpManagerScaleUpTrigger     bool
	EnableDynamicScaleDownManagerScaleDownTrigger bool
	Kubeconfig                                    string
	RawKubernetesConfig                           *clientcmdapi.Config
	StrimziOperatorOLMConfig                      OperatorInstallationConfig
	KasFleetshardOperatorOLMConfig                OperatorInstallationConfig
	DynamicScalingConfig                          DynamicScalingConfig
	NodePrewarmingConfig                          NodePrewarmingConfig
}

type OperatorInstallationConfig struct {
	Namespace               string
	IndexImage              string
	Package                 string
	SubscriptionChannel     string
	SubscriptionConfig      *operatorsv1alpha1.SubscriptionConfig
	SubscriptionConfigFile  string
	SubscriptionStartingCSV string
}

const (
	// ManualScaling is the manual DataPlaneClusterScalingType via the configuration file
	ManualScaling string = "manual"
	// AutoScaling is the automatic DataPlaneClusterScalingType depending on cluster capacity as reported by the Agent Operator
	AutoScaling string = "auto"
	// NoScaling disables cluster scaling. This is useful in testing
	NoScaling string = "none"
)

// constants for operators installation through OpenShift Lifecycle Manager (OLM)
// in `standalone` cluster provider type
const (
	defaultStrimziOperatorOLMPackageName             = "kas-strimzi-bundle"
	defaultStrimziOperatorOLMSubscriptionChannelName = "alpha"
	defaultStrimziOperatorIndexImage                 = "quay.io/osd-addons/rhosak-strimzi-operator-bundle-index:v4.9-v0.1.5-2"
	defaultStrimziOperatorSubscriptionConfigFile     = "config/strimzi-operator-subscription-spec-config.yaml"

	defaultFleetShardOperatorOLMPackageName             = "kas-fleetshard-operator"
	defaultFleetShardOperatorOLMSubscriptionChannelName = "alpha"
	defaultFleetShardOperatorIndexImage                 = "quay.io/osd-addons/rhosak-fleetshard-operator-bundle-index:v4.9-v1.0.7-1"
	defaultFleetShardOperatorSubscriptionConfigFile     = "config/kas-fleetshard-operator-subscription-spec-config.yaml"
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
		OpenshiftVersion:                              "",
		AWSComputeMachineType:                         defaultAWSComputeMachineType,
		GCPComputeMachineType:                         defaultGCPComputeMachineType,
		ImagePullDockerConfigContent:                  "",
		ImagePullDockerConfigFile:                     "secrets/image-pull.dockerconfigjson",
		DataPlaneClusterConfigFile:                    "config/dataplane-cluster-configuration.yaml",
		ReadOnlyUserListFile:                          "config/read-only-user-list.yaml",
		KafkaSREUsersFile:                             "config/kafka-sre-user-list.yaml",
		DataPlaneClusterScalingType:                   ManualScaling,
		ClusterConfig:                                 &ClusterConfig{},
		EnableReadyDataPlaneClustersReconcile:         true,
		EnableKafkaSreIdentityProviderConfiguration:   true,
		EnableDynamicScaleUpManagerScaleUpTrigger:     true,
		EnableDynamicScaleDownManagerScaleDownTrigger: true,
		Kubeconfig: getDefaultKubeconfig(),
		StrimziOperatorOLMConfig: OperatorInstallationConfig{
			IndexImage:             defaultStrimziOperatorIndexImage,
			Namespace:              constants.StrimziOperatorNamespace,
			SubscriptionChannel:    defaultStrimziOperatorOLMSubscriptionChannelName,
			Package:                defaultStrimziOperatorOLMPackageName,
			SubscriptionConfigFile: defaultStrimziOperatorSubscriptionConfigFile,
			SubscriptionConfig:     nil,
		},
		KasFleetshardOperatorOLMConfig: OperatorInstallationConfig{
			IndexImage:             defaultFleetShardOperatorIndexImage,
			Namespace:              constants.KASFleetShardOperatorNamespace,
			SubscriptionChannel:    defaultFleetShardOperatorOLMSubscriptionChannelName,
			Package:                defaultFleetShardOperatorOLMPackageName,
			SubscriptionConfigFile: defaultFleetShardOperatorSubscriptionConfigFile,
			SubscriptionConfig:     nil,
		},
		DynamicScalingConfig: NewDynamicScalingConfig(),
		NodePrewarmingConfig: NewNodePrewarmingConfig(),
	}
}

// manual cluster configuration
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
		if cluster.Region == region && cluster.Schedulable {
			capacity += cluster.KafkaInstanceLimit
		}
	}
	return capacity
}

// GetCapacityForRegionAndInstanceType returns the total capacity for the specified region and instance type.
// Set isolatedClustersOnly to true if you want to get the capacity of clusters that only supports this instance type.
func (conf *ClusterConfig) GetCapacityForRegionAndInstanceType(region, instanceType string, isolatedClustersOnly bool) int {
	var capacity = 0
	for _, cluster := range conf.clusterList {
		if cluster.Region == region && cluster.Schedulable {
			if isolatedClustersOnly {
				if cluster.SupportedInstanceType == instanceType {
					capacity += cluster.KafkaInstanceLimit
				}
			} else {
				if strings.Contains(cluster.SupportedInstanceType, instanceType) {
					capacity += cluster.KafkaInstanceLimit
				}
			}
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

func (conf *ClusterConfig) GetManualClusters() []ManualCluster {
	return conf.clusterList
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

// DefaultComputeMachineType returns the default Compute Machine Type for the
// given `cloudProviderID`. If `cloudProviderID` is not a known cloud provider
// the empty string is returned.
func (c *DataplaneClusterConfig) DefaultComputeMachineType(cloudProviderID cloudproviders.CloudProviderID) string {
	switch cloudProviderID {
	case cloudproviders.AWS:
		return c.AWSComputeMachineType
	case cloudproviders.GCP:
		return c.GCPComputeMachineType
	default:
		return ""
	}
}

func (c *DataplaneClusterConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.OpenshiftVersion, "cluster-openshift-version", c.OpenshiftVersion, "The version of openshift installed on the cluster. An empty string indicates that the latest stable version should be used")
	fs.StringVar(&c.AWSComputeMachineType, "aws-cluster-compute-machine-type", c.AWSComputeMachineType, "The instance type of the AWS compute instances for Data Planes created in AWS")
	fs.StringVar(&c.GCPComputeMachineType, "gcp-cluster-compute-machine-type", c.GCPComputeMachineType, "The instance type of the GCP compute instances for Data Planes created in GCP")
	fs.StringVar(&c.ImagePullDockerConfigFile, "image-pull-docker-config-file", c.ImagePullDockerConfigFile, "The file that contains the docker config content for pulling MK operator images on clusters")
	fs.StringVar(&c.DataPlaneClusterConfigFile, "dataplane-cluster-config-file", c.DataPlaneClusterConfigFile, "File contains properties for manually configuring OSD cluster.")
	fs.StringVar(&c.DataPlaneClusterScalingType, "dataplane-cluster-scaling-type", c.DataPlaneClusterScalingType, "Set to use cluster configuration to configure clusters. Its value should be either 'none' for no scaling, 'manual' or 'auto'.")
	fs.StringVar(&c.ReadOnlyUserListFile, "read-only-user-list-file", c.ReadOnlyUserListFile, "File contains a list of users with read-only permissions to data plane clusters")
	fs.StringVar(&c.KafkaSREUsersFile, "kafka-sre-user-list-file", c.KafkaSREUsersFile, "File contains a list of kafka-sre users with cluster-admin permissions to data plane clusters")
	fs.BoolVar(&c.EnableReadyDataPlaneClustersReconcile, "enable-ready-dataplane-clusters-reconcile", c.EnableReadyDataPlaneClustersReconcile, "Enables reconciliation for data plane clusters in the 'Ready' state")
	fs.BoolVar(&c.EnableKafkaSreIdentityProviderConfiguration, "enable-kafka-sre-identity-provider-configuration", c.EnableKafkaSreIdentityProviderConfiguration, "Enable the configuration of Kafka_SRE identity provider on the data plane cluster")
	fs.StringVar(&c.Kubeconfig, "kubeconfig", c.Kubeconfig, "A path to kubeconfig file used for communication with standalone clusters")
	fs.StringVar(&c.StrimziOperatorOLMConfig.IndexImage, "strimzi-operator-index-image", c.StrimziOperatorOLMConfig.IndexImage, "Strimzi operator index image")
	fs.StringVar(&c.StrimziOperatorOLMConfig.Namespace, "strimzi-operator-namespace", c.StrimziOperatorOLMConfig.Namespace, "Strimzi operator namespace")
	fs.StringVar(&c.StrimziOperatorOLMConfig.Package, "strimzi-operator-package", c.StrimziOperatorOLMConfig.Package, "Strimzi operator package")
	fs.StringVar(&c.StrimziOperatorOLMConfig.SubscriptionStartingCSV, "strimzi-operator-starting-csv", c.StrimziOperatorOLMConfig.SubscriptionStartingCSV, "Strimzi operator subscription starting CSV")
	fs.StringVar(&c.StrimziOperatorOLMConfig.SubscriptionChannel, "strimzi-operator-sub-channel", c.StrimziOperatorOLMConfig.SubscriptionChannel, "Strimzi operator subscription channel")
	fs.StringVar(&c.StrimziOperatorOLMConfig.SubscriptionConfigFile, "strimzi-operator-subscription-config-file", c.StrimziOperatorOLMConfig.SubscriptionConfigFile, "Strimzi operator subscription config. This is applied for standalone clusters only. The configuration must be of type https://pkg.go.dev/github.com/operator-framework/api@v0.3.25/pkg/operators/v1alpha1?utm_source=gopls#SubscriptionConfig")
	fs.StringVar(&c.KasFleetshardOperatorOLMConfig.IndexImage, "kas-fleetshard-operator-index-image", c.KasFleetshardOperatorOLMConfig.IndexImage, "kas-fleetshard operator index image")
	fs.StringVar(&c.KasFleetshardOperatorOLMConfig.Namespace, "kas-fleetshard-operator-namespace", c.KasFleetshardOperatorOLMConfig.Namespace, "kas-fleetshard operator namespace")
	fs.StringVar(&c.KasFleetshardOperatorOLMConfig.Package, "kas-fleetshard-operator-package", c.KasFleetshardOperatorOLMConfig.Package, "kas-fleetshard operator package")
	fs.StringVar(&c.KasFleetshardOperatorOLMConfig.SubscriptionStartingCSV, "kas-fleetshard-operator-starting-csv", c.KasFleetshardOperatorOLMConfig.SubscriptionStartingCSV, "kas-fleetshard operator subscription starting CSV")
	fs.StringVar(&c.KasFleetshardOperatorOLMConfig.SubscriptionChannel, "kas-fleetshard-operator-sub-channel", c.KasFleetshardOperatorOLMConfig.SubscriptionChannel, "kas-fleetshard operator subscription channel")
	fs.StringVar(&c.KasFleetshardOperatorOLMConfig.SubscriptionConfigFile, "kas-fleetshard-operator-subscription-config-file", c.KasFleetshardOperatorOLMConfig.SubscriptionConfigFile, "kas-fleetshard operator subscription config. This is applied for standalone clusters only. The configuration must be of type https://pkg.go.dev/github.com/operator-framework/api@v0.3.25/pkg/operators/v1alpha1?utm_source=gopls#SubscriptionConfig")
	fs.StringVar(&c.DynamicScalingConfig.filePath, "dynamic-scaling-config-file", c.DynamicScalingConfig.filePath, "File path to a file containing the dynamic scaling configuration")
	fs.StringVar(&c.NodePrewarmingConfig.filePath, "node-prewarming-config-file", c.NodePrewarmingConfig.filePath, "File path to a file containing the node prewarming configuration")
}

func (c *DataplaneClusterConfig) Validate(env *environments.Env) error {

	var kafkaConfig *KafkaConfig
	env.MustResolve(&kafkaConfig)

	if c.IsDataPlaneAutoScalingEnabled() {
		err := c.DynamicScalingConfig.validate()
		if err != nil {
			return err
		}
	}

	return c.NodePrewarmingConfig.validate(kafkaConfig)
}

func (c *DataplaneClusterConfig) ReadFiles() error {
	if c.ImagePullDockerConfigContent == "" && c.ImagePullDockerConfigFile != "" {
		err := shared.ReadFileValueString(c.ImagePullDockerConfigFile, &c.ImagePullDockerConfigContent)
		if err != nil && !os.IsNotExist(err) {
			return err
		}

		if (err != nil && os.IsNotExist(err)) || c.ImagePullDockerConfigContent == "" {
			logger.Logger.Warningf("Specified image pull secret file does not exist or has no content. Data plane cluster terraform may fail if operators to be installed use images from a private repository")
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

		err = readOperatorsSubscriptionConfigFile(c.StrimziOperatorOLMConfig.SubscriptionConfigFile, &c.StrimziOperatorOLMConfig.SubscriptionConfig)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Logger.Warningf("Specified Strimzi operator subscription config file %s does not exist. Default configuration will be used", c.StrimziOperatorOLMConfig.SubscriptionConfigFile)
			} else {
				return err
			}
		}

		err = readOperatorsSubscriptionConfigFile(c.KasFleetshardOperatorOLMConfig.SubscriptionConfigFile, &c.KasFleetshardOperatorOLMConfig.SubscriptionConfig)
		if err != nil {
			if os.IsNotExist(err) {
				logger.Logger.Warningf("Specified kas-fleet-shard operator subscription config file %s does not exist. Default configuration will be used", c.KasFleetshardOperatorOLMConfig.SubscriptionConfigFile)
			} else {
				return err
			}
		}
	}

	if c.IsDataPlaneAutoScalingEnabled() {
		err := shared.ReadYamlFile(c.DynamicScalingConfig.filePath, &c.DynamicScalingConfig.Configuration)
		if err != nil {
			return err
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

	err = c.NodePrewarmingConfig.readFile()
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

func readOperatorsSubscriptionConfigFile(file string, subscriptionConfig **operatorsv1alpha1.SubscriptionConfig) error {
	fileContents, err := shared.ReadFile(file)
	if err != nil {
		return err
	}

	return k8sYaml.UnmarshalStrict([]byte(fileContents), subscriptionConfig)
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
