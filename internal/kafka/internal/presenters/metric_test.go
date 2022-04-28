package presenters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	mockMetrics "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/metrics"
	mocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	pModel "github.com/prometheus/common/model"

	. "github.com/onsi/gomega"
)

func TestConvertMatrix(t *testing.T) {
	type args struct {
		from pModel.Matrix
	}

	tests := []struct {
		name string
		args args
		want []public.RangeQuery
	}{
		{
			name: "should return successfully converted RangeQuery slice",
			args: args{
				from: mockMetrics.GetSampleMatrix(),
			},
			want: mockMetrics.GetSampleRangeQuerySlice(),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(convertMatrix(tt.args.from)).To(Equal(tt.want))
		})
	}
}

func TestConvertVector(t *testing.T) {
	type args struct {
		from pModel.Vector
	}

	tests := []struct {
		name string
		args args
		want []public.InstantQuery
	}{
		{
			name: "should return successfully converted InstantQuery slice",
			args: args{
				from: mockMetrics.GetSampleVector(),
			},
			want: mocks.GetSampleInstantQuerySlice(),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(convertVector(tt.args.from)).To(Equal(tt.want))
		})
	}
}

func TestConvertSampleStream(t *testing.T) {
	type args struct {
		from *pModel.SampleStream
	}

	tests := []struct {
		name string
		args args
		want public.RangeQuery
	}{
		{
			name: "should return RangeQuery converted from SampleStream with not supported label",
			args: args{
				from: mocks.GetNotAllowedSampleStream(),
			},
			want: mocks.GetNotAllowedLabelRangeQuery(),
		},
		{
			name: "should return RangeQuery converted from SampleStream with supported label",
			args: args{
				from: mocks.GetSampleStream(),
			},
			want: mocks.GetRangeQueryStringMapMetric(),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(convertSampleStream(tt.args.from)).To(Equal(tt.want))
		})
	}
}

func TestConvertSample(t *testing.T) {
	type args struct {
		from *pModel.Sample
	}

	tests := []struct {
		name string
		args args
		want public.InstantQuery
	}{
		{
			name: "should return InstantQuery converted from Sample with not supported label",
			args: args{
				from: mocks.GetNotAllowedLabelsSample(),
			},
			want: mocks.GetSampleInstantQueryNoMetrics(),
		},
		{
			name: "should return InstantQuery converted from Sample with supported label",
			args: args{
				from: mocks.GetSampleWithValidLabel(),
			},
			want: mocks.GetInstantQueryWithMetrics(),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(convertSample(tt.args.from)).To(Equal(tt.want))
		})
	}
}

func TestConvertSamplePair(t *testing.T) {
	type args struct {
		from *pModel.SamplePair
	}

	tests := []struct {
		name string
		args args
		want public.Values
	}{
		{
			name: "should successfully convert to Values from non-empty SamplePair",
			args: args{
				from: mocks.GetSamplePair(),
			},
			want: mocks.GetSampleValue(),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(convertSamplePair(tt.args.from)).To(Equal(tt.want))
		})
	}
}

func TestPresentMetricsByRangeQuery(t *testing.T) {
	type args struct {
		metrics *observatorium.KafkaMetrics
	}

	tests := []struct {
		name    string
		args    args
		want    []public.RangeQuery
		wantErr bool
	}{
		{
			name: "should throw an error if attempting to present RangeQuery slice with metrics containing error",
			args: args{
				metrics: mocks.GetKafkaMetricWithError(),
			},
			wantErr: true,
		},
		{
			name: "should present RangeQuery slice when using valid metrics object",
			args: args{
				metrics: mocks.GetSampleKafkaMetric(),
			},
			wantErr: false,
			want:    mocks.GetSampleRangeQuerySlice(),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			firstQuery, err := PresentMetricsByRangeQuery(tt.args.metrics)
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for PresentMetricsByRangeQuery: %v", err)
			}
			if !tt.wantErr {
				Expect(tt.want[0].Metric["__name__"]).To(Equal(firstQuery[0].Metric["__name__"]))
			}
		})
	}
}

func TestPresentMetricsByInstantQuery(t *testing.T) {
	type args struct {
		metrics *observatorium.KafkaMetrics
	}

	tests := []struct {
		name    string
		args    args
		want    []public.InstantQuery
		wantErr bool
	}{
		{
			name: "should throw an error if attempting to present InstantQuery slice with metrics containing error",
			args: args{
				metrics: mocks.GetKafkaMetricWithError(),
			},
			wantErr: true,
		},
		{
			name: "should present InstantQuery slice when using valid metrics object",
			args: args{
				metrics: mocks.GetSampleKafkaMetric(),
			},
			wantErr: false,
			want:    mocks.GetSampleInstantQuerySlice(),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query, err := PresentMetricsByInstantQuery(tt.args.metrics)
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for PresentMetricsByInstantQuery: %v", err)
				return
			}
			Expect(tt.wantErr).To(Equal(err != nil), "unexpected error for PresentMetricsByInstantQuery")
			Expect(query).To(Equal(tt.want))
		})
	}
}

func TestIsAllowedLabel(t *testing.T) {
	type args struct {
		label string
	}

	notAllowedLabel := "not allowed label"

	allowedLabel := "__name__"

	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should return false if providing not allowed label",
			args: args{
				label: notAllowedLabel,
			},
			want: false,
		},
		{
			name: "should return true if providing allowed label",
			args: args{
				label: allowedLabel,
			},
			want: true,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(isAllowedLabel(tt.args.label)).To(Equal(tt.want))
		})
	}
}

func TestGetSupportedLabels(t *testing.T) {
	tests := []struct {
		name string
		want []string
	}{
		{
			name: "Should return a slice of supported labels",
			want: []string{"__name__", "strimzi_io_cluster", "topic", "persistentvolumeclaim", "statefulset_kubernetes_io_pod_name", "exported_service", "exported_pod", "route", "broker_id", "quota_type"},
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(getSupportedLabels()).To(Equal(tt.want))
		})
	}
}
