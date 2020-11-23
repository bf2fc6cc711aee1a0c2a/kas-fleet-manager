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
	// KafkaOperationCreate - Kafka cluster create operations
	KafkaOperationCreate KafkaOperation = "create"
	// KafkaOperationDelete = Kafka cluster delete operations
	KafkaOperationDelete KafkaOperation = "delete"
)

func (k KafkaStatus) String() string {
	return string(k)
}

func (k KafkaOperation) String() string {
	return string(k)
}
