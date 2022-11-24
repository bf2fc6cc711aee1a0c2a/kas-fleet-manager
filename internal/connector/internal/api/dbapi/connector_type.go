package dbapi

import (
	"encoding/json"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"gorm.io/gorm"
)

type ConnectorType struct {
	db.Model
	Version     string
	Name        string `gorm:"index"`
	Description string
	// A json schema that can be used to validate a connector's connector_spec field.
	JsonSchema api.JSON `gorm:"type:jsonb"`

	// Type's channels
	Channels []ConnectorChannel `gorm:"many2many:connector_type_channels;"`
	// URL to an icon of the connector.
	IconHref string
	// labels used to categorize the connector
	Labels []ConnectorTypeLabel `gorm:"foreignKey:ConnectorTypeID"`
	// annotations metadata
	Annotations []ConnectorTypeAnnotation `gorm:"foreignKey:ConnectorTypeID;references:ID"`
	// connector capabilities used to understand what features a connector support
	Capabilities []ConnectorTypeCapability `gorm:"foreignKey:ConnectorTypeID"`
	Checksum     *string
	FeaturedRank int32 `gorm:"not null;default:0"`
}

type ConnectorTypeList []*ConnectorType

type ConnectorChannel struct {
	Channel   string `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	// needed for soft delete. See https://gorm.io/docs/delete.html#Soft-Delete
	DeletedAt gorm.DeletedAt
}

type ConnectorTypeAnnotation struct {
	ConnectorTypeID string `gorm:"primaryKey;index"`
	Key             string `gorm:"primaryKey;not null"`
	Value           string `gorm:"not null"`
}

type ConnectorTypeLabel struct {
	ConnectorTypeID string `gorm:"primaryKey"`
	Label           string `gorm:"primaryKey"`
}

type ConnectorTypeLabelCount struct {
	Label string
	Count int32
}

type ConnectorTypeLabelCountList []*ConnectorTypeLabelCount

type ConnectorTypeCapability struct {
	ConnectorTypeID string `gorm:"primaryKey"`
	Capability      string `gorm:"primaryKey"`
}

type ConnectorShardMetadata struct {
	ID              int64  `gorm:"primaryKey:autoIncrement"`
	ConnectorTypeId string `gorm:"index:idx_typeid_channel_revision;index:idx_typeid_channel"`
	Channel         string `gorm:"index:idx_typeid_channel_revision;index:idx_typeid_channel"`
	Revision        int64  `gorm:"index:idx_typeid_channel_revision;default:0"`
	LatestRevision  *int64
	ShardMetadata   api.JSON `gorm:"type:jsonb"`
}

type ConnectorCatalogEntry struct {
	ConnectorType *ConnectorType
	Channels      map[string]*ConnectorShardMetadata
}

func (ct *ConnectorType) ChannelNames() []string {
	channels := make([]string, len(ct.Channels))
	for i := 0; i < len(channels); i++ {
		channels[i] = ct.Channels[i].Channel
	}
	return channels
}

func (ct *ConnectorType) SetChannels(channels []string) {
	ct.Channels = make([]ConnectorChannel, len(channels))
	for i, name := range channels {
		ct.Channels[i] = ConnectorChannel{
			Channel: name,
		}
	}
}

func (ct *ConnectorType) CapabilitiesNames() []string {
	capabilities := make([]string, len(ct.Capabilities))
	for i := 0; i < len(capabilities); i++ {
		capabilities[i] = ct.Capabilities[i].Capability
	}
	return capabilities
}

func (ct *ConnectorType) SetCapabilities(capabilities []string) {
	ct.Capabilities = make([]ConnectorTypeCapability, len(capabilities))
	for i, name := range capabilities {
		ct.Capabilities[i] = ConnectorTypeCapability{
			Capability: name,
		}
	}
}

func (ct *ConnectorType) LabelNames() []string {
	labels := make([]string, len(ct.Labels))
	for i, label := range ct.Labels {
		labels[i] = label.Label
	}
	return labels
}

func (ct *ConnectorType) SetLabels(labels []string) {
	id := ct.ID
	ct.Labels = make([]ConnectorTypeLabel, len(labels))
	for i, label := range labels {
		ct.Labels[i] = ConnectorTypeLabel{ConnectorTypeID: id, Label: label}
	}
}

func (ct *ConnectorType) JsonSchemaAsMap() (map[string]interface{}, *errors.ServiceError) {
	schema, err := ct.JsonSchema.Object()
	if err != nil {
		return schema, errors.GeneralError("failed to convert json schema for connector type %q: %v", ct.ID, err)
	}
	return schema, nil
}

func (ct *ConnectorType) SetSchema(schema map[string]interface{}) error {
	var err error
	ct.JsonSchema, err = json.Marshal(schema)
	return err
}
