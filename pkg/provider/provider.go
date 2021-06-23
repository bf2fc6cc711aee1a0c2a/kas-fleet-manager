package provider

import (
	"github.com/goava/di"
)

type Provider interface {
	Providers() di.Option
}

func Func(f func(parent *di.Container) di.Option) func(container *di.Container) Provider {
	return func(container *di.Container) Provider {
		return providerFunc{parent: container, apply: f}
	}
}

type providerFunc struct {
	parent *di.Container
	apply  func(parent *di.Container) di.Option
}

func (s providerFunc) Providers() di.Option {
	return s.apply(s.parent)
}
