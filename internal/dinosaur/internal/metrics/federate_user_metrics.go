package metrics

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/golang/glog"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/observatorium"
	"github.com/prometheus/client_golang/prometheus"
	pModel "github.com/prometheus/common/model"
)

type FederatedUserMetricsCollector struct {
	DinosaurMetricsMetadata map[string]constants.MetricsMetadata
	DinosaurMetrics         *observatorium.DinosaurMetrics
}

func NewFederatedUserMetricsCollector(dinosaurMetrics *observatorium.DinosaurMetrics) *FederatedUserMetricsCollector {
	return &FederatedUserMetricsCollector{
		DinosaurMetricsMetadata: constants.GetMetricsMetaData(),
		DinosaurMetrics:         dinosaurMetrics,
	}
}

func (f FederatedUserMetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	for _, metricMetadata := range f.DinosaurMetricsMetadata {
		ch <- f.buildMetricDesc(metricMetadata)
	}
}

func (f FederatedUserMetricsCollector) Collect(ch chan<- prometheus.Metric) {
	// collect metric
	for _, m := range *f.DinosaurMetrics {
		if m.Vector != nil {
			for _, v := range m.Vector {
				name := string(v.Metric["__name__"])

				// Check if we have metadata for the given metric
				if metadata, ok := f.DinosaurMetricsMetadata[name]; ok {
					switch metadata.Type {
					case prometheus.GaugeValue, prometheus.CounterValue:
						ch <- prometheus.MustNewConstMetric(
							f.buildMetricDesc(metadata),
							metadata.Type,
							float64(v.Value),
							f.extractLabelValues(v.Metric)...,
						)
					default:
						glog.Infof("skipping unsupported federated metric: %v (%v)", name, metadata.Type)
					}
				}
			}
		}
	}
}

// buildMetricDesc returns the metric description based on the metricMetadata passed in
func (f FederatedUserMetricsCollector) buildMetricDesc(metricMetadata constants.MetricsMetadata) *prometheus.Desc {
	return prometheus.NewDesc(
		metricMetadata.Name,
		metricMetadata.Help,
		metricMetadata.VariableLabels,
		metricMetadata.ConstantLabels,
	)
}

// extractLabelValues gets values of the labels from the given metric
// metricLabels is a label set with the following type map[LabelName]LabelValue
//
// The label values returned needs to be in the order of the variable labels that's specified in the metric description
func (f FederatedUserMetricsCollector) extractLabelValues(metricLabels pModel.Metric) []string {
	labelValues := []string{}
	metric := f.DinosaurMetricsMetadata[string(metricLabels["__name__"])]
	for _, label := range metric.VariableLabels {
		label := pModel.LabelName(label)
		labelValues = append(labelValues, string(metricLabels[label]))
	}
	return labelValues
}
