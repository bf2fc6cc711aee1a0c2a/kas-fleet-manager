/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage Kafka instances.
 *
 * API version: 1.1.2
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// RangeQuery struct for RangeQuery
type RangeQuery struct {
	Metric map[string]string `json:"metric,omitempty"`
	Values []Values          `json:"values,omitempty"`
}
