package observatorium

import (
	"context"
	"fmt"
	"strings"
	"time"

	pV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	pModel "github.com/prometheus/common/model"
)

func (c *Client) MockAPI() pV1.API {
	return &httpAPIMock{}
}

type httpAPIMock struct {
}

var _ pV1.API = &httpAPIMock{}

func (t *httpAPIMock) WalReplay(ctx context.Context) (pV1.WalReplayStatus, error) {
	return pV1.WalReplayStatus{}, fmt.Errorf("not implemented")
}

func (t *httpAPIMock) QueryExemplars(ctx context.Context, query string, startTime time.Time, endTime time.Time) ([]pV1.ExemplarQueryResult, error) {
	return []pV1.ExemplarQueryResult{}, fmt.Errorf("not implemented")
}

// performs a query for the kafka metrics.
func (t *httpAPIMock) Query(ctx context.Context, query string, ts time.Time, opts ...pV1.Option) (pModel.Value, pV1.Warnings, error) {
	values := getMockQueryData(query)
	return values, []string{}, nil
}

// QueryRange(ctx context.Context, query string, r pV1.Range) (pModel.Value, pV1.Warnings, error) Performs a query range for the kafka metrics
func (*httpAPIMock) QueryRange(ctx context.Context, query string, r pV1.Range, opts ...pV1.Option) (pModel.Value, pV1.Warnings, error) {
	values := getMockQueryRangeData(query)
	return values, []string{}, nil
}

// Not implemented
func (*httpAPIMock) Alerts(ctx context.Context) (pV1.AlertsResult, error) {
	return pV1.AlertsResult{}, fmt.Errorf("not implemented")
}
func (*httpAPIMock) AlertManagers(ctx context.Context) (pV1.AlertManagersResult, error) {
	return pV1.AlertManagersResult{}, fmt.Errorf("not implemented")
}
func (*httpAPIMock) CleanTombstones(ctx context.Context) error {
	return fmt.Errorf("not implemented")
}
func (*httpAPIMock) Config(ctx context.Context) (pV1.ConfigResult, error) {
	return pV1.ConfigResult{}, fmt.Errorf("not implemented")
}
func (*httpAPIMock) DeleteSeries(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) error {
	return fmt.Errorf("not implemented")
}
func (*httpAPIMock) Flags(ctx context.Context) (pV1.FlagsResult, error) {
	return pV1.FlagsResult{}, fmt.Errorf("not implemented")
}
func (*httpAPIMock) LabelNames(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) ([]string, pV1.Warnings, error) {
	return []string{}, pV1.Warnings{}, fmt.Errorf("not implemented")
}
func (*httpAPIMock) LabelValues(ctx context.Context, label string, matches []string, startTime time.Time, endTime time.Time) (pModel.LabelValues, pV1.Warnings, error) {
	return pModel.LabelValues{}, pV1.Warnings{}, fmt.Errorf("not implemented")
}

func (*httpAPIMock) Series(ctx context.Context, matches []string, startTime time.Time, endTime time.Time) ([]pModel.LabelSet, pV1.Warnings, error) {
	return []pModel.LabelSet{}, pV1.Warnings{}, fmt.Errorf("not implemented")
}
func (*httpAPIMock) Snapshot(ctx context.Context, skipHead bool) (pV1.SnapshotResult, error) {
	return pV1.SnapshotResult{}, fmt.Errorf("not implemented")
}
func (*httpAPIMock) Rules(ctx context.Context) (pV1.RulesResult, error) {
	return pV1.RulesResult{}, fmt.Errorf("not implemented")
}
func (*httpAPIMock) Targets(ctx context.Context) (pV1.TargetsResult, error) {
	return pV1.TargetsResult{}, fmt.Errorf("not implemented")
}
func (*httpAPIMock) TargetsMetadata(ctx context.Context, matchTarget string, metric string, limit string) ([]pV1.MetricMetadata, error) {
	return []pV1.MetricMetadata{}, fmt.Errorf("not implemented")
}
func (*httpAPIMock) Metadata(ctx context.Context, metric string, limit string) (map[string][]pV1.Metadata, error) {
	return nil, fmt.Errorf("not implemented")
}

