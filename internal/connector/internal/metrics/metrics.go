package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// CosFleetManager - metrics prefix
	CosFleetManager = "cos_fleet_manager"

	// label for operation name
	labelOperation = "operation"

	VaultServiceTotalCount   = "vault_service_total_count"
	VaultServiceSuccessCount = "vault_service_success_count"
	VaultServiceFailureCount = "vault_service_failure_count"
	VaultServiceErrorsCount  = "vault_service_errors_count"
)

var VaultServiceMetricsLabels = []string{
	labelOperation,
}

// #### Metrics for Vault Service ####

var vaultServiceTotalCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: CosFleetManager,
		Name:      VaultServiceTotalCount,
		Help:      "total count of operations since start of vault service",
	}, VaultServiceMetricsLabels)

func IncreaseVaultServiceTotalCount(operationType string) {
	labels := prometheus.Labels{
		labelOperation: operationType,
	}
	vaultServiceTotalCountMetric.With(labels).Inc()
}

var vaultServiceSuccessCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: CosFleetManager,
		Name:      VaultServiceSuccessCount,
		Help:      "count of successful operations of vault service",
	}, VaultServiceMetricsLabels)

func IncreaseVaultServiceSuccessCount(operationType string) {
	labels := prometheus.Labels{
		labelOperation: operationType,
	}
	vaultServiceSuccessCountMetric.With(labels).Inc()
}

var vaultServiceFailureCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: CosFleetManager,
		Name:      VaultServiceFailureCount,
		Help:      "count of system failures (e.g. connectivity issues) in the vault service",
	}, VaultServiceMetricsLabels)

func IncreaseVaultServiceFailureCount(operationType string) {
	labels := prometheus.Labels{
		labelOperation: operationType,
	}
	vaultServiceFailureCountMetric.With(labels).Inc()
}

var vaultServiceErrorsCountMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Subsystem: CosFleetManager,
		Name:      VaultServiceErrorsCount,
		Help:      "count of user level errors (e.g. missing secrets) in the vault service",
	}, VaultServiceMetricsLabels)

func IncreaseVaultServiceErrorsCount(operationType string) {
	labels := prometheus.Labels{
		labelOperation: operationType,
	}
	vaultServiceErrorsCountMetric.With(labels).Inc()
}

// #### Metrics for Vault Service - End ####

// register the metric(s)
func init() {
	// metrics for vault service
	prometheus.MustRegister(vaultServiceTotalCountMetric)
	prometheus.MustRegister(vaultServiceSuccessCountMetric)
	prometheus.MustRegister(vaultServiceFailureCountMetric)
	prometheus.MustRegister(vaultServiceErrorsCountMetric)
}

// ResetMetricsForVaultService will reset the metrics related to Vault Service requests
// This is needed because if current process is not the leader anymore, the metrics need to be reset otherwise staled data will be scraped
func ResetMetricsForVaultService() {
	vaultServiceTotalCountMetric.Reset()
	vaultServiceSuccessCountMetric.Reset()
	vaultServiceFailureCountMetric.Reset()
	vaultServiceErrorsCountMetric.Reset()
}

// Reset the metrics we have defined. It is mainly used for testing.
func Reset() {
	ResetMetricsForVaultService()
}
