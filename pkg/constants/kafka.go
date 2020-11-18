package constants

// KafkaStatus type
type KafkaStatus string

const (
	// KafkaRequestStatusAccepted - kafka request status when accepted by kafka worker
	KafkaRequestStatusAccepted KafkaStatus = "accepted"
	// KafkaRequestStatusProvisioning - kafka request status of a provioned kafka
	KafkaRequestStatusProvisioning KafkaStatus = "provisioning"
	// KafkaRequestStatusComplete - completed kafka request
	KafkaRequestStatusComplete KafkaStatus = "complete"
)

func (k KafkaStatus) String() string {
	return string(k)
}
