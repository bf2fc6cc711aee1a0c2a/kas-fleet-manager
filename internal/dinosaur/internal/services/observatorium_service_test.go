package services

import (
	"errors"
	"reflect"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/client/observatorium"
)

func Test_ObservatoriumService_GetDinosaurState(t *testing.T) {

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
		want    observatorium.DinosaurState
		wantErr bool
	}{
		{
			name: "successful state retrieval",
			fields: fields{
				observatorium: &ObservatoriumServiceMock{
					GetDinosaurStateFunc: func(name string, namespaceName string) (observatorium.DinosaurState, error) {
						return observatorium.DinosaurState{}, nil
					},
				},
			},
			args: args{
				name:          "test_name",
				namespaceName: "test_namespace",
			},
			wantErr: false,
			want:    observatorium.DinosaurState{},
		},
		{
			name: "fail state retrieval",
			fields: fields{
				observatorium: &ObservatoriumServiceMock{
					GetDinosaurStateFunc: func(name string, namespaceName string) (observatorium.DinosaurState, error) {
						return observatorium.DinosaurState{}, errors.New("fail to get Dinosaur state")
					},
				},
			},
			args: args{
				name:          "test_name",
				namespaceName: "test_namespace",
			},
			wantErr: true,
			want:    observatorium.DinosaurState{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			o := tt.fields.observatorium
			got, err := o.GetDinosaurState(tt.args.name, tt.args.namespaceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetDinosaurState() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDinosaurState() got = %+v, want %+v", got, tt.want)
			}
		})
	}
}
