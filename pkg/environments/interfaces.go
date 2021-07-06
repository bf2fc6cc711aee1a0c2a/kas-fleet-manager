package environments

import (
	"github.com/goava/di"
	"github.com/gorilla/mux"
	"github.com/spf13/pflag"
)

// ConfigModule values can load configuration from flags and files
type ConfigModule interface {
	AddFlags(fs *pflag.FlagSet)
	ReadFiles() error
}

type ServiceValidator interface {
	Validate() error
}

// BootService are services that get started on application boot.
type BootService interface {
	Start()
	Stop()
}

// RouteLoader is load http routes into the https server's mux.Router
type RouteLoader interface {
	AddRoutes(mainRouter *mux.Router) error
}

type EnvHook struct {
	Func di.Invocation
}
type BeforeCreateServicesHook EnvHook
type AfterCreateServicesHook EnvHook
