package metrics

import (
	"strconv"
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// KasFleetManager - metrics prefix
	KasFleetManager = "kas_fleet_manager"

	// ClusterCreateRequestDuration - name of cluster creation duration metric
	ClusterCreateRequestDuration = "worker_cluster_duration"
	// KafkaCreateRequestDuration - name of kafka creation duration metric
	KafkaCreateRequestDuration = "worker_kafka_duration"

	labelJobType           = "jobType"
	LabelID                = "id"
	LabelStatus            = "status"
	LabelClusterID         = "cluster_id"
	LabelClusterExternalID = "external_id"

	// KafkaOperationsSuccessCount - name of the metric for Kafka-related successful operations
	KafkaOperationsSuccessCount = "kafka_operations_success_count"
	// KafkaOperationsTotalCount - name of the metric for all Kafka-related operations
	KafkaOperationsTotalCount = "kafka_operations_total_count"

	// KafkaRequestsStatus - kafka requests status metric
	KafkaRequestsStatusSinceCreated = "kafka_requests_status_since_created_in_seconds"
	KafkaRequestsStatusCount        = "kafka_requests_status_count"

	// ClusterOperationsSuccessCount - name of the metric for cluster-related successful operations
	ClusterOperationsSuccessCount = "cluster_operations_success_count"
	// ClusterOperationsTotalCount - name of the metric for all cluster-related operations
	ClusterOperationsTotalCount = "cluster_operations_total_count"
	labelOperation              = "operation"

	ReconcilerDuration     = "reconciler_duration_in_seconds"
	ReconcilerSuccessCount = "reconciler_success_count"
	ReconcilerFailureCount = "reconciler_failure_count"
	ReconcilerErrorsCount  = "reconciler_errors_count"
	labelWorkerType        = "worker_type"

	ClusterStatusSinceCreated = "cluster_status_since_created_in_seconds"
	ClusterStatusCount        = "cluster_status_count"

	KafkaPerClusterCount = "kafka_per_cluster_count"

	LeaderWorker = "leader_worker"

	// ObservatoriumRequestCount - metric name for the number of observatorium requests sent
	ObservatoriumRequestCount = "observatorium_request_count"
	// ObservatoriumRequestDuration - metric name for observatorium request duration in seconds
	ObservatoriumRequestDuration = "observatorium_request_duration"

	// DatabaseQueryCount - metric name for the number of database query sent
	DatabaseQueryCount = "database_query_count"
	// DatabaseQueryDuration - metric name for database query duration in milliseconds
	DatabaseQueryDuration = "database_query_duration"

	// ClusterStatusMaxCapacity - metric name for the maximum kafka instance capacity
	ClusterStatusCapacityMax = "cluster_status_capacity_max"

	// ClusterStatusCapacityUsed - metric name for the current number of instances
	ClusterStatusCapacityUsed = "cluster_status_capacity_used"

	// ClusterStatusCapacityAvailable - metric name for the number of available instances
	ClusterStatusCapacityAvailable = "cluster_status_capacity_available"

	LabelStatusCode = "code"
	LabelMethod     = "method"
	LabelPath       = "path"

	LabelDatabaseQueryStatus = "status"
	LabelDatabaseQueryType   = "query"
	LabelRegion              = "region"
	LabelInstanceType        = "instance_type"
	LabelCloudProvider       = "cloud_provider"
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

// kafkaStatusSinceCreatedMetricLabels  is the slice of labels to add to
var kafkaStatusSinceCreatedMetricLabels = []string{
	LabelStatus,
	LabelID,
	LabelClusterID,
}

// kafkaStatusCountMetricLabels  is the slice of labels to add to
var kafkaStatusCountMetricLabels = []string{
	LabelStatus,
}

// KafkaOperationsCountMetricsLabels - is the slice of labels to add to Kafka operations count metrics
var KafkaOperationsCountMetricsLabels = []string{
	labelOperation,
}

var KafkaPerClusterCountMetricsLabels = []string{
	LabelClusterID,
	LabelClusterExternalID,
}

// ClusterOperationsCountMetricsLabels - is the slice of labels to add to Kafka operations count metrics
var ClusterOperationsCountMetricsLabels = []string{
	labelOperation,
}

var ClusterStatusSinceCreatedMetricsLabels = []string{
	LabelID,
	LabelClusterID,
	LabelStatus,
}

var ClusterStatusCountMetricsLabels = []string{
	LabelStatus,
}

var ReconcilerMetricsLabels = []string{
	labelWorkerType,
}

var observatoriumRequestMetricsLabels = []string{
	LabelStatusCode,
	LabelMethod,
	LabelPath,
}

var DatabaseMetricsLabels = []string{
	LabelDatabaseQueryStatus,
	LabelDatabaseQueryType,
}

var clusterStatusCapacityLabels = []string{
	LabelRegion,
	LabelInstanceType,
	LabelClusterID,
	LabelCloudProvider,
}

// #### Metrics for Dataplane clusters - Start ####
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
func IncreaseClusterSuccessOperationsCountMetric(operation constants2.ClusterOperation) {
	labels := prometheus.Labels{
		labelOperation: operation.String(),
	}
	clusterOperationsSuccessCountMetric.With(labels).Inc()
}

// UpdateClusterStatusCapacityMaxCount - sets maximum capacity per region and instance type
func UpdateClusterStatusCapacityMaxCount(provider string, region, instanceType, clusterId string, count float64) {
	labels := prometheus.Labels{
		LabelRegion:        region,
		LabelInstanceType:  instanceType,
		LabelClusterID:     clusterId,
		LabelCloudProvider: provider,
	}
	clusterStatusCapacityMaxMetric.With(labels).Set(count)
}

// UpdateClusterStatusCapacityUsedCount - sets used capacity per region and instance type
func UpdateClusterStatusCapacityUsedCount(provider string, region, instanceType, clusterId string, count float64) {
	labels := prometheus.Labels{
		LabelRegion:        region,
		LabelInstanceType:  instanceType,
		LabelClusterID:     clusterId,
		LabelCloudProvider: provider,
	}
	clusterStatusCapacityUsedMetric.With(labels).Set(count)
}

// UpdateClusterStatusCapacityAvailableCount - sets used capacity per region and instance type
func UpdateClusterStatusCapacityAvailableCount(provider string, region, instanceType, clusterId string, count float64) {
	labels := prometheus.Labels{
		LabelRegion:        region,
		LabelInstanceType:  instanceType,
		LabelClusterID:     clusterId,
		LabelCloudProvider: provider,
	}
	clusterStatusCapacityAvailableMetric.With(labels).Set(count)
}

// create a new counterVec for total cluster operation counts
var clusterOperationsTotalCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: KasFleetManager,
		Name:      ClusterOperationsTotalCount,
		Help:      "number of total cluster operations",
	},
	ClusterOperationsCountMetricsLabels,
)

