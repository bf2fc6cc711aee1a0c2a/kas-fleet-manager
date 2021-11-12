package dbapi

import (
	"strings"
)

type DataPlaneKafkaStatus struct {
	KafkaClusterId string
	Conditions     []DataPlaneKafkaStatusCondition
	// Going to ignore the rest of fields (like capacity and versions) for now, until when they are needed
	Routes          []DataPlaneKafkaRouteRequest
	KafkaVersion    string
	StrimziVersion  string
	KafkaIBPVersion string
}

type DataPlaneKafkaStatusCondition struct {
	Type    string
	Reason  string
	Status  string
	Message string
}

type DataPlaneKafkaRoute struct {
	Domain string
	Router string
}

type DataPlaneKafkaRouteRequest struct {
	Name   string
	Prefix string
	Router string
}

func (d *DataPlaneKafkaStatus) GetReadyCondition() (DataPlaneKafkaStatusCondition, bool) {
	for _, c := range d.Conditions {
		if strings.EqualFold(c.Type, "Ready") {
			return c, true
		}
	}
	return DataPlaneKafkaStatusCondition{}, false
}