func (*httpAPIMock) TSDB(ctx context.Context) (pV1.TSDBResult, error) {
	return pV1.TSDBResult{}, fmt.Errorf("not implemented")
}

func (*httpAPIMock) Runtimeinfo(ctx context.Context) (pV1.RuntimeinfoResult, error) {

	return pV1.RuntimeinfoResult{}, fmt.Errorf("not implemented")
}

func (*httpAPIMock) Buildinfo(ctx context.Context) (pV1.BuildinfoResult, error) {
	return pV1.BuildinfoResult{}, fmt.Errorf("not implemented")
}

// getMockQueryData
func getMockQueryData(query string) pModel.Vector {
	for key, values := range queryData {
		if strings.HasPrefix(query, key) {
			return values
		}
	}
	return pModel.Vector{}

}

// getMockQueryRangeData
func getMockQueryRangeData(query string) pModel.Matrix {
	for key, values := range rangeQuerydata {
		if strings.HasPrefix(query, key) {
			return values
		}
	}
	return pModel.Matrix{}

}

var rangeQuerydata = map[string]pModel.Matrix{
	"kafka_server_brokertopicmetrics_messages_in_total": {
		fakeMetricData("kafka_server_brokertopicmetrics_messages_in_total", 3040),
	},

	"kafka_server_brokertopicmetrics_bytes_in_total": {
		fakeMetricData("kafka_server_brokertopicmetrics_bytes_in_total", 293617),
	},
	"kafka_server_brokertopicmetrics_bytes_out_total": {
		fakeMetricData("kafka_server_brokertopicmetrics_bytes_out_total", 152751),
	},
	"kubelet_volume_stats_available_bytes": {
		fakeMetricData("kubelet_volume_stats_available_bytes", 220792516608),
	},
	"kafka_controller_kafkacontroller_offline_partitions_count": {
		fakeMetricData("kafka_controller_kafkacontroller_offline_partitions_count", 0),
	},
	"kafka_controller_kafkacontroller_global_partition_count": {
		fakeMetricData("kafka_controller_kafkacontroller_global_partition_count", 0),
	},
	"kafka_broker_quota_softlimitbytes": {
		fakeMetricData("kafka_broker_quota_softlimitbytes", 10000),
	},
	"kafka_broker_quota_totalstorageusedbytes": {
		fakeMetricData("kafka_broker_quota_totalstorageusedbytes", 1237582),
	},
	"kafka_topic:kafka_log_log_size:sum": {
		fakeMetricData("kafka_topic:kafka_log_log_size:sum", 220),
	},
	"kafka_namespace:kafka_server_socket_server_metrics_connection_creation_rate:sum": {
		fakeMetricData("kafka_namespace:kafka_server_socket_server_metrics_connection_creation_rate:sum", 20),
	},
	"kafka_topic:kafka_topic_partitions:sum": {
		fakeMetricData("kafka_topic:kafka_topic_partitions:sum", 20),
	},
	"kafka_topic:kafka_topic_partitions:count": {
		fakeMetricData("kafka_topic:kafka_topic_partitions:count", 20),
	},
	"consumergroup:kafka_consumergroup_members:count": {
		fakeMetricData("consumergroup:kafka_consumergroup_members:count", 20),
	},
	"kafka_namespace:kafka_server_socket_server_metrics_connection_count:sum": {
		fakeMetricData("kafka_namespace:kafka_server_socket_server_metrics_connection_count:sum", 20),
	},
	"kafka_topic:kafka_server_brokertopicmetrics_messages_in_total:rate5m": {
		fakeMetricData("kafka_topic:kafka_server_brokertopicmetrics_messages_in_total:rate5m", 1321),
	},
	"kafka_topic:kafka_server_brokertopicmetrics_bytes_in_total:rate5m": {
		fakeMetricData("kafka_topic:kafka_server_brokertopicmetrics_bytes_in_total:rate5m", 9834345),
	},
	"kafka_topic:kafka_server_brokertopicmetrics_bytes_out_total:rate5m": {
		fakeMetricData("kafka_topic:kafka_server_brokertopicmetrics_bytes_out_total:rate5m", 9194889),
	},

	"{__name__=~'kafka_topic:kafka_topic_partitions:sum|kafka_topic:kafka_topic_partitions:count|consumergroup:kafka_consumergroup_members:count|kafka_namespace:kafka_server_socket_server_metrics_connection_count:sum|kafka_namespace:kafka_server_socket_server_metrics_connection_creation_rate:sum|kafka_topic:kafka_server_brokertopicmetrics_messages_in_total:rate5m|kafka_topic:kafka_server_brokertopicmetrics_bytes_in_total:rate5m|kafka_topic:kafka_server_brokertopicmetrics_bytes_out_total:rate5m|kafka_instance_spec_brokers_desired_count|kafka_instance_max_message_size_limit|kafka_instance_partition_limit|kafka_instance_connection_limit|kafka_instance_connection_creation_rate_limit'": {
		fakeMetricData("kafka_topic:kafka_topic_partitions:sum", 20),
	},

	"{__name__=~'kubelet_volume_stats_available_bytes|kubelet_volume_stats_used_bytes', persistentvolumeclaim=~\"data-.*-kafka-[0-9]*$\"": {
		fakeMetricData("kubelet_volume_stats_available_bytes", 20),
	},

	"{__name__=~'kafka_broker_quota_softlimitbytes|kafka_broker_quota_hardlimitbytes|kafka_broker_quota_totalstorageusedbytes|kafka_broker_client_quota_limit', strimzi_io_kind=~'Kafka'": {
		fakeMetricData("kafka_broker_quota_softlimitbytes", 20),
	},

	"{__name__=~'kafka_server_brokertopicmetrics_messages_in_total|kafka_server_brokertopicmetrics_bytes_in_total|kafka_server_brokertopicmetrics_bytes_out_total'": {
		fakeMetricData("kafka_server_brokertopicmetrics_messages_in_total", 20),
	},

	"{__name__=~'kafka_controller_kafkacontroller_offline_partitions_count|kafka_controller_kafkacontroller_global_partition_count'": {
		fakeMetricData("kafka_controller_kafkacontroller_offline_partitions_count", 20),
	},

	"{__name__=~'kafka_topic:kafka_log_log_size:sum', topic!~'__redhat_.*|__consumer_offsets|__transaction_state'": {
		fakeMetricData("kafka_topic:kafka_log_log_size:sum", 20),
	},

	"{__name__=~'kafka_namespace:haproxy_server_bytes_in_total:rate5m|kafka_namespace:haproxy_server_bytes_out_total:rate5m'": {
		fakeMetricData("kafka_namespace:haproxy_server_bytes_in_total:rate5m", 20),
	},
}

