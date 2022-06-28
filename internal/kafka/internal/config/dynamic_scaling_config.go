package config

import (
	"fmt"
)

type DynamicScalingConfig struct {
	filePath      string
	Configuration map[string]InstanceTypeDynamicScalingConfig
}

func NewDynamicScalingConfig() DynamicScalingConfig {
	return DynamicScalingConfig{
		filePath:      "config/dynamic-scaling-configuration.yaml",
		Configuration: make(map[string]InstanceTypeDynamicScalingConfig),
	}
}

func (c *DynamicScalingConfig) ForInstanceType(instanceTypeID string) (InstanceTypeDynamicScalingConfig, bool) {
	if instanceTypeConfig, found := c.Configuration[instanceTypeID]; found {
		return instanceTypeConfig, true
	}
	return InstanceTypeDynamicScalingConfig{}, false
}

func (c *DynamicScalingConfig) InstanceTypeConfigs() map[string]InstanceTypeDynamicScalingConfig {
	newConfig := make(map[string]InstanceTypeDynamicScalingConfig, len(c.Configuration))
	for k, v := range c.Configuration {
		newConfig[k] = v
	}

	return newConfig
}

func (c *DynamicScalingConfig) validate() error {
	if c.filePath == "" {
		return fmt.Errorf("dynamic scaling config file path is not specified")
	}

	if c.Configuration == nil {
		return fmt.Errorf("dynamic scaling configuration file %s has not been read or is empty", c.filePath)
	}

	for _, v := range c.Configuration {
		err := v.validate()
		if err != nil {
			return err
		}
	}

	return nil
}

type InstanceTypeDynamicScalingConfig struct {
	ComputeNodesConfig  *DynamicScalingComputeNodesConfig `yaml:"compute_nodes_config"`
	CapacityReservation int                               `yaml:"capacity_reservation"`
}

func (c *InstanceTypeDynamicScalingConfig) validate() error {
	if c.ComputeNodesConfig == nil {
		return fmt.Errorf("compute_nodes_config is mandatory")
	}

	if c.CapacityReservation <= 0 {
		return fmt.Errorf("capacity_reservation has to be greater than 0")
	}

	err := c.ComputeNodesConfig.validate()
	if err != nil {
		return err
	}

	return nil
}

type DynamicScalingComputeNodesConfig struct {
	MaxComputeNodes int `yaml:"max_compute_nodes"`
}

func (c *DynamicScalingComputeNodesConfig) validate() error {
	if c.MaxComputeNodes <= 0 {
		return fmt.Errorf("max_compute_nodes has to be greater than 0")
	}

	return nil
}
