package config

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/spf13/pflag"
)

type ProcessorsConfig struct {
	ProcessorEnableUnassignedProcessors bool `json:"processor_enable_unassigned_processors"`
}

var _ environments.ConfigModule = &ProcessorsConfig{}

func NewProcessorsConfig() *ProcessorsConfig {
	return &ProcessorsConfig{
		ProcessorEnableUnassignedProcessors: false,
	}
}

func (c *ProcessorsConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.ProcessorEnableUnassignedProcessors, "processor-enable-unassigned-processors", c.ProcessorEnableUnassignedProcessors, "Enable support for 'unassigned' state for Processors")
}

func (c *ProcessorsConfig) ReadFiles() error {
	// There ae no external configuration files for Processors
	return nil
}
