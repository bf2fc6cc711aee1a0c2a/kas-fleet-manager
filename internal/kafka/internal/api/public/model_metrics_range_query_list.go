/*
 * Kafka Management API
 *
 * Kafka Management API is a REST API to manage Kafka instances
 *
 * API version: 1.13.0
 * Contact: rhosak-support@redhat.com
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

// MetricsRangeQueryList struct for MetricsRangeQueryList
type MetricsRangeQueryList struct {
	Kind  string       `json:"kind,omitempty"`
	Id    string       `json:"id,omitempty"`
	Items []RangeQuery `json:"items,omitempty"`
}
