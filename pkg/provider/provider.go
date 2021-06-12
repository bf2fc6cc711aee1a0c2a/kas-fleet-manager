package provider

import (
	"github.com/goava/di"
)

type Provider interface {
	Providers() (Map, error)
}

func Func(f func(parent *di.Container) (Map, error)) func(container *di.Container) Provider {
	return func(container *di.Container) Provider {
		return providerFunc{parent: container, apply: f}
	}
}

type providerFunc struct {
	parent *di.Container
	apply  func(parent *di.Container) (Map, error)
}

func (s providerFunc) Providers() (Map, error) {
	return s.apply(s.parent)
}
