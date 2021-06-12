package provider

import "github.com/goava/di"

type Map map[string]di.Option

func (o Map) AsOption() di.Option {
	var options []di.Option
	for _, option := range o {
		options = append(options, option)
	}
	return di.Options(options...)
}
