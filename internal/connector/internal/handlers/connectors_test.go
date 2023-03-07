package handlers

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/onsi/gomega"
	"testing"
)

func TestValidateConnectorImmutableProperties(t *testing.T) {
	type fields struct {
		patch            public.ConnectorRequest
		originalResource public.Connector
		invalid          bool
	}

	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Check Name :: Modified - Mutable field",
			fields: fields{
				patch: public.ConnectorRequest{
					Name: "name",
				},
				originalResource: public.Connector{},
				invalid:          false,
			},
		},
		{
			name: "Check ConnectorTypeId :: Modified",
			fields: fields{
				patch: public.ConnectorRequest{
					ConnectorTypeId: "connector-type-id",
				},
				originalResource: public.Connector{},
				invalid:          true,
			},
		},
		{
			name: "Check NamespaceId :: Modified",
			fields: fields{
				patch: public.ConnectorRequest{
					NamespaceId: "namespace-id",
				},
				originalResource: public.Connector{},
				invalid:          true,
			},
		},
		{
			name: "Check Channel :: Modified",
			fields: fields{
				patch: public.ConnectorRequest{
					Channel: "beta",
				},
				originalResource: public.Connector{},
				invalid:          true,
			},
		},
		{
			name: "Check DesiredState :: Modified - Mutable field",
			fields: fields{
				patch: public.ConnectorRequest{
					DesiredState: "ready",
				},
				originalResource: public.Connector{},
				invalid:          false,
			},
		},
		{
			name: "Check Annotations :: Modified - Mutable field",
			fields: fields{
				patch: public.ConnectorRequest{
					Annotations: map[string]string{"annotation1": "value1"},
				},
				originalResource: public.Connector{},
				invalid:          false,
			},
		},
		{
			name: "Check Kafka :: Modified - Mutable field",
			fields: fields{
				patch: public.ConnectorRequest{
					Kafka: public.KafkaConnectionSettings{
						Id:  "id",
						Url: "url",
					},
				},
				originalResource: public.Connector{},
				invalid:          false,
			},
		},
		{
			name: "Check ServiceAccount :: Modified - Mutable field",
			fields: fields{
				patch: public.ConnectorRequest{
					ServiceAccount: public.ServiceAccount{
						ClientId:     "client-id",
						ClientSecret: "client-secret",
					},
				},
				originalResource: public.Connector{},
				invalid:          false,
			},
		},
		{
			name: "Check SchemaRegistry :: Modified - Mutable field",
			fields: fields{
				patch: public.ConnectorRequest{
					SchemaRegistry: public.SchemaRegistryConnectionSettings{
						Id:  "id",
						Url: "url",
					},
				},
				originalResource: public.Connector{},
				invalid:          false,
			},
		},
		{
			name: "Check Connector :: Modified - Mutable field",
			fields: fields{
				patch: public.ConnectorRequest{
					Connector: map[string]interface{}{"field1": "value1"},
				},
				originalResource: public.Connector{},
				invalid:          false,
			},
		},
		{
			name: "Check Name :: Unmodified",
			fields: fields{
				patch: public.ConnectorRequest{
					Name: "name",
				},
				originalResource: public.Connector{
					Name: "name",
				},
				invalid: false,
			},
		},
		{
			name: "Check ConnectorTypeId :: Unmodified",
			fields: fields{
				patch: public.ConnectorRequest{
					ConnectorTypeId: "connector-type-id",
				},
				originalResource: public.Connector{
					ConnectorTypeId: "connector-type-id",
				},
				invalid: false,
			},
		},
		{
			name: "Check NamespaceId :: Unmodified",
			fields: fields{
				patch: public.ConnectorRequest{
					NamespaceId: "namespace-id",
				},
				originalResource: public.Connector{
					NamespaceId: "namespace-id",
				},
				invalid: false,
			},
		},
		{
			name: "Check Channel :: Unmodified",
			fields: fields{
				patch: public.ConnectorRequest{
					Channel: "beta",
				},
				originalResource: public.Connector{
					Channel: "beta",
				},
				invalid: false,
			},
		},
		{
			name: "Check DesiredState :: Unmodified",
			fields: fields{
				patch: public.ConnectorRequest{
					DesiredState: "ready",
				},
				originalResource: public.Connector{
					DesiredState: "ready",
				},
				invalid: false,
			},
		},
		{
			name: "Check Annotations :: Unmodified",
			fields: fields{
				patch: public.ConnectorRequest{
					Annotations: map[string]string{"annotation1": "value1"},
				},
				originalResource: public.Connector{
					Annotations: map[string]string{"annotation1": "value1"},
				},
				invalid: false,
			},
		},
		{
			name: "Check Kafka :: Unmodified",
			fields: fields{
				patch: public.ConnectorRequest{
					Kafka: public.KafkaConnectionSettings{
						Id:  "id",
						Url: "url",
					},
				},
				originalResource: public.Connector{
					Kafka: public.KafkaConnectionSettings{
						Id:  "id",
						Url: "url",
					},
				},
				invalid: false,
			},
		},
		{
			name: "Check ServiceAccount :: Unmodified",
			fields: fields{
				patch: public.ConnectorRequest{
					ServiceAccount: public.ServiceAccount{
						ClientId:     "client-id",
						ClientSecret: "client-secret",
					},
				},
				originalResource: public.Connector{
					ServiceAccount: public.ServiceAccount{
						ClientId:     "client-id",
						ClientSecret: "client-secret",
					},
				},
				invalid: false,
			},
		},
		{
			name: "Check SchemaRegistry :: Unmodified",
			fields: fields{
				patch: public.ConnectorRequest{
					SchemaRegistry: public.SchemaRegistryConnectionSettings{
						Id:  "id",
						Url: "url",
					},
				},
				originalResource: public.Connector{
					SchemaRegistry: public.SchemaRegistryConnectionSettings{
						Id:  "id",
						Url: "url",
					},
				},
				invalid: false,
			},
		},
		{
			name: "Check Connector :: Unmodified",
			fields: fields{
				patch: public.ConnectorRequest{
					Connector: map[string]interface{}{"field1": "value1"},
				},
				originalResource: public.Connector{
					Connector: map[string]interface{}{"field1": "value1"},
				},
				invalid: false,
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			err := validateConnectorImmutableProperties(tt.fields.patch, tt.fields.originalResource)()
			g.Expect(err != nil).To(gomega.Equal(tt.fields.invalid))
		})
	}

}
