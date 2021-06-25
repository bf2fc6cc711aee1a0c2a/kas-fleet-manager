package signalbus

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/provider"
	"github.com/goava/di"
)

func ConfigProviders() di.Option {
	return di.Provide(provider.Func(ServiceProviders))
}

func ServiceProviders() di.Option {
	return di.Provide(func(dbFactory *db.ConnectionFactory) *PgSignalBus {
		return NewPgSignalBus(NewSignalBus(), dbFactory)
	}, di.As(new(SignalBus)), di.As(new(provider.BootService)))
}
