package mocks

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	pModel "github.com/prometheus/common/model"
)

const (
	SampleTimeStamp  = int64(12345)
	SampleFloatValue = 200.99
)

func GetSampleKafkaMetric() *observatorium.KafkaMetrics {
	return &observatorium.KafkaMetrics{
		{
			Matrix: pModel.Matrix{
				GetSampleStream(),
			},
			Vector: GetSampleVector(),
		},
	}
}

func GetKafkaMetricWithError() *observatorium.KafkaMetrics {
	return &observatorium.KafkaMetrics{
		{
			Vector: GetSampleVector(),
			Err:    errors.New(errors.ErrorGeneral, "error"),
		},
	}
}

func GetSamplePair() *pModel.SamplePair {
	return &pModel.SamplePair{
		Timestamp: pModel.Time(SampleTimeStamp),
		Value:     SampleFloatValue,
	}
}

func GetInstantQueryWithMetrics() public.InstantQuery {
	return public.InstantQuery{
		Metric:    GetSampleStringMapMetric(),
		Timestamp: SampleTimeStamp,
		Value:     SampleFloatValue,
	}
}

func GetSampleWithValidLabel() *pModel.Sample {
	return &pModel.Sample{
		Timestamp: pModel.Time(SampleTimeStamp),
		Value:     pModel.SampleValue(SampleFloatValue),
		Metric:    GetSamplePmodelmapMetric(),
	}
}

func GetSampleInstantQueryNoMetrics() public.InstantQuery {
	return public.InstantQuery{
		Metric:    make(map[string]string, 1),
		Timestamp: SampleTimeStamp,
		Value:     SampleFloatValue,
	}
}

func GetNotAllowedLabelsSample() *pModel.Sample {
	return &pModel.Sample{
		Timestamp: pModel.Time(SampleTimeStamp),
		Value:     pModel.SampleValue(SampleFloatValue),
		Metric: map[pModel.LabelName]pModel.LabelValue{
			"notSupportedLabel": "value",
		},
	}
}

func GetNotAllowedSampleStream() *pModel.SampleStream {
	return &pModel.SampleStream{
		Metric: map[pModel.LabelName]pModel.LabelValue{
			"notSupportedLabel": "value",
		},
		Values: []pModel.SamplePair{
			*GetSamplePair(),
		},
	}
}

func GetSampleStream() *pModel.SampleStream {
	return &pModel.SampleStream{
		Metric: GetSamplePmodelmapMetric(),
		Values: []pModel.SamplePair{
			*GetSamplePair(),
		},
	}
}

func GetSampleMatrix() pModel.Matrix {
	return pModel.Matrix{
		GetSampleStream(),
	}
}

func GetSampleVector() pModel.Vector {
	return pModel.Vector{
		&pModel.Sample{
			Metric:    GetSamplePmodelmapMetric(),
			Value:     SampleFloatValue,
			Timestamp: pModel.Time(SampleTimeStamp),
		},
	}
}

func GetSampleValuePair() pModel.SamplePair {
	return pModel.SamplePair{
		Timestamp: pModel.Time(SampleTimeStamp),
		Value:     pModel.SampleValue(SampleFloatValue),
	}
}

func GetSamplePmodelmapMetric() map[pModel.LabelName]pModel.LabelValue {
	return map[pModel.LabelName]pModel.LabelValue{
		"__name__": "kafka_server_brokertopicmetrics_messages_in_total",
	}
}

func GetSampleStringMapMetric() map[string]string {
	return map[string]string{
		"__name__": "kafka_server_brokertopicmetrics_messages_in_total",
	}
}

func GetSampleValue() public.Values {
	return public.Values{
		Timestamp: int64(SampleTimeStamp),
		Value:     float64(SampleFloatValue),
	}
}

func GetNotAllowedLabelRangeQuery() public.RangeQuery {
	return public.RangeQuery{
		Metric: make(map[string]string, 1),
		Values: []public.Values{
			GetSampleValue(),
		},
	}
}

func GetRangeQueryStringMapMetric() public.RangeQuery {
	return public.RangeQuery{
		Metric: GetSampleStringMapMetric(),
		Values: []public.Values{
			GetSampleValue(),
		},
	}
}

func GetSampleRangeQuerySlice() []public.RangeQuery {
	return []public.RangeQuery{
		{
			Metric: GetSampleStringMapMetric(),
			Values: []public.Values{
				GetSampleValue(),
			},
		},
	}
}

func GetSampleInstantQuerySlice() []public.InstantQuery {
	return []public.InstantQuery{
		{
			Metric:    GetSampleStringMapMetric(),
			Timestamp: SampleTimeStamp,
			Value:     SampleFloatValue,
		},
	}
}

func GetSampleInstantQuerySliceEmptyMetrics() []public.InstantQuery {
	return []public.InstantQuery{
		{
			Timestamp: SampleTimeStamp,
			Value:     SampleFloatValue,
		},
	}
}
