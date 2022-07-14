package dbapi

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestDataPlaneKafkastatus_GetReadyCondition(t *testing.T) {
	tests := []struct {
		name         string
		wantCondType string
		wantOK       bool
		statusConds  []DataPlaneKafkaStatusCondition
	}{
		{
			name:   "When no ready condition exists ok is false",
			wantOK: false,
			statusConds: []DataPlaneKafkaStatusCondition{
				{Type: "CondType1"},
				{Type: "CondType2"},
			},
		},
		{
			name:         "When ready condition exists ok is true",
			wantOK:       true,
			wantCondType: "Ready",
			statusConds: []DataPlaneKafkaStatusCondition{
				{Type: "CondType1"},
				{Type: "Ready"},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			input := DataPlaneKafkaStatus{Conditions: tt.statusConds}
			res, ok := input.GetReadyCondition()
			g.Expect(ok).To(gomega.Equal(tt.wantOK))
			g.Expect(res.Type).To(gomega.Equal(tt.wantCondType))
		})
	}
}
