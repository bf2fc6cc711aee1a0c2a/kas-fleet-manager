package ocm

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func Test_idGenerator_Generate(t *testing.T) {
	type fields struct {
		prefix string
	}
	tests := []struct {
		name       string
		fields     fields
		validateFn func(id string) error
	}{
		{
			name: "valid prefix",
			fields: fields{
				prefix: ClusterNamePrefix,
			},
			validateFn: func(id string) error {
				if !strings.HasPrefix(id, ClusterNamePrefix) {
					return errors.New(fmt.Sprintf("expected id to have prefix %s, got = %s", ClusterNamePrefix, id))
				}
				return nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := idGenerator{
				prefix: tt.fields.prefix,
			}
			got := i.Generate()
			if err := tt.validateFn(got); err != nil {
				t.Errorf("Generate() = %v", err.Error())
			}
			if len(got) > MaxClusterNameLength {
				t.Errorf("Generated ID length should not exceed 15 chars: %v", got)
			}
		})
	}
}
