package config

import (
	"fmt"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/cloudproviders"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
)

const minNumberOfComputeNodesForClusterWideWorkload = 3

// use a single instance of Validate, it caches struct info
var validate *validator.Validate = validator.New()

type DynamicScalingConfig struct {
	FilePath                                      string                                                   `validate:"required"`
	ComputeMachinePerCloudProvider                map[cloudproviders.CloudProviderID]ComputeMachinesConfig `yaml:"compute_machine_per_cloud_provider" validate:"required"`
	EnableDynamicScaleUpManagerScaleUpTrigger     bool                                                     `yaml:"enable_dynamic_data_plane_scale_up"`
	EnableDynamicScaleDownManagerScaleDownTrigger bool                                                     `yaml:"enable_dynamic_data_plane_scale_down"`
	NewDataPlaneOpenShiftVersion                  string                                                   `yaml:"new_data_plane_openshift_version"`
}

func NewDynamicScalingConfig() DynamicScalingConfig {
	return DynamicScalingConfig{
		FilePath:                       "config/dynamic-scaling-configuration.yaml",
		ComputeMachinePerCloudProvider: map[cloudproviders.CloudProviderID]ComputeMachinesConfig{},
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

func (c *DynamicScalingConfig) validate() error {
	err := validate.Struct(c)
	if err != nil {
		return errors.Wrap(err, "error validating dynamic scaling configuration")
	}

	for k, v := range c.ComputeMachinePerCloudProvider {
		err := v.validate(k)
		if err != nil {
			return err
		}
	}

	return nil
}

type ComputeNodesAutoscalingConfig struct {
	MaxComputeNodes int `yaml:"max_compute_nodes" validate:"gt=0,gtefield=MinComputeNodes"`
	MinComputeNodes int `yaml:"min_compute_nodes" validate:"gt=0"`
}

type ComputeMachineConfig struct {
	ComputeMachineType      string                         `yaml:"compute_machine_type" validate:"required"`
	ComputeNodesAutoscaling *ComputeNodesAutoscalingConfig `yaml:"compute_node_autoscaling" validate:"required"`
}

type ComputeMachinesConfig struct {
	ClusterWideWorkload          *ComputeMachineConfig           `yaml:"cluster_wide_workload" validate:"required"`
	KafkaWorkloadPerInstanceType map[string]ComputeMachineConfig `yaml:"kafka_workload_per_instance_type" validate:"required"`
}

func (c *ComputeMachinesConfig) GetKafkaWorkloadConfigForInstanceType(instanceTypeID string) (ComputeMachineConfig, bool) {
	if instanceTypeConfig, found := c.KafkaWorkloadPerInstanceType[instanceTypeID]; found {
		return instanceTypeConfig, true
	}
	return ComputeMachineConfig{}, false
}

func (c ComputeMachinesConfig) validate(cloudProvider cloudproviders.CloudProviderID) error {
	err := validate.Struct(c)
	if err != nil {
		return errors.Wrapf(err, "error validating compute machines configuration for cloud provider %q", cloudProvider)
	}

	err = c.ClusterWideWorkload.validate("cluster wide workload", cloudProvider)
	if err != nil {
		return err
	}

	if c.ClusterWideWorkload.ComputeNodesAutoscaling.MinComputeNodes < minNumberOfComputeNodesForClusterWideWorkload {
		return fmt.Errorf("cluster wide minimum number of nodes for cloud provider %q has to be greate or equal to %d", cloudProvider, minNumberOfComputeNodesForClusterWideWorkload)
	}

	for k, v := range c.KafkaWorkloadPerInstanceType {
		err := v.validate(fmt.Sprintf("instance type %s", k), cloudProvider)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c ComputeMachineConfig) validate(logKey string, cloudProvider cloudproviders.CloudProviderID) error {
	err := validate.Struct(c)
	if err != nil {
		return errors.Wrapf(err, "error validating compute machine configuration for %q in cloud provider %q", logKey, cloudProvider)
	}

	return nil
}
