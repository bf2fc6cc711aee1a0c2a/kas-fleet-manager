package config

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
)

type DynamicScalingConfig struct {
	filePath                                      string
	ComputeNodesPerInstanceType                   map[string]InstanceTypeDynamicScalingConfig          `yaml:"compute_nodes_per_instance_type"`
	MachineTypePerCloudProvider                   map[cloudproviders.CloudProviderID]MachineTypeConfig `yaml:"machine_type_per_cloud_provider"`
	EnableDynamicScaleUpManagerScaleUpTrigger     bool                                                 `yaml:"enable_dynamic_data_plane_scale_up"`
	EnableDynamicScaleDownManagerScaleDownTrigger bool                                                 `yaml:"enable_dynamic_data_plane_scale_down"`
	NewDataPlaneOpenShiftVersion                  string                                               `yaml:"new_data_plane_openshift_version"`
}

func NewDynamicScalingConfig() DynamicScalingConfig {
	return DynamicScalingConfig{
		filePath:                                      "config/dynamic-scaling-configuration.yaml",
		ComputeNodesPerInstanceType:                   map[string]InstanceTypeDynamicScalingConfig{},
		MachineTypePerCloudProvider:                   map[cloudproviders.CloudProviderID]MachineTypeConfig{},
		EnableDynamicScaleUpManagerScaleUpTrigger:     true,
		EnableDynamicScaleDownManagerScaleDownTrigger: true,
		NewDataPlaneOpenShiftVersion:                  "",
	}
}

func (c *DynamicScalingConfig) IsDataplaneScaleUpEnabled() bool {
	return c.EnableDynamicScaleUpManagerScaleUpTrigger
}

func (c *DynamicScalingConfig) IsDataplaneScaleDownEnabled() bool {
	return c.EnableDynamicScaleDownManagerScaleDownTrigger
}

func (c *DynamicScalingConfig) GetConfigForInstanceType(instanceTypeID string) (InstanceTypeDynamicScalingConfig, bool) {
	if instanceTypeConfig, found := c.ComputeNodesPerInstanceType[instanceTypeID]; found {
		return instanceTypeConfig, true
	}
	return InstanceTypeDynamicScalingConfig{}, false
}

func (c *DynamicScalingConfig) validate() error {
	if c.filePath == "" {
		return fmt.Errorf("dynamic scaling config file path is not specified")
	}

	if c.ComputeNodesPerInstanceType == nil {
		return fmt.Errorf("dynamic scaling configuration file %s has not been read or is empty", c.filePath)
	}

	for k, v := range c.ComputeNodesPerInstanceType {
		err := v.validate(k)
		if err != nil {
			return err
		}
	}

	for k, v := range c.MachineTypePerCloudProvider {
		err := v.validate(k)
		if err != nil {
			return err
		}
	}

	return nil
}

type InstanceTypeDynamicScalingConfig struct {
	ComputeNodesConfig *DynamicScalingComputeNodesConfig `yaml:"compute_nodes_config"`
}

func (c *InstanceTypeDynamicScalingConfig) validate(instanceType string) error {
	if c.ComputeNodesConfig == nil {
		return fmt.Errorf("compute_nodes_config is mandatory. It is missing from %q instance type", instanceType)
	}

	return c.ComputeNodesConfig.validate(instanceType)
}

type DynamicScalingComputeNodesConfig struct {
	MaxComputeNodes int `yaml:"max_compute_nodes"`
}

func (c *DynamicScalingComputeNodesConfig) validate(instanceType string) error {
	if c.MaxComputeNodes <= 0 {
		return fmt.Errorf("max_compute_nodes for %q instance type has to be greater than 0", instanceType)
	}

	return nil
}

type MachineTypeConfig struct {
	ClusterWideWorkloadMachineType string `yaml:"cluster_wide_workload_machine_type"`
	KafkaWorkloadMachineType       string `yaml:"kafka_workload_machine_type"`
}

func (c MachineTypeConfig) validate(cloudProvider cloudproviders.CloudProviderID) error {
	if c.ClusterWideWorkloadMachineType == "" {
		return fmt.Errorf("cluster_wide_workload_machine_type is mandatory. It is missing for %q cloud provider.", cloudProvider)
	}

	if c.KafkaWorkloadMachineType == "" {
		return fmt.Errorf("kafka_workload_machine_type is mandatory. It is missing for %q cloud provider.", cloudProvider)
	}

	return nil
}
