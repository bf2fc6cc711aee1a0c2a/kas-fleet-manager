package dbapi

import (
	"encoding/json"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"gorm.io/gorm"
)

type ProcessorType struct {
	db.Model
	Name    string `gorm:"index"`
	Version string
	// Type's channels
	Channels    []ProcessorChannel `gorm:"many2many:processor_type_channels;"`
	Description string
	Deprecated  bool `gorm:"not null;default:false"`
	// URL to an icon of the processor.
	IconHref string
	// labels used to categorize the processor
	Labels []ProcessorTypeLabel `gorm:"foreignKey:ProcessorTypeID"`
	// annotations metadata
	Annotations []ProcessorTypeAnnotation `gorm:"foreignKey:ProcessorTypeID;references:ID"`
	// processor capabilities used to understand what features a processor might support
	FeaturedRank int32                     `gorm:"not null;default:0"`
	Capabilities []ProcessorTypeCapability `gorm:"foreignKey:ProcessorTypeID"`
	// A json schema that can be used to validate a processor's definition field.
	JsonSchema api.JSON `gorm:"type:jsonb"`
	Checksum   *string
}

type ProcessorTypeList []*ProcessorType

type ProcessorChannel struct {
	Channel   string `gorm:"primaryKey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	// needed for soft delete. See https://gorm.io/docs/delete.html#Soft-Delete
	DeletedAt gorm.DeletedAt
}

type ProcessorTypeLabel struct {
	ProcessorTypeID string `gorm:"primaryKey"`
	Label           string `gorm:"primaryKey"`
}

type ProcessorTypeAnnotation struct {
	ProcessorTypeID string `gorm:"primaryKey;index"`
	Key             string `gorm:"primaryKey;not null"`
	Value           string `gorm:"not null"`
}

type ProcessorTypeCapability struct {
	ProcessorTypeID string `gorm:"primaryKey"`
	Capability      string `gorm:"primaryKey"`
}

type ProcessorCatalogEntry struct {
	ProcessorType *ProcessorType
	Channels      map[string]*ProcessorShardMetadata
}

func (ct *ProcessorType) ChannelNames() []string {
	channels := make([]string, len(ct.Channels))
	for i := 0; i < len(channels); i++ {
		channels[i] = ct.Channels[i].Channel
	}
	return channels
}

func (ct *ProcessorType) SetChannels(channels []string) {
	ct.Channels = make([]ProcessorChannel, len(channels))
	for i, name := range channels {
		ct.Channels[i] = ProcessorChannel{
			Channel: name,
		}
	}
}

func (ct *ProcessorType) CapabilitiesNames() []string {
	capabilities := make([]string, len(ct.Capabilities))
	for i := 0; i < len(capabilities); i++ {
		capabilities[i] = ct.Capabilities[i].Capability
	}
	return capabilities
}

func (ct *ProcessorType) SetCapabilities(capabilities []string) {
	ct.Capabilities = make([]ProcessorTypeCapability, len(capabilities))
	for i, name := range capabilities {
		ct.Capabilities[i] = ProcessorTypeCapability{
			Capability: name,
		}
	}
}

func (ct *ProcessorType) LabelNames() []string {
	labels := make([]string, len(ct.Labels))
	for i, label := range ct.Labels {
		labels[i] = label.Label
	}
	return labels
}

func (ct *ProcessorType) SetLabels(labels []string) {
	id := ct.ID
	ct.Labels = make([]ProcessorTypeLabel, len(labels))
	for i, label := range labels {
		ct.Labels[i] = ProcessorTypeLabel{ProcessorTypeID: id, Label: label}
	}
}

func (ct *ProcessorType) JsonSchemaAsMap() (map[string]interface{}, *errors.ServiceError) {
	schema, err := ct.JsonSchema.Object()
	if err != nil {
		return schema, errors.GeneralError("failed to convert json schema for processor type %q: %v", ct.ID, err)
	}
	return schema, nil
}

func (ct *ProcessorType) SetSchema(schema map[string]interface{}) error {
	var err error
	ct.JsonSchema, err = json.Marshal(schema)
	return err
}
