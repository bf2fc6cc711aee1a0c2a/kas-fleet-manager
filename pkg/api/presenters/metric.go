package presenters

import (
	pmod "github.com/prometheus/common/model"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/observatorium"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
)

func convertMatrix(from pmod.Matrix) []openapi.Metric {
	series := make([]openapi.Metric, len(from))

	for i, s := range from {
		series[i] = convertSampleStream(s)
	}
	return series
}

func convertSampleStream(from *pmod.SampleStream) openapi.Metric {
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
	return openapi.Metric{
		Metric: labelSet,
		Values: values,
	}
}
func convertSamplePair(from *pmod.SamplePair) openapi.Values {
	return openapi.Values{
		Timestamp: int64(from.Timestamp),
		Value:     float64(from.Value),
	}
}
func PresentMetrics(metrics *observatorium.KafkaMetrics) ([]openapi.Metric, *errors.ServiceError) {
	var out []openapi.Metric
	for _, m := range *metrics {
		if m.Err != nil {
			return nil, errors.GeneralError("error in metric %s: %v", m.Matrix, m.Err)
		}
		metric := convertMatrix(m.Matrix)
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
	return []string{"__name__", "strimzi_io_cluster", "topic", "persistentvolumeclaim"}
}
