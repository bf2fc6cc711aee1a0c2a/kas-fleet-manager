package metrics

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
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
							ID:                     "1",
							ClusterID:              "cluster1",
							DesiredStrimziVersion:  "1.0.1",
							ActualStrimziVersion:   "1.0.0",
							StrimziUpgrading:       true,
							DesiredKafkaVersion:    "1.0.1",
							ActualKafkaVersion:     "1.0.0",
							KafkaUpgrading:         true,
							DesiredKafkaIBPVersion: "1.0",
							ActualKafkaIBPVersion:  "1.0",
							KafkaIBPUpgrading:      true,
						},
					}, nil
				},
			}},
			args: args{ch: make(chan prometheus.Metric, 100)},
			want: 9,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			m := newVersionMetrics(tt.fields.kafkaService)
			ch := tt.args.ch
			m.Collect(ch)
			g.Expect(ch).To(gomega.HaveLen(tt.want))
		})
	}
}

func TestVersionsMetrics_Describe(t *testing.T) {
	type fields struct {
		kafkaService services.KafkaService
	}

	type args struct {
		ch chan<- *prometheus.Desc
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name: "return",
			fields: fields{kafkaService: &services.KafkaServiceMock{
				ListComponentVersionsFunc: func() ([]services.KafkaComponentVersions, error) {
					return []services.KafkaComponentVersions{
						{
							ID:                    "1",
							ClusterID:             "cluster1",
							DesiredStrimziVersion: "1.0.1",
							ActualStrimziVersion:  "1.0.0",
							StrimziUpgrading:      true,
						},
					}, nil
				},
			}},
			args: args{ch: make(chan *prometheus.Desc, 100)},
			want: 1,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			m := newVersionMetrics(tt.fields.kafkaService)
			ch := tt.args.ch
			m.Describe(ch)
			g.Expect(ch).To(gomega.HaveLen(tt.want))
		})
	}
}
