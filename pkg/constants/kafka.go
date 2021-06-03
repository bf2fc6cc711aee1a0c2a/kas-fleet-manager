package constants

import (
	"time"
)

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
	// KafkaRequestStatusDeleting - external resources are being deleted for the kafka request
	KafkaRequestStatusDeleting KafkaStatus = "deleting"
	// KafkaOperationCreate - Kafka cluster create operations
	KafkaOperationCreate KafkaOperation = "create"
	// KafkaOperationDelete = Kafka cluster delete operations
	KafkaOperationDelete KafkaOperation = "delete"
	// KafkaOperationDeprovision = Kafka cluster deprovision operations
	KafkaOperationDeprovision KafkaOperation = "deprovision"

	// ObservabilityCanaryPodLabelKey that will be used by the observability operator to scrap metrics
	ObservabilityCanaryPodLabelKey = "managed-kafka-canary"

	// ObservabilityCanaryPodLabelValue the value for ObservabilityCanaryPodLabelKey
	ObservabilityCanaryPodLabelValue = "true"

	// KafkaMaxDurationWithProvisioningErrs the maximum duration a Kafka request
	// might be in provisioning state while receiving 5XX errors
	KafkaMaxDurationWithProvisioningErrs = 5 * time.Minute
)

// ordinals - Used to decide if a status comes after or before a given state
var ordinals = map[string]int{
	KafkaRequestStatusAccepted.String():     0,
	KafkaRequestStatusPreparing.String():    10,
	KafkaRequestStatusProvisioning.String(): 20,
	KafkaRequestStatusReady.String():        30,
	KafkaRequestStatusDeprovision.String():  40,
	KafkaRequestStatusDeleting.String():     50,
	KafkaRequestStatusFailed.String():       500,
}

// NamespaceLabels contains labels that indicates if a namespace is a managed application services namespace.
// A namespace with these labels will be scrapped by the Observability operator to retrieve metrics
var NamespaceLabels = map[string]string{
	"mas-managed": "true",
}

func (k KafkaOperation) String() string {
	return string(k)
}

// KafkaStatus Methods
func (k KafkaStatus) String() string {
	return string(k)
}

// CompareTo - Compare this status with the given status returning an int. The result will be 0 if k==k1, -1 if k < k1, and +1 if k > k1
func (k KafkaStatus) CompareTo(k1 KafkaStatus) int {
	ordinalK := ordinals[k.String()]
	ordinalK1 := ordinals[k1.String()]

	switch {
	case ordinalK == ordinalK1:
		return 0
	case ordinalK > ordinalK1:
		return 1
	default:
		return -1
	}
}
