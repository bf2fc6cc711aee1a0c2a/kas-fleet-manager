package observatorium

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_GetKafkaState(t *testing.T) {
	RegisterTestingT(t)
	type fields struct {
		client *Client
	}

	type args struct {
		name              string
		resourceNamespace string
	}

	obsClientMock, err := NewClientMock(&Configuration{})
	Expect(err).ToNot(HaveOccurred())

	tests := []struct {
		name   string
		fields fields
		args   args
		want   KafkaState
	}{
		{
			name: "should return no error and Ready KafkaState",
			fields: fields{
				client: obsClientMock,
			},
			args: args{
				name:              "test",
				resourceNamespace: "test",
			},
			want: KafkaState{
				State: ClusterStateReady,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &ServiceObservatorium{
				client: tt.fields.client,
			}
			state, err := obs.GetKafkaState(tt.args.name, tt.args.resourceNamespace)
			Expect(err).ToNot(HaveOccurred())
			Expect(state).To(Equal(tt.want))
		})
	}
}

func TestServiceObservatorium_GetMetrics(t *testing.T) {
	RegisterTestingT(t)

	type fields struct {
		client *Client
	}

	type args struct {
		metrics   *KafkaMetrics
		namespace string
		rq        *MetricsReqParams
	}

	obsClientMock, err := NewClientMock(&Configuration{})
	Expect(err).ToNot(HaveOccurred(), "failed to create a mock observatorium client")

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Return metrics successfully for Query result type",
			fields: fields{
				client: obsClientMock,
			},
			args: args{
				metrics:   &KafkaMetrics{},
				namespace: "kafka-test",
				rq: &MetricsReqParams{
					ResultType: Query,
				},
			},
			wantErr: false,
		},
		{
			name: "Return metrics successfully for Query result type",
			fields: fields{
				client: obsClientMock,
			},
			args: args{
				metrics:   &KafkaMetrics{},
				namespace: "kafka-test",
				rq: &MetricsReqParams{
					ResultType: RangeQuery,
				},
			},
			wantErr: false,
		},
		{
			name: "Return metrics successfully with specified rqq filters",
			fields: fields{
				client: obsClientMock,
			},
			args: args{
				metrics:   &KafkaMetrics{},
				namespace: "kafka-test",
				rq: &MetricsReqParams{
					Filters:    []string{"kubelet_volume_stats_available_bytes"},
					ResultType: RangeQuery,
				},
			},
			wantErr: false,
		},
		{
			name: "Return an error if result type is not supported for specified filter",
			fields: fields{
				client: obsClientMock,
			},
			args: args{
				metrics:   &KafkaMetrics{},
				namespace: "kafka-test",
				rq: &MetricsReqParams{
					Filters:    []string{"kubelet_volume_stats_available_bytes"},
					ResultType: "unsupported",
				},
			},
			wantErr: true,
		},
		{
			name: "Return an error if result type is not supported",
			fields: fields{
				client: obsClientMock,
			},
			args: args{
				metrics:   &KafkaMetrics{},
				namespace: "kafka-test",
				rq: &MetricsReqParams{
					ResultType: "unsupported",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obs := &ServiceObservatorium{
				client: tt.fields.client,
			}
			Expect(obs.GetMetrics(tt.args.metrics, tt.args.namespace, tt.args.rq) != nil).To(Equal(tt.wantErr))
		})
	}
}
