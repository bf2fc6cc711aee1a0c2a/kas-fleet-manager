package metrics

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/observatorium"
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	pModel "github.com/prometheus/common/model"
)

func TestFederateMetrics_Collect(t *testing.T) {
	tests := []struct {
		name    string
		metrics observatorium.DinosaurMetrics
	}{
		{
			name: "test if correct number of metrics are gathered",
			metrics: observatorium.DinosaurMetrics{
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_server_brokertopicmetrics_messages_in_total",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_server_brokertopicmetrics_bytes_in_total",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_server_brokertopicmetrics_bytes_out_total",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_controller_dinosaurcontroller_offline_partitions_count",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_controller_dinosaurcontroller_global_partition_count",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_topic:dinosaur_log_log_size:sum",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_broker_quota_softlimitbytes",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_broker_quota_totalstorageusedbytes",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kubelet_volume_stats_available_bytes",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kubelet_volume_stats_used_bytes",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_namespace:haproxy_server_bytes_in_total:rate5m",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_namespace:haproxy_server_bytes_out_total:rate5m",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_topic:dinosaur_topic_partitions:sum",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_topic:dinosaur_topic_partitions:count",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "consumergroup:dinosaur_consumergroup_members:count",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_count:sum",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "dinosaur_namespace:dinosaur_server_socket_server_metrics_connection_creation_rate:sum",
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			collector := NewFederatedUserMetricsCollector(&tt.metrics)
			registry := prometheus.NewPedanticRegistry()
			registry.MustRegister(collector)
			federatedMetrics, err := registry.Gather()
			metadata := constants.GetMetricsMetaData()

			if err != nil {
				t.Error(err)
			}

			if len(federatedMetrics) != len(tt.metrics) {
				t.Errorf("expected %v metrics to be gathered", len(tt.metrics))
			}

			for _, metric := range federatedMetrics {
				name := *metric.Name
				metricMetadata := metadata[name]
				var metricType = int32(*metric.Type)

				if metricMetadata.Help != *metric.Help {
					t.Errorf("unexpected help '%v' for metric %v", *metric.Help, name)
				}

				if metricMetadata.TypeName != io_prometheus_client.MetricType_name[metricType] {
					t.Errorf("unexpected type '%v' for metric %v", *metric.Type, name)
				}
			}
		})
	}
}
