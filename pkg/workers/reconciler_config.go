package workers

import (
	"time"

	"github.com/spf13/pflag"
)

type ReconcilerConfig struct {
	ReconcilerRepeatInterval               time.Duration `json:"reconciler_repeat_interval"`
	LeaderLeaseExpirationTime              time.Duration `json:"leader_lease_expiration_time"`
	LeaderElectionReconcilerRepeatInterval time.Duration `json:"leader_election_reconciler_repeat_interval"`
}

func NewReconcilerConfig() *ReconcilerConfig {
	return &ReconcilerConfig{
		ReconcilerRepeatInterval:               30 * time.Second,
		LeaderLeaseExpirationTime:              1 * time.Minute,
		LeaderElectionReconcilerRepeatInterval: 15 * time.Second,
	}
}

func (r *ReconcilerConfig) AddFlags(fs *pflag.FlagSet) {
	fs.DurationVar(&r.ReconcilerRepeatInterval, "reconciler-repeat-interval", r.ReconcilerRepeatInterval, "The frequency at which each scheduled reconciler worker is running.")
	fs.DurationVar(&r.LeaderLeaseExpirationTime, "leader-lease-expiration-time", r.LeaderLeaseExpirationTime, "The time before a lease expires.")
	fs.DurationVar(&r.LeaderElectionReconcilerRepeatInterval, "leader-election-reconciler-repeat-interval", r.LeaderElectionReconcilerRepeatInterval, "The scheduled interval between leader election reconciliation.")
}

func (c *ReconcilerConfig) ReadFiles() error {
	return nil
}
