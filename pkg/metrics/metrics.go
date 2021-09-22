package metrics

import (
	"strconv"
	"time"

	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/api"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	// FleetManager - metrics prefix
	FleetManager = "_yournamehere_fleet_manager"

	// ClusterCreateRequestDuration - name of cluster creation duration metric
	ClusterCreateRequestDuration = "worker_cluster_duration"
	// PineappleCreateRequestDuration - name of pineapple creation duration metric
	PineappleCreateRequestDuration = "worker_pineapple_duration"

	labelJobType           = "jobType"
	LabelID                = "id"
	LabelStatus            = "status"
	LabelClusterID         = "cluster_id"
	LabelClusterExternalID = "external_id"

	// PineappleOperationsSuccessCount - name of the metric for Pineapple-related successful operations
	PineappleOperationsSuccessCount = "pineapple_operations_success_count"
	// PineappleOperationsTotalCount - name of the metric for all Pineapple-related operations
	PineappleOperationsTotalCount = "pineapple_operations_total_count"

	// PineappleRequestsStatus - pineapple requests status metric
	PineappleRequestsStatusSinceCreated = "pineapple_requests_status_since_created_in_seconds"
	PineappleRequestsStatusCount        = "pineapple_requests_status_count"

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

	PineapplePerClusterCount = "pineapple_per_cluster_count"

	LeaderWorker = "leader_worker"

	// ObservatoriumRequestCount - metric name for the number of observatorium requests sent
	ObservatoriumRequestCount = "observatorium_request_count"
	// ObservatoriumRequestDuration - metric name for observatorium request duration in seconds
	ObservatoriumRequestDuration = "observatorium_request_duration"

	LabelStatusCode = "code"
	LabelMethod     = "method"
	LabelPath       = "path"
)

// JobType metric to capture
type JobType string

var (
	// JobTypeClusterCreate - cluster_create job type
	JobTypeClusterCreate JobType = "cluster_create"
	// JobTypePineappleCreate - pineapple_create job type
	JobTypePineappleCreate JobType = "pineapple_create"
)

// JobsMetricsLabels is the slice of labels to add to job metrics
var JobsMetricsLabels = []string{
	labelJobType,
}

// pineappleStatusSinceCreatedMetricLabels  is the slice of labels to add to
var pineappleStatusSinceCreatedMetricLabels = []string{
	LabelStatus,
	LabelID,
	LabelClusterID,
}

// pineappleStatusCountMetricLabels  is the slice of labels to add to
var pineappleStatusCountMetricLabels = []string{
	LabelStatus,
}

// PineappleOperationsCountMetricsLabels - is the slice of labels to add to Pineapple operations count metrics
var PineappleOperationsCountMetricsLabels = []string{
	labelOperation,
}

var PineapplePerClusterCountMetricsLabels = []string{
	LabelClusterID,
	LabelClusterExternalID,
}

// ClusterOperationsCountMetricsLabels - is the slice of labels to add to Pineapple operations count metrics
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

// #### Metrics for Dataplane clusters - Start ####
// create a new histogramVec for cluster creation duration
var requestClusterCreationDurationMetric = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: FleetManager,
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
		Subsystem: FleetManager,
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

// reate a new counterVec for total cluster operation counts
var clusterOperationsTotalCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: FleetManager,
		Name:      ClusterOperationsTotalCount,
		Help:      "number of total cluster operations",
	},
	ClusterOperationsCountMetricsLabels,
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
		Subsystem: FleetManager,
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
		Subsystem: FleetManager,
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

var pineapplePerClusterCountMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: FleetManager,
		Name:      PineapplePerClusterCount,
		Help:      "the number of Pineapple instances per data plane cluster",
	},
	PineapplePerClusterCountMetricsLabels)

func UpdatePineapplePerClusterCountMetric(clusterId string, clusterExternalID string, count int) {
	labels := prometheus.Labels{
		LabelClusterID:         clusterId,
		LabelClusterExternalID: clusterExternalID,
	}
	pineapplePerClusterCountMetric.With(labels).Set(float64(count))
}

// #### Metrics for Dataplane clusters - End ####

// #### Metrics for Pineapples - Start ####
// create a new histogramVec for pineapple creation duration
var requestPineappleCreationDurationMetric = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Subsystem: FleetManager,
		Name:      PineappleCreateRequestDuration,
		Help:      "Pineapple creation duration in seconds.",
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

// UpdatePineappleCreationDurationMetric records the duration of a job type
func UpdatePineappleCreationDurationMetric(jobType JobType, elapsed time.Duration) {
	labels := prometheus.Labels{
		labelJobType: string(jobType),
	}
	requestPineappleCreationDurationMetric.With(labels).Observe(elapsed.Seconds())
}

// create a new counterVec for Pineapple operations counts
var pineappleOperationsSuccessCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: FleetManager,
		Name:      PineappleOperationsSuccessCount,
		Help:      "number of successful pineapple operations",
	},
	PineappleOperationsCountMetricsLabels,
)

//UpdatePineappleRequestsStatusSinceCreatedMetric
func UpdatePineappleRequestsStatusSinceCreatedMetric(status constants2.DinosaurStatus, pineappleId string, clusterId string, elapsed time.Duration) {
	labels := prometheus.Labels{
		LabelStatus:    string(status),
		LabelID:        pineappleId,
		LabelClusterID: clusterId,
	}
	pineappleStatusSinceCreatedMetric.With(labels).Set(elapsed.Seconds())
}

