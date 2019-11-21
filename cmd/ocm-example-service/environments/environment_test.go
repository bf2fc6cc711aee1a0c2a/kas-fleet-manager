package environments

import (
	"reflect"
	"testing"

	"github.com/spf13/pflag"
)

func TestLoadServices(t *testing.T) {
	env := Environment()
	// Override environment name
	env.Name = "testing"
	err := env.AddFlags(pflag.CommandLine)
	if err != nil {
		t.Errorf("Unable to add flags for testing environment: %s", err.Error())
		return
	}
	pflag.Parse()
	err = env.Initialize()
	if err != nil {
		t.Errorf("Unable to load testing environment: %s", err.Error())
		return
	}

	s := reflect.ValueOf(env.Services)

	for i := 0; i < s.NumField(); i++ {
		if s.Field(i).IsNil() {
			t.Errorf("Service %v is nil", s)
		}
	}
}
