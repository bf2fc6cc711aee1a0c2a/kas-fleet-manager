package presenters

import (
	pmod "github.com/prometheus/common/model"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/observatorium"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

func convertMatrix(from pmod.Matrix) []openapi.RangeQuery {
	series := make([]openapi.RangeQuery, len(from))

	for i, s := range from {
		series[i] = convertSampleStream(s)
	}
	return series
}
func convertVector(from pmod.Vector) []openapi.InstantQuery {
	series := make([]openapi.InstantQuery, len(from))

	for i, s := range from {
		series[i] = convertSample(s)
	}
	return series
}

func convertSampleStream(from *pmod.SampleStream) openapi.RangeQuery {
	labelSet := make(map[string]string, len(from.Metric))
	for k, v := range from.Metric {
		if !isAllowedLabel(string(k)) {
			// Do not add these labels
			continue
		}
		labelSet[string(k)] = string(v)
	}
	values := make([]openapi.Values, len(from.Values))
	for i, v := range from.Values {
		values[i] = convertSamplePair(&v)
	}
	return openapi.RangeQuery{
		Metric: labelSet,
		Values: values,
	}
}
func convertSample(from *pmod.Sample) openapi.InstantQuery {
	labelSet := make(map[string]string, len(from.Metric))
	for k, v := range from.Metric {
		if !isAllowedLabel(string(k)) {
			// Do not add these labels
			continue
		}
		labelSet[string(k)] = string(v)
	}
	return openapi.InstantQuery{
		Metric:    labelSet,
		Timestamp: int64(from.Timestamp),
		Value:     float64(from.Value),
	}
}

func convertSamplePair(from *pmod.SamplePair) openapi.Values {
	return openapi.Values{
		Timestamp: int64(from.Timestamp),
		Value:     float64(from.Value),
	}
}
func PresentMetricsByRangeQuery(metrics *observatorium.KafkaMetrics) ([]openapi.RangeQuery, *errors.ServiceError) {
	var out []openapi.RangeQuery
	for _, m := range *metrics {
		if m.Err != nil {
			return nil, errors.GeneralError("error in metric %s: %v", m.Matrix, m.Err)
		}
		metric := convertMatrix(m.Matrix)
		out = append(out, metric...)
	}
	return out, nil
}

func PresentMetricsByInstantQuery(metrics *observatorium.KafkaMetrics) ([]openapi.InstantQuery, *errors.ServiceError) {
	var out []openapi.InstantQuery
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
	return []string{"__name__", "strimzi_io_cluster", "topic", "persistentvolumeclaim", "statefulset_kubernetes_io_pod_name","exported_service","exported_pod","route"}
}
