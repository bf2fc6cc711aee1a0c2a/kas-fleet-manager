package converters

import (
	"testing"
	"time"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	. "github.com/onsi/gomega"
)

func Test_ConvertConnectors(t *testing.T) {
	RegisterTestingT(t)
	request := &api.Connector{
		Meta: api.Meta{
			ID:        "a",
			CreatedAt: time.Time{},
			UpdatedAt: time.Time{},
			DeletedAt: nil,
		},
		ConnectorTypeId: "b",
		ConnectorSpec:   api.JSON("{}"),
		Region:          "d",
		ClusterID:       "e",
		CloudProvider:   "f",
		MultiAZ:         false,
		Name:            "g",
		Status:          "h",
		Owner:           "i",
		KafkaID:         "j",
		Version:         7,
		TargetKind:      "k",
		AddonClusterId:  "l",
		OrganisationId:  "m",
	}
	Expect([]map[string]interface{}{
		map[string]interface{}{
			"CreatedAt":         "0001-01-01T00:00:00Z",
			"DeletedAt":         nil,
			"UpdatedAt":         "0001-01-01T00:00:00Z",
			"id":                "a",
			"connector_type_id": "b",
			"connector_spec":    map[string]interface{}{},
			"region":            "d",
			"cluster_id":        "e",
			"cloud_provider":    "f",
			"name":              "g",
			"status":            "h",
			"owner":             "i",
			"kafka_id":          "j",
			"multi_az":          false,
			"version":           float64(7),
			"target_kind":       "k",
			"addon_cluster_id":  "l",
			"organisation_id":   "m",
		},
	}).Should(Equal(ConvertConnectors(request)))
}