//UpdatePineappleRequestsStatusCountMetric
func UpdatePineappleRequestsStatusCountMetric(status constants2.DinosaurStatus, count int) {
	labels := prometheus.Labels{
		LabelStatus: string(status),
	}
	PineappleStatusCountMetric.With(labels).Set(float64(count))
}

// create a new GaugeVec for status counts
var PineappleStatusCountMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: FleetManager,
		Name:      PineappleRequestsStatusCount,
		Help:      "number of total Pineapple instances in each status",
	},
	pineappleStatusCountMetricLabels,
)

// create a new GaugeVec for pineapples status duration
var pineappleStatusSinceCreatedMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: FleetManager,
		Name:      PineappleRequestsStatusSinceCreated,
		Help:      "metrics to track the status of a Pineapple instance and how long since it's been created",
	},
	pineappleStatusSinceCreatedMetricLabels,
)

// IncreasePineappleSuccessOperationsCountMetric - increase counter for the pineappleOperationsSuccessCountMetric
func IncreasePineappleSuccessOperationsCountMetric(operation constants2.DinosaurOperation) {
	labels := prometheus.Labels{
		labelOperation: operation.String(),
	}
	pineappleOperationsSuccessCountMetric.With(labels).Inc()
}

// create a new counterVec for total Pineapple operations counts
var pineappleOperationsTotalCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: FleetManager,
		Name:      PineappleOperationsTotalCount,
		Help:      "number of total pineapple operations",
	},
	PineappleOperationsCountMetricsLabels,
)

// IncreasePineappleTotalOperationsCountMetric - increase counter for the pineappleOperationsTotalCountMetric
func IncreasePineappleTotalOperationsCountMetric(operation constants2.DinosaurOperation) {
	labels := prometheus.Labels{
		labelOperation: operation.String(),
	}
	pineappleOperationsTotalCountMetric.With(labels).Inc()
}

// #### Metrics for Pineapples - End ####

// #### Metrics for Reconcilers - Start ####
// create a new gaugeVec for reconciler duration
var reconcilerDurationMetric = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Subsystem: FleetManager,
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
		Subsystem: FleetManager,
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
		Subsystem: FleetManager,
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
		Subsystem: FleetManager,
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
		Subsystem: FleetManager,
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
	Subsystem: FleetManager,
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
		Subsystem: FleetManager,
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

// register the metric(s)
func init() {
	// metrics for data plane clusters
	prometheus.MustRegister(requestClusterCreationDurationMetric)
	prometheus.MustRegister(clusterOperationsSuccessCountMetric)
	prometheus.MustRegister(clusterOperationsTotalCountMetric)
	prometheus.MustRegister(clusterStatusSinceCreatedMetric)
	prometheus.MustRegister(clusterStatusCountMetric)
	prometheus.MustRegister(pineapplePerClusterCountMetric)

	// metrics for Pineapples
	prometheus.MustRegister(requestPineappleCreationDurationMetric)
	prometheus.MustRegister(pineappleOperationsSuccessCountMetric)
	prometheus.MustRegister(pineappleOperationsTotalCountMetric)
	prometheus.MustRegister(pineappleStatusSinceCreatedMetric)
	prometheus.MustRegister(PineappleStatusCountMetric)

	// metrics for reconcilers
	prometheus.MustRegister(reconcilerDurationMetric)
	prometheus.MustRegister(reconcilerSuccessCountMetric)
	prometheus.MustRegister(reconcilerFailureCountMetric)
	prometheus.MustRegister(reconcilerErrorsCountMetric)
	prometheus.MustRegister(leaderWorkerMetric)

	// metrics for observatorium
	prometheus.MustRegister(observatoriumRequestCountMetric)
	prometheus.MustRegister(observatoriumRequestDurationMetric)
}

// ResetMetricsForPineappleManagers will reset the metrics for the PineappleManager background reconciler
// This is needed because if current process is not the leader anymore, the metrics need to be reset otherwise staled data will be scraped
func ResetMetricsForPineappleManagers() {
	pineappleStatusSinceCreatedMetric.Reset()
	PineappleStatusCountMetric.Reset()
}

// ResetMetricsForClusterManagers will reset the metrics for the ClusterManager background reconciler
// This is needed because if current process is not the leader anymore, the metrics need to be reset otherwise staled data will be scraped
func ResetMetricsForClusterManagers() {
	clusterStatusSinceCreatedMetric.Reset()
	clusterStatusCountMetric.Reset()
	pineapplePerClusterCountMetric.Reset()
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
	pineapplePerClusterCountMetric.Reset()

	requestPineappleCreationDurationMetric.Reset()
	pineappleOperationsSuccessCountMetric.Reset()
	pineappleOperationsTotalCountMetric.Reset()
	pineappleStatusSinceCreatedMetric.Reset()
	PineappleStatusCountMetric.Reset()

	reconcilerDurationMetric.Reset()
	reconcilerSuccessCountMetric.Reset()
	reconcilerFailureCountMetric.Reset()
	reconcilerErrorsCountMetric.Reset()
	leaderWorkerMetric.Reset()

	ResetMetricsForObservatorium()
}
