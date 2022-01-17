package dbapi

import (
	"strings"
)

type DataPlaneDinosaurStatus struct {
	DinosaurClusterId string
	Conditions     []DataPlaneDinosaurStatusCondition
	// Going to ignore the rest of fields (like capacity and versions) for now, until when they are needed
	Routes          []DataPlaneDinosaurRouteRequest
	DinosaurVersion    string
	StrimziVersion  string
	DinosaurIBPVersion string
}

type DataPlaneDinosaurStatusCondition struct {
	Type    string
	Reason  string
	Status  string
	Message string
}

type DataPlaneDinosaurRoute struct {
	Domain string
	Router string
}

type DataPlaneDinosaurRouteRequest struct {
	Name   string
	Prefix string
	Router string
}

func (d *DataPlaneDinosaurStatus) GetReadyCondition() (DataPlaneDinosaurStatusCondition, bool) {
	for _, c := range d.Conditions {
		if strings.EqualFold(c.Type, "Ready") {
			return c, true
		}
	}
	return DataPlaneDinosaurStatusCondition{}, false
}