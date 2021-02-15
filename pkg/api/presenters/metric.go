package presenters

import (
	pmod "github.com/prometheus/common/model"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/observatorium"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

func convertMatrix(from pmod.Matrix) []openapi.QueryRange {
	series := make([]openapi.QueryRange, len(from))

	for i, s := range from {
		series[i] = convertSampleStream(s)
	}
	return series
}
func convertVector(from pmod.Vector) []openapi.QueryInstant {
	series := make([]openapi.QueryInstant, len(from))

	for i, s := range from {
		series[i] = convertSample(s)
	}
	return series
}

func convertSampleStream(from *pmod.SampleStream) openapi.QueryRange {
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
	return openapi.QueryRange{
		Metric: labelSet,
		Values: values,
	}
}
func convertSample(from *pmod.Sample) openapi.QueryInstant {
	labelSet := make(map[string]string, len(from.Metric))
	for k, v := range from.Metric {
		if !isAllowedLabel(string(k)) {
			// Do not add these labels
			continue
		}
		labelSet[string(k)] = string(v)
	}
	return openapi.QueryInstant{
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
func PresentMetricsByQueryRange(metrics *observatorium.KafkaMetrics) ([]openapi.QueryRange, *errors.ServiceError) {
	var out []openapi.QueryRange
	for _, m := range *metrics {
		if m.Err != nil {
			return nil, errors.GeneralError("error in metric %s: %v", m.Matrix, m.Err)
		}
		metric := convertMatrix(m.Matrix)
		out = append(out, metric...)
	}
	return out, nil
}

func PresentMetricsByQueryInstant(metrics *observatorium.KafkaMetrics) ([]openapi.QueryInstant, *errors.ServiceError) {
	var out []openapi.QueryInstant
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
	return []string{"__name__", "strimzi_io_cluster", "topic", "persistentvolumeclaim", "pod"}
}
