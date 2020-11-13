package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	jobsSubsystem            = "managed_services_jobs"
	transactionsSubsystem    = "managed_services_transactions"
	managedServicesSubsystem = "managed_services_api"

	requestDuration = "cluster_creation_duration"

	labelJobType = "jobType"
)

// JobType metric to capture
type JobType string

var (
	// JobTypeClusterCreate - cluster_create job type
	JobTypeClusterCreate JobType = "cluster_create"
)

// JobsMetricsLabels is the slice of labels to add to job metrics
var JobsMetricsLabels = []string{
	labelJobType,
}

// create a new histogramVec for cluster creation duration
var requestClusterCreationDurationMetric = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: jobsSubsystem,
		Name:      requestDuration,
		Help:      "Cluster creation duration in seconds.",
		Buckets: []float64{
			1800.0,
			2400.0,
			3600.0,
			4800.0,
			7200.0,
			10800.0,
		},
	},
	JobsMetricsLabels,
)

// UpdateJobDurationMetric records the duration of a job type
func UpdateClusterCreationDurationMetric(jobType JobType, elapsed time.Duration) {
	labels := prometheus.Labels{
		labelJobType: string(jobType),
	}
	requestClusterCreationDurationMetric.With(labels).Observe(elapsed.Seconds())
}

// register the metric(s)
func init() {
	prometheus.MustRegister(requestClusterCreationDurationMetric)
}
