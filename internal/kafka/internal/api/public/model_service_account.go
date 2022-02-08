/*
 * Kafka Service Fleet Manager
 *
 * Kafka Service Fleet Manager is a Rest API to manage Kafka instances.
 *
 * API version: 1.3.0
 * Generated by: OpenAPI Generator (https://openapi-generator.tech)
 */

package public

import (
	"time"
)

// ServiceAccount Service Account created in MAS-SSO for the Kafka Cluster for authentication
type ServiceAccount struct {
	// server generated unique id of the service account
	Id           string `json:"id,omitempty"`
	Kind         string `json:"kind,omitempty"`
	Href         string `json:"href,omitempty"`
	Name         string `json:"name,omitempty"`
	Description  string `json:"description,omitempty"`
	ClientId     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	// Deprecated
	DeprecatedOwner string    `json:"owner,omitempty"`
	CreatedBy       string    `json:"created_by,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
}
