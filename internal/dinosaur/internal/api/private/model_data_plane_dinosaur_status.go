/*
 * Dinosaur Service Fleet Manager
 *
 * Dinosaur Service Fleet Manager APIs that are used by internal services e.g fleetshard operators.
 *
 * API version: 1.4.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package private

// DataPlaneDinosaurStatus Schema of the status object for a Dinosaur cluster
type DataPlaneDinosaurStatus struct {
	// The status conditions of a Dinosaur cluster
	Conditions []DataPlaneClusterUpdateStatusRequestConditions `json:"conditions,omitempty"`
	Capacity   DataPlaneDinosaurStatusCapacity                 `json:"capacity,omitempty"`
	Versions   DataPlaneDinosaurStatusVersions                 `json:"versions,omitempty"`
	// Routes created for a Dinosaur cluster
	Routes *[]DataPlaneDinosaurStatusRoutes `json:"routes,omitempty"`
}
