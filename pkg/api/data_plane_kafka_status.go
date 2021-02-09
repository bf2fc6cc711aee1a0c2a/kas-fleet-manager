package api

type DataPlaneKafkaStatus struct {
	KafkaClusterId string
	Conditions     []DataPlaneKafkaStatusCondition
	// Going to ignore the rest of fields (like capacity and versions) for now, until when they are needed
}

type DataPlaneKafkaStatusCondition struct {
	Type    string
	Reason  string
	Status  string
	Message string
}
