package services

import (
	"errors"
	v1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/ocm"
	"reflect"
	"testing"
)

func TestSyncsetService_Create(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}

	kafkaSyncBuilder, _, _ := newKafkaSyncsetBuilder(&api.KafkaRequest{
		Name:      testKafkaRequestName,
		ClusterID: testClusterID,
	})

	type args struct {
		syncsetBuilder *v1.SyncsetBuilder
	}

	wantedSynceset, _ := kafkaSyncBuilder.ID(testSyncsetID).Build()

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *v1.Syncset
	}{
		{
			name: "successful syncset creation",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateSyncSetFunc: func(clusterId string, syncset *v1.Syncset) (*v1.Syncset, error) {
						return syncset, nil
					},
				},
			},
			args: args{
				syncsetBuilder: kafkaSyncBuilder,
			},
			want: wantedSynceset,
		},
		{
			name: "failed syncset creation",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					CreateSyncSetFunc: func(clusterId string, syncset *v1.Syncset) (*v1.Syncset, error) {
						return syncset, errors.New("MGD-SERV-API-9: failed to create syncset: test-syncset-id for cluster id: test-cluster-id")
					},
				},
			},
			args: args{
				syncsetBuilder: kafkaSyncBuilder,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &syncsetService{
				ocmClient: tt.fields.ocmClient,
			}
			got, err := k.Create(tt.args.syncsetBuilder, testSyncsetID, testClusterID)

			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}

		})
	}
}

func TestSyncsetService_Delete(t *testing.T) {
	type fields struct {
		ocmClient ocm.Client
	}

	kafkaSyncBuilder, _, _ := newKafkaSyncsetBuilder(&api.KafkaRequest{
		Name:      testKafkaRequestName,
		ClusterID: testClusterID,
	})

	type args struct {
		syncsetBuilder *v1.SyncsetBuilder
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "successful syncset deletion",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					DeleteSyncSetFunc: func(clusterID string, syncsetID string) (int, error) {
						return 204, nil
					},
				},
			},
			args: args{
				syncsetBuilder: kafkaSyncBuilder,
			},
		},
		{
			name: "failed to syncset deletion",
			fields: fields{
				ocmClient: &ocm.ClientMock{
					DeleteSyncSetFunc: func(clusterID string, syncsetId string) (int, error) {
						return 0, errors.New("MGD-SERV-API-9: failed to delete syncset: test-syncset-id for cluster id")
					},
				},
			},
			args: args{
				syncsetBuilder: kafkaSyncBuilder,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &syncsetService{
				ocmClient: tt.fields.ocmClient,
			}
			err := k.Delete(testSyncsetID, testClusterID)

			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
