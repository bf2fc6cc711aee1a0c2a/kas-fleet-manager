package provider

import (
	"github.com/goava/di"
)

type Provider interface {
	Providers() di.Option
}

func Func(f func() di.Option) func() Provider {
	return func() Provider {
		return providerFunc{apply: f}
	}
}

type providerFunc struct {
	apply func() di.Option
}

func (s providerFunc) Providers() di.Option {
	return s.apply()
}
