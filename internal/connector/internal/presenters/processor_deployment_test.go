package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/onsi/gomega"
	"gorm.io/gorm"
	"testing"
	"time"
)

func Test_PresentProcessorDeployment(t *testing.T) {
	g := gomega.NewWithT(t)

	createdAt := time.Now()
	updatedAt := createdAt.Add(time.Minute)
	deletedAt := updatedAt.Add(time.Minute)
	processor := dbapi.Processor{
		Model: db.Model{
			ID:        "id",
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			DeletedAt: gorm.DeletedAt{
				Time:  deletedAt,
				Valid: false,
			},
		},
		Name:            "name",
		NamespaceId:     "namespace_id",
		ProcessorTypeId: "processor_type_id",
		Channel:         "stable",
		DesiredState:    "ready",
		Owner:           "owner",
		OrganisationId:  "organisation_id",
		Version:         1,
		Annotations: []dbapi.ProcessorAnnotation{{
			ProcessorID: "id",
			Key:         "key",
			Value:       "value",
		}},
		Kafka: dbapi.KafkaConnectionSettings{
			KafkaID:         "kafka_id",
			BootstrapServer: "kafka_url",
		},
		ServiceAccount: dbapi.ServiceAccount{
			ClientId:     "client_id",
			ClientSecret: "client_secret",
		},
		Definition:   []byte(`{"from":"somewhere"}`),
		ErrorHandler: []byte(`{"log":null,"stop":null,"dead_letter_queue":{"topic":"topic"}}`),
		Status: dbapi.ProcessorStatus{
			Model:       db.Model{},
			NamespaceID: "namespace_id",
			Phase:       dbapi.ProcessorStatusPhaseReady,
		},
	}
	processorDeployment := dbapi.ProcessorDeployment{
		Model: db.Model{
			ID:        "deployment_id",
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
			DeletedAt: gorm.DeletedAt{
				Time:  deletedAt,
				Valid: false,
			},
		},
		Version:                  0,
		ProcessorID:              "processor_id",
		Processor:                processor,
		ProcessorVersion:         1,
		ProcessorShardMetadataID: 2,
		ProcessorShardMetadata: dbapi.ProcessorShardMetadata{
			ID:             3,
			Revision:       4,
			LatestRevision: nil,
			ShardMetadata:  []byte(`{"shard":"metadata"}`),
		},
		ClusterID:    "cluster_id",
		NamespaceID:  "namespace_id",
		OperatorID:   "operator_id",
		AllowUpgrade: false,
		Status: dbapi.ProcessorDeploymentStatus{
			Model: db.Model{
				ID:        "1",
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
				DeletedAt: gorm.DeletedAt{
					Time:  deletedAt,
					Valid: false,
				},
			},
			Phase:            dbapi.ProcessorStatusPhasePrepared,
			Version:          2,
			Conditions:       []byte(`[{"type": "Ready", "status": "True"}]`),
			Operators:        []byte(`{"assigned": { "id": "operator_id", "type": "operator_type" }, "available": {} }`),
			UpgradeAvailable: false,
		},
	}

	pd, _ := PresentProcessorDeployment(&processorDeployment, true)

	g.Expect(pd).ToNot(gomega.BeNil())
	g.Expect(pd.Id).To(gomega.Equal("deployment_id"))
	g.Expect(pd.Kind).To(gomega.Equal(KindProcessorDeployment))
	g.Expect(pd.Href).To(gomega.Equal("/api/connector_mgmt/v2alpha1/agent/kafka_connector_clusters/cluster_id/processors/deployments/deployment_id"))
	g.Expect(pd.Spec).ToNot(gomega.BeNil())
	g.Expect(pd.Spec.ProcessorId).To(gomega.Equal("id"))
	g.Expect(pd.Spec.NamespaceId).To(gomega.Equal("namespace_id"))
	g.Expect(pd.Spec.ProcessorTypeId).To(gomega.Equal("processor_type_id"))
	g.Expect(pd.Spec.DesiredState).To(gomega.Equal(private.PROCESSORDESIREDSTATE_READY))
	g.Expect(pd.Spec.ShardMetadata).To(gomega.Equal(map[string]interface{}{"shard": "metadata"}))
	g.Expect(pd.Spec.ProcessorResourceVersion).To(gomega.Equal(int64(1)))
	g.Expect(pd.Spec.Definition).To(gomega.Equal(map[string]interface{}{"from": "somewhere"}))
	g.Expect(pd.Spec.ErrorHandler.Log).To(gomega.BeNil())
	g.Expect(pd.Spec.ErrorHandler.Stop).To(gomega.BeNil())
	g.Expect(pd.Spec.ErrorHandler.DeadLetterQueue).To(gomega.Equal(private.ErrorHandlerDeadLetterQueueDeadLetterQueue{Topic: "topic"}))
	g.Expect(pd.Spec.Kafka.Id).To(gomega.Equal("kafka_id"))
	g.Expect(pd.Spec.Kafka.Url).To(gomega.Equal("kafka_url"))
	g.Expect(pd.Spec.ServiceAccount.ClientId).To(gomega.Equal("client_id"))
	g.Expect(pd.Spec.ServiceAccount.ClientSecret).To(gomega.Equal("client_secret"))
	g.Expect(pd.Spec.OperatorId).To(gomega.Equal("operator_id"))
	g.Expect(pd.Metadata).ToNot(gomega.BeNil())
	g.Expect(pd.Metadata.CreatedAt).To(gomega.Equal(createdAt))
	g.Expect(pd.Metadata.UpdatedAt).To(gomega.Equal(updatedAt))
	g.Expect(pd.Metadata.ResourceVersion).To(gomega.Equal(int64(0)))
	g.Expect(pd.Metadata.ResolvedSecrets).To(gomega.Equal(true))
	g.Expect(pd.Metadata.Annotations).ToNot(gomega.BeNil())
	g.Expect(len(pd.Metadata.Annotations)).To(gomega.Equal(1))
	g.Expect(pd.Metadata.Annotations["key"]).To(gomega.Equal("value"))
	g.Expect(pd.Status).ToNot(gomega.BeNil())
	g.Expect(pd.Status.Operators).To(gomega.Equal(private.ConnectorDeploymentStatusOperators{
		Assigned: private.ConnectorOperator{
			Id:   "operator_id",
			Type: "operator_type",
		},
		Available: private.ConnectorOperator{},
	}))
	g.Expect(pd.Status.Phase).ToNot(gomega.BeNil())
	g.Expect(pd.Status.Conditions).To(gomega.Equal([]private.MetaV1Condition{{
		Type:   "Ready",
		Status: "True",
	}}))
	g.Expect(pd.Status.ResourceVersion).ToNot(gomega.BeNil())
}
