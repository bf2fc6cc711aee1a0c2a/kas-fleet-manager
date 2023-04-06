package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/onsi/gomega"
	"gorm.io/gorm"
	"testing"
	"time"
)

func Test_ConvertProcessor(t *testing.T) {
	g := gomega.NewWithT(t)

	createdAt := time.Now()
	updatedAt := createdAt.Add(time.Minute)
	processor := public.Processor{
		Id:              "id",
		Kind:            KindProcessor,
		Href:            "href",
		Owner:           "owner",
		CreatedAt:       createdAt,
		ModifiedAt:      updatedAt,
		Name:            "name",
		NamespaceId:     "namespace_id",
		ProcessorTypeId: "processor_type_id",
		Channel:         public.CHANNEL_STABLE,
		DesiredState:    public.PROCESSORDESIREDSTATE_READY,
		Annotations:     map[string]string{"key": "value"},
		ResourceVersion: 1,
		Kafka: public.KafkaConnectionSettings{
			Id:  "kafka_id",
			Url: "kafka_url",
		},
		ServiceAccount: public.ServiceAccount{
			ClientId:     "client_id",
			ClientSecret: "client_secret",
		},
		Definition: map[string]interface{}{"from": "somewhere"},
		ErrorHandler: public.ErrorHandler{
			Log:  nil,
			Stop: nil,
			DeadLetterQueue: public.ErrorHandlerDeadLetterQueueDeadLetterQueue{
				Topic: "topic",
			},
		},
		Status: public.ProcessorStatusStatus{
			State: public.PROCESSORSTATE_READY,
			Error: "error",
		},
	}

	p, _ := ConvertProcessor(&processor)

	g.Expect(p).ToNot(gomega.BeNil())
	g.Expect(p.ID).To(gomega.Equal("id"))
	g.Expect(p.Name).To(gomega.Equal("name"))
	g.Expect(p.NamespaceId).To(gomega.Equal("namespace_id"))
	g.Expect(p.ProcessorTypeId).To(gomega.Equal("processor_type_id"))
	g.Expect(p.Owner).To(gomega.Equal("owner"))
	g.Expect(p.Version).To(gomega.Equal(int64(1)))
	g.Expect(p.DesiredState).To(gomega.Equal(dbapi.ProcessorReady))
	g.Expect(p.Channel).To(gomega.Equal("stable"))
	g.Expect(string(p.Definition)).To(gomega.MatchJSON(`{"from":"somewhere"}`))
	g.Expect(string(p.ErrorHandler)).To(gomega.MatchJSON(`{"log":null,"stop":null,"dead_letter_queue":{"topic":"topic"}}`))
	g.Expect(p.Kafka.KafkaID).To(gomega.Equal("kafka_id"))
	g.Expect(p.Kafka.BootstrapServer).To(gomega.Equal("kafka_url"))
	g.Expect(p.ServiceAccount.ClientId).To(gomega.Equal("client_id"))
	g.Expect(p.ServiceAccount.ClientSecret).To(gomega.Equal("client_secret"))
	g.Expect(len(p.Annotations)).To(gomega.Equal(1))
	g.Expect(p.Annotations[0].Key).To(gomega.Equal("key"))
	g.Expect(p.Annotations[0].Value).To(gomega.Equal("value"))
	g.Expect(p.Annotations[0].ProcessorID).To(gomega.Equal("id"))
	g.Expect(p.Status.Phase).To(gomega.Equal(dbapi.ProcessorStatusPhaseReady))
}

func Test_ConvertProcessorAnnotations(t *testing.T) {
	g := gomega.NewWithT(t)

	annotations := map[string]string{"key": "value"}

	a := ConvertProcessorAnnotations("id", annotations)

	g.Expect(len(a)).To(gomega.Equal(1))
	g.Expect(a[0].Key).To(gomega.Equal("key"))
	g.Expect(a[0].Value).To(gomega.Equal("value"))
	g.Expect(a[0].ProcessorID).To(gomega.Equal("id"))
}

