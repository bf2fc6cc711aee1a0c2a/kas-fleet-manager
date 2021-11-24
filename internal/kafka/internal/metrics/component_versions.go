package metrics

import (
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
)

type versionsMetrics struct {
	kafkaService    services.KafkaService
	strimziVersion  *prometheus.GaugeVec
	kafkaVersion    *prometheus.GaugeVec
	kafkaIBPVersion *prometheus.GaugeVec
}

// need to invoked when the server is started and kafkaService is initialised
func RegisterVersionMetrics(kafkaService services.KafkaService) {
	m := newVersionMetrics(kafkaService)
	// for tests. This function will be called multiple times when run integration tests because `prometheus` is singleton
	prometheus.Unregister(m)
	prometheus.MustRegister(m)
}

func newVersionMetrics(kafkaService services.KafkaService) *versionsMetrics {
	return &versionsMetrics{
		kafkaService: kafkaService,
		strimziVersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "strimzi_version",
			Help: `Reports the version of Strimzi in terms of seconds since the epoch. 
The type 'actual' is the Strimzi version that is reported by kas-fleetshard.
The type 'desired' is the desired Strimzi version that is set in the kas-fleet-manager. 
If the type is 'upgrade' it means the Strimzi is being upgraded.
`,
		}, []string{"cluster_id", "kafka_id", "type", "version"}),
		kafkaVersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "kafka_version",
			Help: `Reports the version of Kafka in terms of seconds since the epoch.
The type 'actual' is the Kafka version that is reported by kas-fleetshard.
The type 'desired' is the desired Kafka version that is set in the kas-fleet-manager. 
If the type is 'upgrade' it means the Kafka is being upgraded.
`,
		}, []string{"cluster_id", "kafka_id", "type", "version"}),
		kafkaIBPVersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "kafka_ibp_version",
			Help: `Reports the version of Kafka in terms of seconds since the epoch.
The type 'actual' is the Kafka IBP version that is reported by kas-fleetshard.
The type 'desired' is the desired Kafka IBP version that is set in the kas-fleet-manager.
If the type is 'upgrade' it means the Kafka IBP version is being upgraded.
`,
		}, []string{"cluster_id", "kafka_id", "type", "version"}),
	}
}

func (m *versionsMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- m.strimziVersion.WithLabelValues("", "", "", "").Desc()
}

func (m *versionsMetrics) Collect(ch chan<- prometheus.Metric) {
	// list all the Kafka instances from kafkaServices and generate metrics for each
	// the generated metrics will be put on the channel
	if versions, err := m.kafkaService.ListComponentVersions(); err == nil {
		for _, v := range versions {
			// actual strimzi version
			actualStrimziMetric := m.strimziVersion.WithLabelValues(v.ClusterID, v.ID, "actual", v.ActualStrimziVersion)
			actualStrimziMetric.Set(float64(time.Now().Unix()))
			ch <- actualStrimziMetric
			//desired metric
			desiredStrimziMetric := m.strimziVersion.WithLabelValues(v.ClusterID, v.ID, "desired", v.DesiredStrimziVersion)
			desiredStrimziMetric.Set(float64(time.Now().Unix()))
			ch <- desiredStrimziMetric

			if v.StrimziUpgrading {
				strimziUpgradingMetric := m.strimziVersion.WithLabelValues(v.ClusterID, v.ID, "upgrade", v.DesiredStrimziVersion)
				strimziUpgradingMetric.Set(float64(time.Now().Unix()))
				ch <- strimziUpgradingMetric
			}

			// actual kafka version
			actualKafkaMetric := m.kafkaVersion.WithLabelValues(v.ClusterID, v.ID, "actual", v.ActualKafkaVersion)
			actualKafkaMetric.Set(float64(time.Now().Unix()))
			ch <- actualKafkaMetric
			//desired kafka version
			desiredKafkaMetric := m.kafkaVersion.WithLabelValues(v.ClusterID, v.ID, "desired", v.DesiredKafkaVersion)
			desiredKafkaMetric.Set(float64(time.Now().Unix()))
			ch <- desiredKafkaMetric

			if v.KafkaUpgrading {
				kafkaUpgradingMetric := m.kafkaVersion.WithLabelValues(v.ClusterID, v.ID, "upgrade", v.DesiredKafkaVersion)
				kafkaUpgradingMetric.Set(float64(time.Now().Unix()))
				ch <- kafkaUpgradingMetric
			}

			// actual kafka ibp version
			actualKafkaIBPMetric := m.kafkaIBPVersion.WithLabelValues(v.ClusterID, v.ID, "actual", v.ActualKafkaIBPVersion)
			actualKafkaIBPMetric.Set(float64(time.Now().Unix()))
			ch <- actualKafkaIBPMetric
			//desired kafka ibp version
			desiredKafkaIBPMetric := m.kafkaIBPVersion.WithLabelValues(v.ClusterID, v.ID, "desired", v.DesiredKafkaIBPVersion)
			desiredKafkaIBPMetric.Set(float64(time.Now().Unix()))
			ch <- desiredKafkaIBPMetric

			if v.KafkaIBPUpgrading {
				kafkaIBPUpgradingMetric := m.kafkaIBPVersion.WithLabelValues(v.ClusterID, v.ID, "upgrade", v.DesiredKafkaIBPVersion)
				kafkaIBPUpgradingMetric.Set(float64(time.Now().Unix()))
				ch <- kafkaIBPUpgradingMetric
			}
		}
	} else {
		logger.Logger.Errorf("failed to get component versions due to err: %v", err)
	}
}