// create a new gaugeVec for the maximum kafka instance capacity per region
var clusterStatusCapacityMaxMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: KasFleetManager,
		Name:      ClusterStatusCapacityMax,
		Help:      "number of allowed Streaming Units per region and kafka instance type",
	},
	clusterStatusCapacityLabels,
)

// create a new gauge vec fot the number of kafka instances grouped by region and instance type
var clusterStatusCapacityUsedMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: KasFleetManager,
		Name:      ClusterStatusCapacityUsed,
		Help:      "number of Streaming Units consumed by existing instances per region and kafka instance type",
	},
	clusterStatusCapacityLabels,
)

// create a new gauge vec fot the number of kafka instances grouped by region and instance type
var clusterStatusCapacityAvailableMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: KasFleetManager,
		Name:      ClusterStatusCapacityAvailable,
		Help:      "number of available Streaming Units per region and kafka instance type",
	},
	clusterStatusCapacityLabels,
)

// IncreaseClusterTotalOperationsCountMetric - increase counter for clusterOperationsTotalCountMetric
func IncreaseClusterTotalOperationsCountMetric(operation constants2.ClusterOperation) {
	labels := prometheus.Labels{
		labelOperation: operation.String(),
	}
	clusterOperationsTotalCountMetric.With(labels).Inc()
}

// create a new GaugeVec for cluster status since created
var clusterStatusSinceCreatedMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: KasFleetManager,
		Name:      ClusterStatusSinceCreated,
		Help:      "metrics to track the status of a dataplane cluster and how long since it's been created",
	},
	ClusterStatusSinceCreatedMetricsLabels,
)

func UpdateClusterStatusSinceCreatedMetric(cluster api.Cluster, status api.ClusterStatus) {
	labels := prometheus.Labels{
		LabelStatus:    string(status),
		LabelID:        cluster.ID,
		LabelClusterID: cluster.ClusterID,
	}
	clusterStatusSinceCreatedMetric.With(labels).Set(time.Since(cluster.CreatedAt).Seconds())
}

var clusterStatusCountMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: KasFleetManager,
		Name:      ClusterStatusCount,
		Help:      "metrics to record the number of Dataplane clusters in each status",
	},
	ClusterStatusCountMetricsLabels,
)

func UpdateClusterStatusCountMetric(status api.ClusterStatus, count int) {
	labels := prometheus.Labels{
		LabelStatus: string(status),
	}
	clusterStatusCountMetric.With(labels).Set(float64(count))
}

var kafkaPerClusterCountMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: KasFleetManager,
		Name:      KafkaPerClusterCount,
		Help:      "the number of Kafka instances per data plane cluster",
	},
	KafkaPerClusterCountMetricsLabels)

func UpdateKafkaPerClusterCountMetric(clusterId string, clusterExternalID string, count int) {
	labels := prometheus.Labels{
		LabelClusterID:         clusterId,
		LabelClusterExternalID: clusterExternalID,
	}
	kafkaPerClusterCountMetric.With(labels).Set(float64(count))
}

// #### Metrics for Dataplane clusters - End ####

// #### Metrics for Kafkas - Start ####
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
			900.0,
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

//UpdateKafkaRequestsStatusSinceCreatedMetric
func UpdateKafkaRequestsStatusSinceCreatedMetric(status constants2.KafkaStatus, kafkaId string, clusterId string, elapsed time.Duration) {
	labels := prometheus.Labels{
		LabelStatus:    string(status),
		LabelID:        kafkaId,
		LabelClusterID: clusterId,
	}
	kafkaStatusSinceCreatedMetric.With(labels).Set(elapsed.Seconds())
}

//UpdateKafkaRequestsStatusCountMetric
func UpdateKafkaRequestsStatusCountMetric(status constants2.KafkaStatus, count int) {
	labels := prometheus.Labels{
		LabelStatus: string(status),
	}
	KafkaStatusCountMetric.With(labels).Set(float64(count))
}

// create a new GaugeVec for status counts
var KafkaStatusCountMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: KasFleetManager,
		Name:      KafkaRequestsStatusCount,
		Help:      "number of total Kafka instances in each status",
	},
	kafkaStatusCountMetricLabels,
)

// create a new GaugeVec for kafkas status duration
var kafkaStatusSinceCreatedMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: KasFleetManager,
		Name:      KafkaRequestsStatusSinceCreated,
		Help:      "metrics to track the status of a Kafka instance and how long since it's been created",
	},
	kafkaStatusSinceCreatedMetricLabels,
)

// IncreaseKafkaSuccessOperationsCountMetric - increase counter for the kafkaOperationsSuccessCountMetric
func IncreaseKafkaSuccessOperationsCountMetric(operation constants2.KafkaOperation) {
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
func IncreaseKafkaTotalOperationsCountMetric(operation constants2.KafkaOperation) {
	labels := prometheus.Labels{
		labelOperation: operation.String(),
	}
	kafkaOperationsTotalCountMetric.With(labels).Inc()
}

// #### Metrics for Kafkas - End ####

// #### Metrics for Reconcilers - Start ####
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
		labelWorkerType: reconcilerType,
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
		labelWorkerType: reconcilerType,
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
		labelWorkerType: reconcilerType,
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
		labelWorkerType: reconcilerType,
	}
	reconcilerErrorsCountMetric.With(labels).Add(float64(numOfErr))
}

var leaderWorkerMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: KasFleetManager,
		Name:      LeaderWorker,
		Help:      "metrics to indicate if the current process is the leader among the workers",
	}, ReconcilerMetricsLabels)

// SetLeaderWorkerMetric will set the metric value to 1 if the worker is the leader, and 0 if the worker is not the leader.
// Then when the metrics is scraped, Prometheus will add additional information like pod name, which then can be used to display which pod is the leader.
func SetLeaderWorkerMetric(workerType string, leader bool) {
	labels := prometheus.Labels{
		labelWorkerType: workerType,
	}
	val := 0
	if leader {
		val = 1
	}
	leaderWorkerMetric.With(labels).Set(float64(val))
}

// #### Metrics for Reconcilers - End ####

// #### Metrics for Observatorium ####

// register observatorium request count metric
//	  observatorium_request_count - Number of Observatorium requests sent partitioned by http status code, method and url path
var observatoriumRequestCountMetric = prometheus.NewCounterVec(prometheus.CounterOpts{
	Subsystem: KasFleetManager,
	Name:      ObservatoriumRequestCount,
	Help:      "number of requests sent to Observatorium. If no response was received, the value of code should be '0' (this can happen on request timeout or failure to connect to Observatorium).",
}, observatoriumRequestMetricsLabels)

