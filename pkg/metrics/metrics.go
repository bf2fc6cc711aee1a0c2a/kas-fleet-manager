package metrics

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/constants"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	jobsSubsystem         = "managed_services_jobs"
	transactionsSubsystem = "managed_services_transactions"

	// ClusterCreateRequestDuration - name of cluster creation duration metric
	ClusterCreateRequestDuration = "worker_cluster_duration"
	// KafkaCreateRequestDuration - name of kafka creation duration metric
	KafkaCreateRequestDuration = "worker_kafka_duration"

	labelJobType               = "jobType"
	kafkaStatus                = "status"
	kafkaStatusOccurrenceCount = "kafka_status_count"

	labelOperation = "operation"
)

// JobType metric to capture
type JobType string

var (
	// JobTypeClusterCreate - cluster_create job type
	JobTypeClusterCreate JobType = "cluster_create"
	// JobTypeKafkaCreate - kafka_create job type
	JobTypeKafkaCreate JobType = "kafka_create"
)

// JobsMetricsLabels is the slice of labels to add to job metrics
var JobsMetricsLabels = []string{
	labelJobType,
}

// KafkaStatusCountMetricsLabels - is the slice of labels to add to kafka status count metrics
var KafkaStatusCountMetricsLabels = []string{
	kafkaStatus,
	labelOperation,
}

// create a new histogramVec for cluster creation duration
var requestClusterCreationDurationMetric = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: jobsSubsystem,
		Name:      ClusterCreateRequestDuration,
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

// UpdateClusterCreationDurationMetric records the duration of a job type
func UpdateClusterCreationDurationMetric(jobType JobType, elapsed time.Duration) {
	labels := prometheus.Labels{
		labelJobType: string(jobType),
	}
	requestClusterCreationDurationMetric.With(labels).Observe(elapsed.Seconds())
}

// create a new histogramVec for kafka creation duration
var requestKafkaCreationDurationMetric = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: jobsSubsystem,
		Name:      KafkaCreateRequestDuration,
		Help:      "Kafka creation duration in seconds.",
		Buckets: []float64{
			1.0,
			30.0,
			60.0,
			120.0,
			150.0,
			180.0,
			210.0,
			240.0,
			270.0,
			300.0,
			330.0,
			360.0,
			390.0,
			420.0,
			450.0,
			480.0,
			510.0,
			540.0,
			570.0,
			600.0,
			1200.0,
		},
	},
	JobsMetricsLabels,
)

// UpdateKafkaCreationDurationMetric records the duration of a job type
func UpdateKafkaCreationDurationMetric(jobType JobType, elapsed time.Duration) {
	labels := prometheus.Labels{
		labelJobType: string(jobType),
	}
	requestKafkaCreationDurationMetric.With(labels).Observe(elapsed.Seconds())
}

// create a new counterVec for transaction counts
var requestTransactionCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: transactionsSubsystem,
		Name:      kafkaStatusOccurrenceCount,
		Help:      "Number of kafka requests in given status",
	},
	KafkaStatusCountMetricsLabels,
)

// IncreaseStatusCountMetric - increase counter for kafka request status
func IncreaseStatusCountMetric(status constants.KafkaStatus, operation constants.KafkaOperation) {
	labels := prometheus.Labels{
		kafkaStatus:    status.String(),
		labelOperation: operation.String(),
	}
	requestTransactionCountMetric.With(labels).Inc()
}

// register the metric(s)
func init() {
	prometheus.MustRegister(requestClusterCreationDurationMetric)
	prometheus.MustRegister(requestKafkaCreationDurationMetric)
	prometheus.MustRegister(requestTransactionCountMetric)
}
