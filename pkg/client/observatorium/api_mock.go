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

// performs a query for the kafka metrics.
func (t *httpAPIMock) Query(ctx context.Context, query string, ts time.Time) (pModel.Value, pV1.Warnings, error) {
	values := getMockQueryData(query)
	return values, []string{}, nil
}

//Query(ctx context.Context, query string, ts time.Time) (model.Value, Warnings, error)
func (*httpAPIMock) QueryRange(ctx context.Context, query string, r pV1.Range) (pModel.Value, pV1.Warnings, error) {
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
func (*httpAPIMock) LabelNames(ctx context.Context, startTime time.Time, endTime time.Time) ([]string, pV1.Warnings, error) {
	return []string{}, pV1.Warnings{}, fmt.Errorf("not implemented")
}
func (*httpAPIMock) LabelValues(ctx context.Context, label string, startTime time.Time, endTime time.Time) (pModel.LabelValues, pV1.Warnings, error) {
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

//getMockQueryData
func getMockQueryData(query string) pModel.Vector {
	for key, values := range queryData {
		if strings.Contains(query, key) {
			return values
		}
	}
	return pModel.Vector{}

}

//getMockQueryRangeData
func getMockQueryRangeData(query string) pModel.Matrix {
	for key, values := range rangeQuerydata {
		if strings.Contains(query, key) {
			return values
		}
	}
	return pModel.Matrix{}

}

var rangeQuerydata = map[string]pModel.Matrix{
	"kubelet_volume_stats_available_bytes": {
		fakeMetricData("kubelet_volume_stats_available_bytes", 220792516608),
	},
	"kafka_server_brokertopicmetrics_messages_in_total": {
		fakeMetricData("kafka_server_brokertopicmetrics_messages_in_total", 0),
	},
	"kafka_server_brokertopicmetrics_bytes_in_total": {
		fakeMetricData("kafka_server_brokertopicmetrics_bytes_in_total", 0),
	},
	"kafka_server_brokertopicmetrics_bytes_out_total": {
		fakeMetricData("kafka_server_brokertopicmetrics_bytes_out_total", 0),
	},
	"kafka_controller_kafkacontroller_offline_partitions_count": {
		fakeMetricData("kafka_controller_kafkacontroller_offline_partitions_count", 0),
	},
	"kafka_controller_kafkacontroller_global_partition_count": {
		fakeMetricData("kafka_controller_kafkacontroller_global_partition_count", 0),
	},
	"sum by (namespace, topic)(kafka_log_log_size": {
		fakeMetricData("sum by (namespace, topic)(kafka_log_log_size", 220),
	},
}

func fakeMetricData(name string, value int) *pModel.SampleStream {
	return &pModel.SampleStream{
		Metric: pModel.Metric{
			"namespace": "kafka-namespace",
			"__name__":  pModel.LabelValue(name),
			"tenant_id": "whatever",
			"job":       "whatever",
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
}
