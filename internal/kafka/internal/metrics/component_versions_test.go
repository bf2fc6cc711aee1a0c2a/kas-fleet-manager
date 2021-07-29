package metrics

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/prometheus/client_golang/prometheus"
	"testing"
)

func TestVersionsMetrics_Collect(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
	}

	type args struct {
		ch chan<- prometheus.Metric
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "will generate metrics",
			fields: fields{kafkaService: &services.KafkaServiceMock{
				ListComponentVersionsFunc: func() ([]services.KafkaComponentVersions, error) {
					return []services.KafkaComponentVersions{
						{
							ID:                    "1",
							ClusterID:             "cluster1",
							DesiredStrimziVersion: "1.0.1",
							ActualStrimziVersion:  "1.0.0",
							StrimziUpgrading:      true,
							DesiredKafkaVersion:   "1.0.1",
							ActualKafkaVersion:    "1.0.0",
							KafkaUpgrading:        false,
						},
					}, nil
				},
			}},
			args: args{ch: make(chan prometheus.Metric, 100)},
			want: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newVersionMetrics(tt.fields.kafkaService)
			ch := tt.args.ch
			m.Collect(ch)
			if len(ch) != tt.want {
				t.Errorf("expect to have %d metrics but got %d", tt.want, len(ch))
			}
		})
	}
}
