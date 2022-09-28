package config

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

// use a single instance of Validate, it caches struct info
var validate *validator.Validate = validator.New()

type DynamicScalingConfig struct {
	filePath                                      string                                               `validate:"required"`
	ComputeNodesPerInstanceType                   map[string]InstanceTypeDynamicScalingConfig          `yaml:"compute_nodes_per_instance_type" validate:"required"`
	MachineTypePerCloudProvider                   map[cloudproviders.CloudProviderID]MachineTypeConfig `yaml:"machine_type_per_cloud_provider" validate:"required"`
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

func (c *DynamicScalingConfig) IsDataplaneScaleUpTriggerEnabled() bool {
	return c.EnableDynamicScaleUpManagerScaleUpTrigger
}

func (c *DynamicScalingConfig) IsDataplaneScaleDownTriggerEnabled() bool {
	return c.EnableDynamicScaleDownManagerScaleDownTrigger
}

func (c *DynamicScalingConfig) GetConfigForInstanceType(instanceTypeID string) (InstanceTypeDynamicScalingConfig, bool) {
	if instanceTypeConfig, found := c.ComputeNodesPerInstanceType[instanceTypeID]; found {
		return instanceTypeConfig, true
	}
	return InstanceTypeDynamicScalingConfig{}, false
}

func (c *DynamicScalingConfig) validate() error {
	err := validate.Struct(c)
	if err != nil {
		return errors.Wrap(err, "error validating dynamic scaling configuration")
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
	ComputeNodesConfig *DynamicScalingComputeNodesConfig `yaml:"compute_nodes_config" validate:"required"`
}

func (c *InstanceTypeDynamicScalingConfig) validate(instanceType string) error {
	err := validate.Struct(c)
	if err != nil {
		return errors.Wrapf(err, "error validating dynamic node configuration for instance type %q", instanceType)
	}

	return nil
}

type DynamicScalingComputeNodesConfig struct {
	MaxComputeNodes int `yaml:"max_compute_nodes" validate:"gt=0"`
}

type MachineTypeConfig struct {
	ClusterWideWorkloadMachineType string `yaml:"cluster_wide_workload_machine_type" validate:"required"`
	KafkaWorkloadMachineType       string `yaml:"kafka_workload_machine_type" validate:"required"`
}

func (c MachineTypeConfig) validate(cloudProvider cloudproviders.CloudProviderID) error {
	err := validate.Struct(c)
	if err != nil {
		return errors.Wrapf(err, "error validating machine type configuration for cloud provider %q", cloudProvider)
	}

	return nil
}
