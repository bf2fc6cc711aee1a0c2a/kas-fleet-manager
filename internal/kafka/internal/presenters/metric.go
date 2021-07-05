package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	pmod "github.com/prometheus/common/model"
)

func convertMatrix(from pmod.Matrix) []public.RangeQuery {
	series := make([]public.RangeQuery, len(from))

	for i, s := range from {
		series[i] = convertSampleStream(s)
	}
	return series
}
func convertVector(from pmod.Vector) []public.InstantQuery {
	series := make([]public.InstantQuery, len(from))

	for i, s := range from {
		series[i] = convertSample(s)
	}
	return series
}

func convertSampleStream(from *pmod.SampleStream) public.RangeQuery {
	labelSet := make(map[string]string, len(from.Metric))
	for k, v := range from.Metric {
		if !isAllowedLabel(string(k)) {
			// Do not add these labels
			continue
		}
		labelSet[string(k)] = string(v)
	}
	values := make([]public.Values, len(from.Values))
	for i, v := range from.Values {
		values[i] = convertSamplePair(&v)
	}
	return public.RangeQuery{
		Metric: labelSet,
		Values: values,
	}
}
func convertSample(from *pmod.Sample) public.InstantQuery {
	labelSet := make(map[string]string, len(from.Metric))
	for k, v := range from.Metric {
		if !isAllowedLabel(string(k)) {
			// Do not add these labels
			continue
		}
		labelSet[string(k)] = string(v)
	}
	return public.InstantQuery{
		Metric:    labelSet,
		Timestamp: int64(from.Timestamp),
		Value:     float64(from.Value),
	}
}

func convertSamplePair(from *pmod.SamplePair) public.Values {
	return public.Values{
		Timestamp: int64(from.Timestamp),
		Value:     float64(from.Value),
	}
}
func PresentMetricsByRangeQuery(metrics *observatorium.KafkaMetrics) ([]public.RangeQuery, *errors.ServiceError) {
	var out []public.RangeQuery
	for _, m := range *metrics {
		if m.Err != nil {
			return nil, errors.GeneralError("error in metric %s: %v", m.Matrix, m.Err)
		}
		metric := convertMatrix(m.Matrix)
		out = append(out, metric...)
	}
	return out, nil
}

func PresentMetricsByInstantQuery(metrics *observatorium.KafkaMetrics) ([]public.InstantQuery, *errors.ServiceError) {
	var out []public.InstantQuery
	for _, m := range *metrics {
		if m.Err != nil {
			return nil, errors.GeneralError("error in metric %s: %v", m.Matrix, m.Err)
		}
		metric := convertVector(m.Vector)
		out = append(out, metric...)
	}
	return out, nil
}

func isAllowedLabel(lable string) bool {
	for _, labelName := range getSupportedLables() {
		if lable == labelName {
			return true
		}
	}

	return false
}

func getSupportedLables() []string {
	return []string{"__name__", "strimzi_io_cluster", "topic", "persistentvolumeclaim", "statefulset_kubernetes_io_pod_name", "exported_service", "exported_pod", "route"}
}
