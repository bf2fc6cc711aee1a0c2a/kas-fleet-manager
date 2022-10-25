package config

import (
	"fmt"
	"os"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/pkg/errors"
)

const defaultBaseStreamingUnitSize = "x1"

type NodePrewarmingConfig struct {
	filePath      string
	Configuration map[string]InstanceTypeNodePrewarmingConfig
}

func NewNodePrewarmingConfig() NodePrewarmingConfig {
	return NodePrewarmingConfig{
		filePath:      "config/node-prewarming-configuration.yaml",
		Configuration: make(map[string]InstanceTypeNodePrewarmingConfig),
	}
}

func (c *NodePrewarmingConfig) ForInstanceType(instanceTypeID string) (InstanceTypeNodePrewarmingConfig, bool) {
	instanceTypeConfig, found := c.Configuration[instanceTypeID]

	if !found {
		return InstanceTypeNodePrewarmingConfig{}, false
	}

	if instanceTypeConfig.BaseStreamingUnitSize == "" {
		instanceTypeConfig.BaseStreamingUnitSize = defaultBaseStreamingUnitSize
	}

	return instanceTypeConfig, true
}

func (c *NodePrewarmingConfig) validate(kafkaConfig *KafkaConfig) error {
	for instanceType, configuration := range c.Configuration {
		err := configuration.validate(instanceType, kafkaConfig)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *NodePrewarmingConfig) readFile() error {
	err := shared.ReadYamlFile(c.filePath, &c.Configuration)
	if err != nil {
		if !os.IsNotExist(err) {
			logger.Logger.Warningf("the node prewarming configuration file '%s' does not exists. No capacity reservation will be applied", c.filePath)
			return nil
		}

		return err
	}

	return nil
}

type InstanceTypeNodePrewarmingConfig struct {
	BaseStreamingUnitSize string `yaml:"base_streaming_unit_size"`
	NumReservedInstances  int    `yaml:"num_reserved_instances"`
}

func (c *InstanceTypeNodePrewarmingConfig) validate(instanceType string, kafkaConfig *KafkaConfig) error {
	baseStreamingUnit := c.BaseStreamingUnitSize
	if baseStreamingUnit == "" {
		baseStreamingUnit = defaultBaseStreamingUnitSize
	}

	_, err := kafkaConfig.GetKafkaInstanceSize(instanceType, baseStreamingUnit)

	if err != nil {
		return errors.Wrapf(err, "error validating node prewarming configuration for instance type %s", instanceType)
	}

	switch {
	case c.NumReservedInstances == 0:
		logger.Logger.Warningf("no capacity reservation will be applied for instance type: %s.", instanceType)
	case c.NumReservedInstances < 0:
		return fmt.Errorf("num_reserved_instances cannot be a negative number for instance type: %s", instanceType)
	}

	return nil
}
