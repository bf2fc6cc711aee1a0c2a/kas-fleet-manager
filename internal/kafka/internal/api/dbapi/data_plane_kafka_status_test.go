package dbapi

import (
	"reflect"
	"testing"
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
				DataPlaneKafkaStatusCondition{Type: "CondType1"},
				DataPlaneKafkaStatusCondition{Type: "CondType2"},
			},
		},
		{
			name:         "When ready condition exists ok is true",
			wantOK:       true,
			wantCondType: "Ready",
			statusConds: []DataPlaneKafkaStatusCondition{
				DataPlaneKafkaStatusCondition{Type: "CondType1"},
				DataPlaneKafkaStatusCondition{Type: "Ready"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := DataPlaneKafkaStatus{Conditions: tt.statusConds}
			res, ok := input.GetReadyCondition()
			if !reflect.DeepEqual(ok, tt.wantOK) {
				t.Errorf("want: %v got: %v", tt.wantOK, ok)
			}
			if !reflect.DeepEqual(res.Type, tt.wantCondType) {
				t.Errorf("want: %v got: %v", tt.wantCondType, res.Type)
			}
		})
	}
}