// Increase the observatorium request count metric with the following labels:
// 	- code: HTTP Status code (i.e. 200 or 500)
// 	- path: Request URL path (i.e. /api/v1/query)
// 	- method: HTTP Method (i.e. GET or POST)
func IncreaseObservatoriumRequestCount(code int, path, method string) {
	labels := prometheus.Labels{
		LabelStatusCode: strconv.Itoa(code),
		LabelPath:       path,
		LabelMethod:     method,
	}
	observatoriumRequestCountMetric.With(labels).Inc()
}

// register observatorium request duration metric. Each metric is partitioned by http status code, method and url path
//	 observatorium_request_duration_sum - Total time to send requests to Observatorium in seconds.
//	 observatorium_request_duration_count - Total number of Observatorium requests measured.
//	 observatorium_request_duration_bucket - Number of Observatorium requests organized in buckets.
var observatoriumRequestDurationMetric = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: KasFleetManager,
		Name:      ObservatoriumRequestDuration,
		Help:      `Observatorium request duration in seconds. If no response was received, the value of code should be '0' (this can happen on request timeout or failure to connect to Observatorium).`,
		Buckets: []float64{
			0.1,
			1.0,
			10.0,
			30.0,
		},
	},
	observatoriumRequestMetricsLabels,
)

// Update the observatorium request duration metric with the following labels:
// 	- code: HTTP Status code (i.e. 200 or 500)
// 	- path: Request url path (i.e. /api/v1/query)
// 	- method: HTTP Method (i.e. GET or POST)
func UpdateObservatoriumRequestDurationMetric(code int, path, method string, elapsed time.Duration) {
	labels := prometheus.Labels{
		LabelStatusCode: strconv.Itoa(code),
		LabelPath:       path,
		LabelMethod:     method,
	}
	observatoriumRequestDurationMetric.With(labels).Observe(elapsed.Seconds())
}

// #### Metrics for Observatorium - End ####

// #### Metrics for Database ####

// register database query count metric
//	  database_query_count - Number of Database query sent partitioned by status, and sql query type
var databaseRequestCountMetric = prometheus.NewCounterVec(prometheus.CounterOpts{
	Subsystem: KasFleetManager,
	Name:      DatabaseQueryCount,
	Help:      "number of query sent to Dataase.",
}, DatabaseMetricsLabels)

// Increase the database query count metric with the following labels:
// 	- status: (i.e. "success" or "failure")
// 	- queryType: (i.e. "SELECT", "UPDATE", "INSERT", "DELETE")
func IncreaseDatabaseQueryCount(status string, queryType string) {
	labels := prometheus.Labels{
		LabelDatabaseQueryStatus: status,
		LabelDatabaseQueryType:   queryType,
	}
	databaseRequestCountMetric.With(labels).Inc()
}

// register database query duration metric. Each metric is partitioned by status, query type
//	 database_query_duration_sum - Total time to send requests to Database in milliseconds.
//	 database_query_duration_count - Total number of database query measured.
//	 database_query_duration_bucket - Number of Database queries organized in buckets.
var databaseQueryDurationMetric = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: KasFleetManager,
		Name:      DatabaseQueryDuration,
		Help:      `Database query duration in milliseconds.`,
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
			900.0,
			1200.0,
			1800.0,
			2400.0,
			3600.0,
			4800.0,
			7200.0,
		},
	},
	DatabaseMetricsLabels,
)

// Update the observatorium request duration metric with the following labels:
// 	- status: (i.e. "success" or "failure")
// 	- queryType: (i.e. "SELECT", "UPDATE", "INSERT", "DELETE")
func UpdateDatabaseQueryDurationMetric(status string, queryType string, elapsed time.Duration) {
	labels := prometheus.Labels{
		LabelDatabaseQueryStatus: status,
		LabelDatabaseQueryType:   queryType,
	}
	databaseQueryDurationMetric.With(labels).Observe(float64(elapsed.Milliseconds()))
}

// #### Metrics for Database - End ####

