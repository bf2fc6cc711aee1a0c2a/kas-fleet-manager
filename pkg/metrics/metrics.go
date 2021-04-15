package metrics

import (
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/constants"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// KasFleetManager - metrics prefix
	KasFleetManager = "kas_fleet_manager"

	// ClusterCreateRequestDuration - name of cluster creation duration metric
	ClusterCreateRequestDuration = "worker_cluster_duration"
	// KafkaCreateRequestDuration - name of kafka creation duration metric
	KafkaCreateRequestDuration = "worker_kafka_duration"

	labelJobType = "jobType"

	// KafkaOperationsSuccessCount - name of the metric for Kafka-related successful operations
	KafkaOperationsSuccessCount = "kafka_operations_success_count"
	// KafkaOperationsTotalCount - name of the metric for all Kafka-related operations
	KafkaOperationsTotalCount = "kafka_operations_total_count"

	// KafkaRequestsStatus - kafka requests status metric
	KafkaRequestsStatusDuration = "kafka_requests_status_duration"
	KafkaRequestsStatusCount    = "kafka_requests_status_count"
	LabelKafkaStatus            = "status"
	LabelKafkaRequestID         = "id"
	LabelKafkaClusterID         = "cluster_id"

	// ClusterOperationsSuccessCount - name of the metric for cluster-related successful operations
	ClusterOperationsSuccessCount = "cluster_operations_success_count"
	// ClusterOperationsTotalCount - name of the metric for all cluster-related operations
	ClusterOperationsTotalCount = "cluster_operations_total_count"
	labelOperation              = "operation"

	ReconcilerDuration     = "reconciler_duration_in_seconds"
	ReconcilerSuccessCount = "reconciler_success_count"
	ReconcilerFailureCount = "reconciler_failure_count"
	ReconcilerErrorsCount  = "reconciler_errors_count"
	labelReconcilerType    = "worker_type"
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

// kafkaStatusDurationMetricLabels  is the slice of labels to add to
var kafkaStatusDurationMetricLabels = []string{
	LabelKafkaStatus,
	LabelKafkaRequestID,
	LabelKafkaClusterID,
}

// kafkaStatusCountMetricLabels  is the slice of labels to add to
var kafkaStatusCountMetricLabels = []string{
	LabelKafkaStatus,
}

// KafkaOperationsCountMetricsLabels - is the slice of labels to add to Kafka operations count metrics
var KafkaOperationsCountMetricsLabels = []string{
	labelOperation,
}

// ClusterOperationsCountMetricsLabels - is the slice of labels to add to Kafka operations count metrics
var ClusterOperationsCountMetricsLabels = []string{
	labelOperation,
}

var ReconcilerMetricsLabels = []string{
	labelReconcilerType,
}

// create a new histogramVec for cluster creation duration
var requestClusterCreationDurationMetric = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: KasFleetManager,
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
		Subsystem: KasFleetManager,
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
			1800.0,
			2400.0,
			3600.0,
			4800.0,
			7200.0,
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

// create a new counterVec for Kafka operations counts
var kafkaOperationsSuccessCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: KasFleetManager,
		Name:      KafkaOperationsSuccessCount,
		Help:      "number of successful kafka operations",
	},
	KafkaOperationsCountMetricsLabels,
)

//KafkaRequestsStatusDurationMetric
func KafkaRequestsStatusDurationMetric(status constants.KafkaStatus, kafkaId string, clusterId string, elapsed time.Duration) {
	labels := prometheus.Labels{
		LabelKafkaStatus:    string(status),
		LabelKafkaRequestID: kafkaId,
		LabelKafkaClusterID: clusterId,
	}
	kafkaStatusDurationMetric.With(labels).Set(elapsed.Seconds())
}

//KafkaRequestsStatusCountMetric
func KafkaRequestsStatusCountMetric(status constants.KafkaStatus, count int) {
	labels := prometheus.Labels{
		LabelKafkaStatus: string(status),
	}
	KafkaStatusCountMetric.With(labels).Set(float64(count))
}

// create a new GaugeVec for status counts
var KafkaStatusCountMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: KasFleetManager,
		Name:      KafkaRequestsStatusCount,
		Help:      "number of total Kafka instances in each status ",
	},
	kafkaStatusCountMetricLabels,
)

// create a new GaugeVec for kafkas status duration
var kafkaStatusDurationMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: KasFleetManager,
		Name:      KafkaRequestsStatusDuration,
		Help:      "metrics to track the Kafka instances in various status and their duration",
	},
	kafkaStatusDurationMetricLabels,
)

// IncreaseKafkaSuccessOperationsCountMetric - increase counter for the kafkaOperationsSuccessCountMetric
func IncreaseKafkaSuccessOperationsCountMetric(operation constants.KafkaOperation) {
	labels := prometheus.Labels{
		labelOperation: operation.String(),
	}
	kafkaOperationsSuccessCountMetric.With(labels).Inc()
}

// create a new counterVec for total Kafka operations counts
var kafkaOperationsTotalCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: KasFleetManager,
		Name:      KafkaOperationsTotalCount,
		Help:      "number of total kafka operations",
	},
	KafkaOperationsCountMetricsLabels,
)