func fakeMetricData(name string, value int) *pModel.SampleStream {
	return &pModel.SampleStream{
		Metric: pModel.Metric{
			"__name__":           pModel.LabelValue(name),
			"pod":                "whatever",
			"strimzi_io_cluster": "whatever",
		},
		Values: []pModel.SamplePair{{Timestamp: 0, Value: pModel.SampleValue(value)},
			{Timestamp: 0, Value: pModel.SampleValue(value)}},
	}
}

var queryData = map[string]pModel.Vector{

	"strimzi_resource_state": pModel.Vector{
		&pModel.Sample{
			Metric: pModel.Metric{
				"strimzi_io_kind": "Kafka",
				"strimzi_io_name": "test-kafka",
				"namespace":       "my-kafka-namespace",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     1,
		},
	},

	"{__name__=~'kafka_topic:kafka_topic_partitions:sum|kafka_topic:kafka_topic_partitions:count|consumergroup:kafka_consumergroup_members:count|kafka_namespace:kafka_server_socket_server_metrics_connection_count:sum|kafka_namespace:kafka_server_socket_server_metrics_connection_creation_rate:sum|kafka_topic:kafka_server_brokertopicmetrics_messages_in_total:rate5m|kafka_topic:kafka_server_brokertopicmetrics_bytes_in_total:rate5m|kafka_topic:kafka_server_brokertopicmetrics_bytes_out_total:rate5m|kafka_instance_spec_brokers_desired_count|kafka_instance_max_message_size_limit|kafka_instance_partition_limit|kafka_instance_connection_limit|kafka_instance_connection_creation_rate_limit'": pModel.Vector{
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":           "kafka_topic:kafka_topic_partitions:sum",
				"pod":                "whatever",
				"strimzi_io_cluster": "whatever",
				"topic":              "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     293617,
		},
	},

	"{__name__=~'kubelet_volume_stats_available_bytes|kubelet_volume_stats_used_bytes', persistentvolumeclaim=~\"data-.*-kafka-[0-9]*$\"": pModel.Vector{
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":           "kubelet_volume_stats_available_bytes",
				"pod":                "whatever",
				"strimzi_io_cluster": "whatever",
				"topic":              "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     293617,
		},
	},

	"{__name__=~'kafka_broker_quota_softlimitbytes|kafka_broker_quota_hardlimitbytes|kafka_broker_quota_totalstorageusedbytes|kafka_broker_client_quota_limit', strimzi_io_kind=~'Kafka'": pModel.Vector{
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":           "kafka_broker_quota_softlimitbytes",
				"pod":                "whatever",
				"strimzi_io_cluster": "whatever",
				"topic":              "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     293617,
		},
	},

	"{__name__=~'kafka_server_brokertopicmetrics_messages_in_total|kafka_server_brokertopicmetrics_bytes_in_total|kafka_server_brokertopicmetrics_bytes_out_total'": pModel.Vector{
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":           "kafka_server_brokertopicmetrics_messages_in_total",
				"pod":                "whatever",
				"strimzi_io_cluster": "whatever",
				"topic":              "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     293617,
		},
	},

	"{__name__=~'kafka_controller_kafkacontroller_offline_partitions_count|kafka_controller_kafkacontroller_global_partition_count'": pModel.Vector{
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":           "kafka_controller_kafkacontroller_offline_partitions_count",
				"pod":                "whatever",
				"strimzi_io_cluster": "whatever",
				"topic":              "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     293617,
		},
	},

	"{__name__=~'kafka_topic:kafka_log_log_size:sum', topic!~'__redhat_.*|__consumer_offsets|__transaction_state'": pModel.Vector{
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":           "kafka_topic:kafka_log_log_size:sum",
				"pod":                "whatever",
				"strimzi_io_cluster": "whatever",
				"topic":              "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     293617,
		},
	},

	"{__name__=~'kafka_namespace:haproxy_server_bytes_in_total:rate5m|kafka_namespace:haproxy_server_bytes_out_total:rate5m'": pModel.Vector{
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":           "kafka_namespace:haproxy_server_bytes_in_total:rate5m",
				"pod":                "whatever",
				"strimzi_io_cluster": "whatever",
				"topic":              "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     293617,
		},
	},

	"kafka_server_brokertopicmetrics_bytes_in_total": pModel.Vector{
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":           "kafka_server_brokertopicmetrics_bytes_in_total",
				"pod":                "whatever",
				"strimzi_io_cluster": "whatever",
				"topic":              "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     293617,
		},
	},
	"kafka_server_brokertopicmetrics_messages_in_total": pModel.Vector{
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":           "kafka_server_brokertopicmetrics_messages_in_total",
				"pod":                "whatever",
				"strimzi_io_cluster": "whatever",
				"topic":              "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     1016,
		},
	},
	"kafka_broker_quota_softlimitbytes": pModel.Vector{
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":           "kafka_broker_quota_softlimitbytes",
				"pod":                "whatever",
				"strimzi_io_cluster": "whatever",
				"topic":              "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     30000,
		},
	},
	"kafka_broker_quota_totalstorageusedbytes": pModel.Vector{
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":           "kafka_broker_quota_totalstorageusedbytes",
				"pod":                "whatever",
				"strimzi_io_cluster": "whatever",
				"topic":              "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     2207924332,
		},
	},
	"kubelet_volume_stats_available_bytes": pModel.Vector{
		&pModel.Sample{
			Metric: pModel.Metric{
				"__name__":              "kubelet_volume_stats_available_bytes",
				"persistentvolumeclaim": "whatever",
			},
			Timestamp: pModel.Time(1607506882175),
			Value:     220792492032,
		},
	},
}