func Test_PresentProcessor(t *testing.T) {
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

	p, _ := PresentProcessor(&processor)

	g.Expect(p).ToNot(gomega.BeNil())
	g.Expect(p.Id).To(gomega.Equal("id"))
	g.Expect(p.Kind).To(gomega.Equal(KindProcessor))
	g.Expect(p.Href).To(gomega.Equal("/api/connector_mgmt/v2alpha1/processors/id"))
	g.Expect(p.Name).To(gomega.Equal("name"))
	g.Expect(p.NamespaceId).To(gomega.Equal("namespace_id"))
	g.Expect(p.ProcessorTypeId).To(gomega.Equal("processor_type_id"))
	g.Expect(p.Owner).To(gomega.Equal("owner"))
	g.Expect(p.ResourceVersion).To(gomega.Equal(int64(1)))
	g.Expect(p.DesiredState).To(gomega.Equal(public.PROCESSORDESIREDSTATE_READY))
	g.Expect(p.Channel).To(gomega.Equal(public.CHANNEL_STABLE))
	g.Expect(p.Definition).To(gomega.Equal(map[string]interface{}{"from": "somewhere"}))
	g.Expect(p.ErrorHandler.Log).To(gomega.BeNil())
	g.Expect(p.ErrorHandler.Stop).To(gomega.BeNil())
	g.Expect(p.ErrorHandler.DeadLetterQueue).To(gomega.Equal(public.ErrorHandlerDeadLetterQueueDeadLetterQueue{Topic: "topic"}))
	g.Expect(p.Kafka.Id).To(gomega.Equal("kafka_id"))
	g.Expect(p.Kafka.Url).To(gomega.Equal("kafka_url"))
	g.Expect(p.ServiceAccount.ClientId).To(gomega.Equal("client_id"))
	g.Expect(p.ServiceAccount.ClientSecret).To(gomega.Equal("client_secret"))
	g.Expect(len(p.Annotations)).To(gomega.Equal(1))
	g.Expect(p.Annotations["key"]).To(gomega.Equal("value"))
	g.Expect(p.Status.State).To(gomega.Equal(public.PROCESSORSTATE_READY))
}

func Test_PresentProcessorWithError(t *testing.T) {
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
	processorWithConditions := dbapi.ProcessorWithConditions{
		Processor:  processor,
		Conditions: []byte(`[{"type": "Ready", "status": "False", "message": "error.message Brokenfailure.count", "reason": "Dead"}]`),
	}

	p, _ := PresentProcessorWithError(&processorWithConditions)

	g.Expect(p).ToNot(gomega.BeNil())
	g.Expect(p.Id).To(gomega.Equal("id"))
	g.Expect(p.Kind).To(gomega.Equal(KindProcessor))
	g.Expect(p.Href).To(gomega.Equal("/api/connector_mgmt/v2alpha1/processors/id"))
	g.Expect(p.Name).To(gomega.Equal("name"))
	g.Expect(p.NamespaceId).To(gomega.Equal("namespace_id"))
	g.Expect(p.ProcessorTypeId).To(gomega.Equal("processor_type_id"))
	g.Expect(p.Owner).To(gomega.Equal("owner"))
	g.Expect(p.ResourceVersion).To(gomega.Equal(int64(1)))
	g.Expect(p.DesiredState).To(gomega.Equal(public.PROCESSORDESIREDSTATE_READY))
	g.Expect(p.Channel).To(gomega.Equal(public.CHANNEL_STABLE))
	g.Expect(p.Definition).To(gomega.Equal(map[string]interface{}{"from": "somewhere"}))
	g.Expect(p.ErrorHandler.Log).To(gomega.BeNil())
	g.Expect(p.ErrorHandler.Stop).To(gomega.BeNil())
	g.Expect(p.ErrorHandler.DeadLetterQueue).To(gomega.Equal(public.ErrorHandlerDeadLetterQueueDeadLetterQueue{Topic: "topic"}))
	g.Expect(p.Kafka.Id).To(gomega.Equal("kafka_id"))
	g.Expect(p.Kafka.Url).To(gomega.Equal("kafka_url"))
	g.Expect(p.ServiceAccount.ClientId).To(gomega.Equal("client_id"))
	g.Expect(p.ServiceAccount.ClientSecret).To(gomega.Equal("client_secret"))
	g.Expect(len(p.Annotations)).To(gomega.Equal(1))
	g.Expect(p.Annotations["key"]).To(gomega.Equal("value"))
	g.Expect(p.Status.State).To(gomega.Equal(public.PROCESSORSTATE_READY))
	g.Expect(p.Status.Error).To(gomega.Equal("Dead: Broken"))
}

func Test_PresentProcessorAnnotations(t *testing.T) {
	g := gomega.NewWithT(t)

	annotations := []dbapi.ProcessorAnnotation{{
		ProcessorID: "id",
		Key:         "key",
		Value:       "value",
	}}

	a := PresentProcessorAnnotations(annotations)

	g.Expect(len(a)).To(gomega.Equal(1))
	g.Expect(a["key"]).To(gomega.Equal("value"))
}
