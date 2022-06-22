package dbapi

import (
	"testing"

	. "github.com/onsi/gomega"
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			input := DataPlaneKafkaStatus{Conditions: tt.statusConds}
			res, ok := input.GetReadyCondition()
			Expect(ok).To(Equal(tt.wantOK))
			Expect(res.Type).To(Equal(tt.wantCondType))
		})
	}
}
