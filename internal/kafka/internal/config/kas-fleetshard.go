package config

import "github.com/spf13/pflag"

type KasFleetshardConfig struct {
	PollInterval                           string `json:"poll_interval"`
	ResyncInterval                         string `json:"resync_interval"`
	EnableProvisionOfKasFleetshardOperator bool   `json:"enabled_provision_of_kas_fleetshard_operator"` // used only to disable provisioning of kas-fleet-shard-operator during testing so as to avoid creating a client in sso
}

func NewKasFleetshardConfig() *KasFleetshardConfig {
	return &KasFleetshardConfig{
		PollInterval:                           "15s",
		ResyncInterval:                         "60s",
		EnableProvisionOfKasFleetshardOperator: true, // default to true as we want to always provision kas-fleet-shard with an exception of some tests
	}
}

func (c *KasFleetshardConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.PollInterval, "kas-fleetshard-poll-interval", c.PollInterval, "Interval defining how often the synchronizer polls and gets updates from the control plane")
	fs.StringVar(&c.ResyncInterval, "kas-fleetshard-resync-interval", c.ResyncInterval, "Interval defining how often the synchronizer reports back status changes to the control plane")
}

func (c *KasFleetshardConfig) ReadFiles() error {
	return nil
}
