package metrics

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/prometheus/client_golang/prometheus"
	io_prometheus_client "github.com/prometheus/client_model/go"
	pModel "github.com/prometheus/common/model"
)

func TestFederateMetrics_Collect(t *testing.T) {
	tests := []struct {
		name    string
		metrics observatorium.KafkaMetrics
	}{
		{
			name: "test if correct number of metrics are gathered",
			metrics: observatorium.KafkaMetrics{
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_server_brokertopicmetrics_messages_in_total",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_server_brokertopicmetrics_bytes_in_total",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_server_brokertopicmetrics_bytes_out_total",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_controller_kafkacontroller_offline_partitions_count",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_controller_kafkacontroller_global_partition_count",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_topic:kafka_log_log_size:sum",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_broker_quota_softlimitbytes",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_broker_quota_hardlimitbytes",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_broker_quota_totalstorageusedbytes",
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
								"__name__": "kafka_namespace:haproxy_server_bytes_in_total:rate5m",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_namespace:haproxy_server_bytes_out_total:rate5m",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_topic:kafka_topic_partitions:sum",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_topic:kafka_topic_partitions:count",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "consumergroup:kafka_consumergroup_members:count",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_namespace:kafka_server_socket_server_metrics_connection_count:sum",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_namespace:kafka_server_socket_server_metrics_connection_creation_rate:sum",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_topic:kafka_server_brokertopicmetrics_messages_in_total:rate5m",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_topic:kafka_server_brokertopicmetrics_bytes_in_total:rate5m",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_topic:kafka_server_brokertopicmetrics_bytes_out_total:rate5m",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_instance_spec_brokers_desired_count",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_instance_max_message_size_limit",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_instance_partition_limit",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_instance_connection_limit",
							},
						},
					},
				},
				{
					Vector: []*pModel.Sample{
						{
							Metric: map[pModel.LabelName]pModel.LabelValue{
								"__name__": "kafka_instance_connection_creation_rate_limit",
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
