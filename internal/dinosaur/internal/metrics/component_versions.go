package metrics

import (
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
)

type versionsMetrics struct {
	dinosaurService    services.DinosaurService
	strimziVersion  *prometheus.GaugeVec
	dinosaurVersion    *prometheus.GaugeVec
	dinosaurIBPVersion *prometheus.GaugeVec
}

// need to invoked when the server is started and dinosaurService is initialised
func RegisterVersionMetrics(dinosaurService services.DinosaurService) {
	m := newVersionMetrics(dinosaurService)
	// for tests. This function will be called multiple times when run integration tests because `prometheus` is singleton
	prometheus.Unregister(m)
	prometheus.MustRegister(m)
}

func newVersionMetrics(dinosaurService services.DinosaurService) *versionsMetrics {
	return &versionsMetrics{
		dinosaurService: dinosaurService,
		strimziVersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "strimzi_version",
			Help: `Reports the version of Strimzi in terms of seconds since the epoch. 
The type 'actual' is the Strimzi version that is reported by fleetshard.
The type 'desired' is the desired Strimzi version that is set in the fleet-manager. 
If the type is 'upgrade' it means the Strimzi is being upgraded.
`,
		}, []string{"cluster_id", "dinosaur_id", "type", "version"}),
		dinosaurVersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dinosaur_version",
			Help: `Reports the version of Dinosaur in terms of seconds since the epoch.
The type 'actual' is the Dinosaur version that is reported by fleetshard.
The type 'desired' is the desired Dinosaur version that is set in the fleet-manager. 
If the type is 'upgrade' it means the Dinosaur is being upgraded.
`,
		}, []string{"cluster_id", "dinosaur_id", "type", "version"}),
		dinosaurIBPVersion: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "dinosaur_ibp_version",
			Help: `Reports the version of Dinosaur in terms of seconds since the epoch.
The type 'actual' is the Dinosaur IBP version that is reported by fleetshard.
The type 'desired' is the desired Dinosaur IBP version that is set in the fleet-manager.
If the type is 'upgrade' it means the Dinosaur IBP version is being upgraded.
`,
		}, []string{"cluster_id", "dinosaur_id", "type", "version"}),
	}
}

func (m *versionsMetrics) Describe(ch chan<- *prometheus.Desc) {
	ch <- m.strimziVersion.WithLabelValues("", "", "", "").Desc()
}

func (m *versionsMetrics) Collect(ch chan<- prometheus.Metric) {
	// list all the Dinosaur instances from dinosaurServices and generate metrics for each
	// the generated metrics will be put on the channel
	if versions, err := m.dinosaurService.ListComponentVersions(); err == nil {
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

			// actual dinosaur version
			actualDinosaurMetric := m.dinosaurVersion.WithLabelValues(v.ClusterID, v.ID, "actual", v.ActualDinosaurVersion)
			actualDinosaurMetric.Set(float64(time.Now().Unix()))
			ch <- actualDinosaurMetric
			//desired dinosaur version
			desiredDinosaurMetric := m.dinosaurVersion.WithLabelValues(v.ClusterID, v.ID, "desired", v.DesiredDinosaurVersion)
			desiredDinosaurMetric.Set(float64(time.Now().Unix()))
			ch <- desiredDinosaurMetric

			if v.DinosaurUpgrading {
				dinosaurUpgradingMetric := m.dinosaurVersion.WithLabelValues(v.ClusterID, v.ID, "upgrade", v.DesiredDinosaurVersion)
				dinosaurUpgradingMetric.Set(float64(time.Now().Unix()))
				ch <- dinosaurUpgradingMetric
			}

			// actual dinosaur ibp version
			actualDinosaurIBPMetric := m.dinosaurIBPVersion.WithLabelValues(v.ClusterID, v.ID, "actual", v.ActualDinosaurIBPVersion)
			actualDinosaurIBPMetric.Set(float64(time.Now().Unix()))
			ch <- actualDinosaurIBPMetric
			//desired dinosaur ibp version
			desiredDinosaurIBPMetric := m.dinosaurIBPVersion.WithLabelValues(v.ClusterID, v.ID, "desired", v.DesiredDinosaurIBPVersion)
			desiredDinosaurIBPMetric.Set(float64(time.Now().Unix()))
			ch <- desiredDinosaurIBPMetric

			if v.DinosaurIBPUpgrading {
				dinosaurIBPUpgradingMetric := m.dinosaurIBPVersion.WithLabelValues(v.ClusterID, v.ID, "upgrade", v.DesiredDinosaurIBPVersion)
				dinosaurIBPUpgradingMetric.Set(float64(time.Now().Unix()))
				ch <- dinosaurIBPUpgradingMetric
			}
		}
	} else {
		logger.Logger.Errorf("failed to get component versions due to err: %v", err)
	}
}
