package services

import (
	"context"
	"errors"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	svcErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
)

func Test_NewObservatoriumService(t *testing.T) {
	type args struct {
		observatorium *observatorium.Client
		kafkaService  KafkaService
	}
	tests := []struct {
		name string
		args args
		want ObservatoriumService
	}{
		{
			name: "Should return the New Observatorium Service",
			args: args{
				observatorium: &observatorium.Client{},
				kafkaService:  &KafkaServiceMock{},
			},
			want: &observatoriumService{
				observatorium: &observatorium.Client{},
				kafkaService:  &KafkaServiceMock{},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			g.Expect(NewObservatoriumService(tt.args.observatorium, tt.args.kafkaService)).To(gomega.Equal(tt.want))
		})
	}
}
func Test_ObservatoriumService_GetKafkaState(t *testing.T) {

	type fields struct {
		observatorium ObservatoriumService
	}
	type args struct {
		name          string
		namespaceName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    observatorium.KafkaState
		wantErr bool
	}{
		{
			name: "successful state retrieval",
			fields: fields{
				observatorium: &ObservatoriumServiceMock{
					GetKafkaStateFunc: func(name string, namespaceName string) (observatorium.KafkaState, error) {
						return observatorium.KafkaState{}, nil
					},
				},
			},
			args: args{
				name:          "test_name",
				namespaceName: "test_namespace",
			},
			wantErr: false,
			want:    observatorium.KafkaState{},
		},
		{
			name: "fail state retrieval",
			fields: fields{
				observatorium: &ObservatoriumServiceMock{
					GetKafkaStateFunc: func(name string, namespaceName string) (observatorium.KafkaState, error) {
						return observatorium.KafkaState{}, errors.New("fail to get Kafka state")
					},
				},
			},
			args: args{
				name:          "test_name",
				namespaceName: "test_namespace",
			},
			wantErr: true,
			want:    observatorium.KafkaState{},
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			o := tt.fields.observatorium
			got, err := o.GetKafkaState(tt.args.name, tt.args.namespaceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetKafkaState() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_observatoriumService_GetMetricsByKafkaId(t *testing.T) {
	type fields struct {
		observatorium *observatorium.Client
		kafkaService  KafkaService
	}
	type args struct {
		ctx           context.Context
		kafkasMetrics *observatorium.KafkaMetrics
		id            string
		query         observatorium.MetricsReqParams
	}

	client, _ := observatorium.NewClientMock(&observatorium.Configuration{})
	q := observatorium.MetricsReqParams{}
	q.ResultType = observatorium.RangeQuery
	q.FillDefaults()

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Should return the kafka request id",
			fields: fields{
				kafkaService: &KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *svcErrors.ServiceError) {
						return &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: "Kafka-request-Id",
							},
							ClusterID: "cluster-id",
							Namespace: "kafkaRequestNamespace",
						}, nil
					},
				},
				observatorium: client,
			},
			args: args{
				ctx: context.Background(),
				kafkasMetrics: &observatorium.KafkaMetrics{
					observatorium.Metric{},
				},
				id:    "cluster-id",
				query: q,
			},
			want: "Kafka-request-Id",
		},
		{
			name: "Should return empty string and err if kafka service Get() fails",
			fields: fields{
				kafkaService: &KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *svcErrors.ServiceError) {
						return nil, &svcErrors.ServiceError{}
					},
				},
			},
			args:    args{},
			want:    "",
			wantErr: true,
		},
		{
			name: "Should return kafkaRequest.ID and error if GetMetrics() fails ",
			fields: fields{
				kafkaService: &KafkaServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.KafkaRequest, *svcErrors.ServiceError) {
						return &dbapi.KafkaRequest{
							Meta: api.Meta{
								ID: "Kafka-request-Id",
							},
							ClusterID: "cluster-id",
							Namespace: "kafkaRequestNamespace",
						}, nil
					},
				},
				observatorium: client,
			},
			args: args{
				ctx:           context.Background(),
				kafkasMetrics: &observatorium.KafkaMetrics{},
				id:            "cluster-id",
				query:         observatorium.MetricsReqParams{},
			},
			want:    "Kafka-request-Id",
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			obs := observatoriumService{
				observatorium: tt.fields.observatorium,
				kafkaService:  tt.fields.kafkaService,
			}
			got, err := obs.GetMetricsByKafkaId(tt.args.ctx, tt.args.kafkasMetrics, tt.args.id, tt.args.query)
			g.Expect(got).To(gomega.Equal(tt.want))
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
