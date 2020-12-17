package observatorium

import (
	"context"
	"fmt"
	"time"

	pV1 "github.com/prometheus/client_golang/api/prometheus/v1"
	pModel "github.com/prometheus/common/model"
)

func (c *Client) MockAPI() pV1.API {
	return &httpAPIMock{}
}

type httpAPIMock struct {
}

type mockAPISample struct {
	value    pModel.Value
	warnings pV1.Warnings
	err      error
}

// performs a query for the kafka state, retun value '1' which mean ready state.
func (t *httpAPIMock) Query(ctx context.Context, query string, ts time.Time) (pModel.Value, pV1.Warnings, error) {
	api := SetMockValue(1)
	return api.value, api.warnings, api.err
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

//Query(ctx context.Context, query string, ts time.Time) (model.Value, Warnings, error)
func (*httpAPIMock) QueryRange(ctx context.Context, query string, r pV1.Range) (pModel.Value, pV1.Warnings, error) {
	return nil, pV1.Warnings{}, fmt.Errorf("not implemented")
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

//SetMockValue in httpAPIMock
func SetMockValue(expectedValue float64) *mockAPISample {
	sample := &pModel.Sample{Value: pModel.SampleValue(expectedValue)}
	vector := []*pModel.Sample{sample}

	api := mockAPISample{
		value: pModel.Vector(vector),
	}

	return &api

}
