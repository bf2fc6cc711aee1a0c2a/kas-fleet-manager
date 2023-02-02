package presenters

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/onsi/gomega"
	"testing"
)

func Test_ConvertProcessorRequest(t *testing.T) {
	g := gomega.NewWithT(t)

	request := public.ProcessorRequest{
		Name:            "name",
		NamespaceId:     "namespace_id",
		ProcessorTypeId: "processor_type_id",
		Channel:         public.CHANNEL_STABLE,
		DesiredState:    public.PROCESSORDESIREDSTATE_READY,
		Annotations:     map[string]string{"key": "value"},
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
	}

	r, _ := ConvertProcessorRequest("id", &request)

	g.Expect(r).ToNot(gomega.BeNil())
	g.Expect(r.ID).To(gomega.Equal("id"))
	g.Expect(r.Name).To(gomega.Equal("name"))
	g.Expect(r.NamespaceId).To(gomega.Equal("namespace_id"))
	g.Expect(r.ProcessorTypeId).To(gomega.Equal("processor_type_id"))
	g.Expect(r.Owner).To(gomega.Equal(""))
	g.Expect(r.OrganisationId).To(gomega.Equal(""))
	g.Expect(r.Version).To(gomega.Equal(int64(0)))
	g.Expect(r.DesiredState).To(gomega.Equal(dbapi.ProcessorReady))
	g.Expect(r.Channel).To(gomega.Equal("stable"))
	g.Expect(string(r.Definition)).To(gomega.MatchJSON(`{"from":"somewhere"}`))
	g.Expect(string(r.ErrorHandler)).To(gomega.MatchJSON(`{"log":null,"stop":null,"dead_letter_queue":{"topic":"topic"}}`))
	g.Expect(r.Kafka.KafkaID).To(gomega.Equal("kafka_id"))
	g.Expect(r.Kafka.BootstrapServer).To(gomega.Equal("kafka_url"))
	g.Expect(r.ServiceAccount.ClientId).To(gomega.Equal("client_id"))
	g.Expect(r.ServiceAccount.ClientSecret).To(gomega.Equal("client_secret"))
	g.Expect(len(r.Annotations)).To(gomega.Equal(1))
	g.Expect(r.Annotations[0].Key).To(gomega.Equal("key"))
	g.Expect(r.Annotations[0].Value).To(gomega.Equal("value"))
	g.Expect(r.Annotations[0].ProcessorID).To(gomega.Equal("id"))
	g.Expect(r.Status.Phase).To(gomega.Equal(dbapi.ProcessorStatusPhase("")))
}
