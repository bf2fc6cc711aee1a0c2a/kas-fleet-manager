/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage kafka instances and connectors.
 *
 * API version: 0.0.1
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package openapi

import (
	"time"
)

// KafkaRequest struct for KafkaRequest
type KafkaRequest struct {
	Id                  string    `json:"id,omitempty"`
	Kind                string    `json:"kind,omitempty"`
	Href                string    `json:"href,omitempty"`
	Status              string    `json:"status,omitempty"`
	CloudProvider       string    `json:"cloud_provider,omitempty"`
	MultiAz             bool      `json:"multi_az,omitempty"`
	Region              string    `json:"region,omitempty"`
	Owner               string    `json:"owner,omitempty"`
	Name                string    `json:"name,omitempty"`
	BootstrapServerHost string    `json:"bootstrapServerHost,omitempty"`
	CreatedAt           time.Time `json:"created_at,omitempty"`
	UpdatedAt           time.Time `json:"updated_at,omitempty"`
	FailedReason        string    `json:"failed_reason,omitempty"`
	Version             string    `json:"version,omitempty"`
}
