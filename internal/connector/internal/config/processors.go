package config

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/environments"
	"github.com/spf13/pflag"
)

type ProcessorsConfig struct {
	ProcessorsEnabled bool `json:"processors_enabled"`
}

var _ environments.ConfigModule = &ProcessorsConfig{}

func NewProcessorsConfig() *ProcessorsConfig {
	return &ProcessorsConfig{
		ProcessorsEnabled: false,
	}
}

func (c *ProcessorsConfig) AddFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&c.ProcessorsEnabled, "processors-enabled", c.ProcessorsEnabled, "Enable support for (Smart Event) Processors")
}

func (c *ProcessorsConfig) ReadFiles() error {
	// There ae no external configuration files for Processors
	return nil
}
