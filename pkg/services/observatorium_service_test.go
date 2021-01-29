package services

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
)

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
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			o := tt.fields.observatorium
			got, err := o.GetKafkaState(tt.args.name, tt.args.namespaceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetKafkaState() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetKafkaState() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}