// IncreaseKafkaTotalOperationsCountMetric - increase counter for the kafkaOperationsTotalCountMetric
func IncreaseKafkaTotalOperationsCountMetric(operation constants.KafkaOperation) {
	labels := prometheus.Labels{
		labelOperation: operation.String(),
	}
	kafkaOperationsTotalCountMetric.With(labels).Inc()
}

// create a new counterVec for successful cluster operation counts
var clusterOperationsSuccessCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: KasFleetManager,
		Name:      ClusterOperationsSuccessCount,
		Help:      "number of successful cluster operations",
	},
	ClusterOperationsCountMetricsLabels,
)

// IncreaseClusterSuccessOperationsCountMetric - increase counter for clusterOperationsSuccessCountMetric
func IncreaseClusterSuccessOperationsCountMetric(operation constants.ClusterOperation) {
	labels := prometheus.Labels{
		labelOperation: operation.String(),
	}
	clusterOperationsSuccessCountMetric.With(labels).Inc()
}

// reate a new counterVec for total cluster operation counts
var clusterOperationsTotalCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: KasFleetManager,
		Name:      ClusterOperationsTotalCount,
		Help:      "number of total cluster operations",
	},
	ClusterOperationsCountMetricsLabels,
)

// IncreaseClusterTotalOperationsCountMetric - increase counter for clusterOperationsTotalCountMetric
func IncreaseClusterTotalOperationsCountMetric(operation constants.ClusterOperation) {
	labels := prometheus.Labels{
		labelOperation: operation.String(),
	}
	clusterOperationsTotalCountMetric.With(labels).Inc()
}

// create a new gaugeVec for reconciler duration
var reconcilerDurationMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: KasFleetManager,
		Name:      ReconcilerDuration,
		Help:      "Duration of each background reconcile in seconds.",
	},
	ReconcilerMetricsLabels,
)

func UpdateReconcilerDurationMetric(reconcilerType string, elapsed time.Duration) {
	labels := prometheus.Labels{
		labelReconcilerType: reconcilerType,
	}
	reconcilerDurationMetric.With(labels).Set(float64(elapsed.Seconds()))
}

var reconcilerSuccessCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: KasFleetManager,
		Name:      ReconcilerSuccessCount,
		Help:      "count of success operations of the backgroup reconcilers",
	}, ReconcilerMetricsLabels)

func IncreaseReconcilerSuccessCount(reconcilerType string) {
	labels := prometheus.Labels{
		labelReconcilerType: reconcilerType,
	}
	reconcilerSuccessCountMetric.With(labels).Inc()
}

var reconcilerFailureCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: KasFleetManager,
		Name:      ReconcilerFailureCount,
		Help:      "count of failed operations of the backgroup reconcilers",
	}, ReconcilerMetricsLabels)

func IncreaseReconcilerFailureCount(reconcilerType string) {
	labels := prometheus.Labels{
		labelReconcilerType: reconcilerType,
	}
	reconcilerFailureCountMetric.With(labels).Inc()
}

var reconcilerErrorsCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: KasFleetManager,
		Name:      ReconcilerErrorsCount,
		Help:      "count of errors occured during backgroup reconcilers runs",
	}, ReconcilerMetricsLabels)

func IncreaseReconcilerErrorsCount(reconcilerType string, numOfErr int) {
	labels := prometheus.Labels{
		labelReconcilerType: reconcilerType,
	}
	reconcilerErrorsCountMetric.With(labels).Add(float64(numOfErr))
}

// register the metric(s)
func init() {
	prometheus.MustRegister(requestClusterCreationDurationMetric)
	prometheus.MustRegister(requestKafkaCreationDurationMetric)
	prometheus.MustRegister(kafkaOperationsSuccessCountMetric)
	prometheus.MustRegister(kafkaOperationsTotalCountMetric)
	prometheus.MustRegister(clusterOperationsSuccessCountMetric)
	prometheus.MustRegister(clusterOperationsTotalCountMetric)
	prometheus.MustRegister(kafkaStatusDurationMetric)
	prometheus.MustRegister(reconcilerDurationMetric)
	prometheus.MustRegister(reconcilerSuccessCountMetric)
	prometheus.MustRegister(reconcilerFailureCountMetric)
	prometheus.MustRegister(reconcilerErrorsCountMetric)
	prometheus.MustRegister(KafkaStatusCountMetric)
}

// Reset the metrics we have defined. It is mainly used for testing.
func Reset() {
	requestClusterCreationDurationMetric.Reset()
	requestKafkaCreationDurationMetric.Reset()
	kafkaOperationsSuccessCountMetric.Reset()
	kafkaOperationsTotalCountMetric.Reset()
	clusterOperationsSuccessCountMetric.Reset()
	clusterOperationsTotalCountMetric.Reset()
	kafkaStatusDurationMetric.Reset()
	reconcilerDurationMetric.Reset()
	reconcilerSuccessCountMetric.Reset()
	reconcilerFailureCountMetric.Reset()
	reconcilerErrorsCountMetric.Reset()
	KafkaStatusCountMetric.Reset()
}