// register the metric(s)
func init() {
	// metrics for data plane clusters
	prometheus.MustRegister(requestClusterCreationDurationMetric)
	prometheus.MustRegister(clusterOperationsSuccessCountMetric)
	prometheus.MustRegister(clusterOperationsTotalCountMetric)
	prometheus.MustRegister(clusterStatusSinceCreatedMetric)
	prometheus.MustRegister(clusterStatusCountMetric)
	prometheus.MustRegister(kafkaPerClusterCountMetric)
	prometheus.MustRegister(clusterStatusCapacityMaxMetric)
	prometheus.MustRegister(clusterStatusCapacityUsedMetric)
	prometheus.MustRegister(clusterStatusCapacityAvailableMetric)

	// metrics for Kafkas
	prometheus.MustRegister(requestKafkaCreationDurationMetric)
	prometheus.MustRegister(kafkaOperationsSuccessCountMetric)
	prometheus.MustRegister(kafkaOperationsTotalCountMetric)
	prometheus.MustRegister(kafkaStatusSinceCreatedMetric)
	prometheus.MustRegister(KafkaStatusCountMetric)

	// metrics for reconcilers
	prometheus.MustRegister(reconcilerDurationMetric)
	prometheus.MustRegister(reconcilerSuccessCountMetric)
	prometheus.MustRegister(reconcilerFailureCountMetric)
	prometheus.MustRegister(reconcilerErrorsCountMetric)
	prometheus.MustRegister(leaderWorkerMetric)

	// metrics for observatorium
	prometheus.MustRegister(observatoriumRequestCountMetric)
	prometheus.MustRegister(observatoriumRequestDurationMetric)

	// metrics for database
	prometheus.MustRegister(databaseRequestCountMetric)
	prometheus.MustRegister(databaseQueryDurationMetric)
}

// ResetMetricsForKafkaManagers will reset the metrics for the KafkaManager background reconciler
// This is needed because if current process is not the leader anymore, the metrics need to be reset otherwise staled data will be scraped
func ResetMetricsForKafkaManagers() {
	kafkaStatusSinceCreatedMetric.Reset()
	KafkaStatusCountMetric.Reset()
}

// ResetMetricsForClusterManagers will reset the metrics for the ClusterManager background reconciler
// This is needed because if current process is not the leader anymore, the metrics need to be reset otherwise staled data will be scraped
func ResetMetricsForClusterManagers() {
	clusterStatusSinceCreatedMetric.Reset()
	clusterStatusCountMetric.Reset()
	kafkaPerClusterCountMetric.Reset()
	clusterStatusCapacityMaxMetric.Reset()
	clusterStatusCapacityUsedMetric.Reset()
	clusterStatusCapacityAvailableMetric.Reset()
}

// ResetMetricsForReconcilers will reset the metrics related to the reconcilers
// This is needed because if current process is not the leader anymore, the metrics need to be reset otherwise staled data will be scraped
func ResetMetricsForReconcilers() {
	reconcilerDurationMetric.Reset()
	reconcilerSuccessCountMetric.Reset()
	reconcilerFailureCountMetric.Reset()
	reconcilerErrorsCountMetric.Reset()
}

// ResetMetricsForObservatorium will reset the metrics related to Observatorium requests
// This is needed because if current process is not the leader anymore, the metrics need to be reset otherwise staled data will be scraped
func ResetMetricsForObservatorium() {
	observatoriumRequestCountMetric.Reset()
	observatoriumRequestDurationMetric.Reset()
}

// Reset the metrics we have defined. It is mainly used for testing.
func Reset() {
	requestClusterCreationDurationMetric.Reset()
	clusterOperationsSuccessCountMetric.Reset()
	clusterOperationsTotalCountMetric.Reset()
	clusterStatusSinceCreatedMetric.Reset()
	clusterStatusCountMetric.Reset()
	kafkaPerClusterCountMetric.Reset()
	clusterStatusCapacityMaxMetric.Reset()
	clusterStatusCapacityUsedMetric.Reset()
	clusterStatusCapacityAvailableMetric.Reset()

	requestKafkaCreationDurationMetric.Reset()
	kafkaOperationsSuccessCountMetric.Reset()
	kafkaOperationsTotalCountMetric.Reset()
	kafkaStatusSinceCreatedMetric.Reset()
	KafkaStatusCountMetric.Reset()

	reconcilerDurationMetric.Reset()
	reconcilerSuccessCountMetric.Reset()
	reconcilerFailureCountMetric.Reset()
	reconcilerErrorsCountMetric.Reset()
	leaderWorkerMetric.Reset()

	ResetMetricsForObservatorium()

	databaseRequestCountMetric.Reset()
	databaseQueryDurationMetric.Reset()
}
