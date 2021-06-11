package common

import (
	"github.com/goava/di"
	"github.com/gorilla/mux"
)

type RouteLoader interface {
	AddRoutes(mainRouter *mux.Router) error
}

type InjectionMap map[string]di.Option

func (o InjectionMap) AsOption() di.Option {
	var options []di.Option
	for _, option := range o {
		options = append(options, option)
	}
	return di.Options(options...)
}
