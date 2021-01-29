package constants

import "time"

// KafkaStatus type
type KafkaStatus string

// KafkaOperation type
type KafkaOperation string

const (
	// KafkaRequestStatusAccepted - kafka request status when accepted by kafka worker
	KafkaRequestStatusAccepted KafkaStatus = "accepted"
	// KafkaRequestStatusPreparing - kafka request status of a preparing kafka
	KafkaRequestStatusPreparing KafkaStatus = "preparing"
	// KafkaRequestStatusProvisioning - kafka in provisioning state
	KafkaRequestStatusProvisioning KafkaStatus = "provisioning"
	// KafkaRequestStatusReady - completed kafka request
	KafkaRequestStatusReady KafkaStatus = "ready"
	// KafkaRequestStatusFailed - kafka request failed
	KafkaRequestStatusFailed KafkaStatus = "failed"
	// KafkaRequestStatusDeprovision - kafka request status when to be deleted by kafka
	KafkaRequestStatusDeprovision KafkaStatus = "deprovision"

	// KafkaOperationCreate - Kafka cluster create operations
	KafkaOperationCreate KafkaOperation = "create"
	// KafkaOperationDelete = Kafka cluster delete operations
	KafkaOperationDelete KafkaOperation = "delete"
	// KafkaOperationDelete = Kafka cluster deprovision operations
	KafkaOperationDeprovision KafkaOperation = "deprovision"

	// ObservabilityCanaryPodLabelKey that will be used by the observability operator to scrap metrics
	ObservabilityCanaryPodLabelKey = "managed-kafka-canary"

	// ObservabilityCanaryPodLabelValue the value for ObservabilityCanaryPodLabelKey
	ObservabilityCanaryPodLabelValue = "true"

	// KafkaMaxDurationWithProvisioningErrs the maximum duration a Kafka request
	// might be in provisioning state while receiving 5XX errors
	KafkaMaxDurationWithProvisioningErrs = 5 * time.Minute
)

// NamespaceLabels contains labels that indicates if a namespace is a managed application services namespace.
// A namespace with these labels will be scrapped by the Observability operator to retrieve metrics
var NamespaceLabels = map[string]string{
	"mas-managed": "true",
}

func (k KafkaStatus) String() string {
	return string(k)
}

func (k KafkaOperation) String() string {
	return string(k)
}
