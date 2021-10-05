package config

import "github.com/spf13/pflag"

type KasFleetshardConfig struct {
	PollInterval   string `json:"poll_interval"`
	ResyncInterval string `json:"resync_interval"`
}

func NewKasFleetshardConfig() *KasFleetshardConfig {
	return &KasFleetshardConfig{
		PollInterval:   "15s",
		ResyncInterval: "60s",
	}
}

func (c *KasFleetshardConfig) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(&c.PollInterval, "kas-fleetshard-poll-interval", c.PollInterval, "Interval defining how often the synchronizer polls and gets updates from the control plane")
	fs.StringVar(&c.ResyncInterval, "kas-fleetshard-resync-interval", c.ResyncInterval, "Interval defining how often the synchronizer reports back status changes to the control plane")
}

func (c *KasFleetshardConfig) ReadFiles() error {
	return nil
}
