package ocm

import (
	"fmt"
	"strings"
	"testing"

	"github.com/onsi/gomega"
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
				prefix: "mk-",
			},
			validateFn: func(id string) error {
				if !strings.HasPrefix(id, "mk-") {
					return fmt.Errorf("expected id to have prefix %s, got = %s", "mk-", id)
				}
				return nil
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got := NewIDGenerator(tt.fields.prefix).Generate()
			g.Expect(tt.validateFn(got)).To(gomega.Succeed())
			g.Expect(len(got)).To(gomega.BeNumerically("<=", MaxClusterNameLength))
		})
	}
}
