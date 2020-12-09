package constants

// KafkaStatus type
type KafkaStatus string

// KafkaOperation type
type KafkaOperation string

const (
	// KafkaRequestStatusAccepted - kafka request status when accepted by kafka worker
	KafkaRequestStatusAccepted KafkaStatus = "accepted"
	// KafkaRequestStatusProvisioning - kafka request status of a provioned kafka
	KafkaRequestStatusProvisioning KafkaStatus = "provisioning"
	// KafkaRequestStatusComplete - completed kafka request
	KafkaRequestStatusComplete KafkaStatus = "complete"
	// KafkaRequestStatusComplete - kafka request failed
	KafkaRequestStatusFailed KafkaStatus = "failed"

	// KafkaOperationCreate - Kafka cluster create operations
	KafkaOperationCreate KafkaOperation = "create"
	// KafkaOperationDelete = Kafka cluster delete operations
	KafkaOperationDelete KafkaOperation = "delete"

	// ObservabilityCanaryPodLabelKey that will be used by the observability operator to scrap metrics
	ObservabilityCanaryPodLabelKey = "managed-kafka-canary"

	// ObservabilityCanaryPodLabelValue the value for ObservabilityCanaryPodLabelKey
	ObservabilityCanaryPodLabelValue = "true"
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
