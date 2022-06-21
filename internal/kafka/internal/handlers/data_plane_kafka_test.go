package handlers

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	v1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"

	. "github.com/onsi/gomega"
)

var (
	emptyJsonBody = "{}"
	testId        = "test-id"
)

func Test_UpdateUpdateKafkaStatuses(t *testing.T) {
	type fields struct {
		dataplaneKafkaService services.DataPlaneKafkaService
		kafkaService          services.KafkaService
	}

	type args struct {
		body []byte
	}

	tests := []struct {
		name           string
		args           args
		fields         fields
		wantStatusCode int
	}{
		{
			name: "should fail if UpdateDataPlaneKafkaService fails",
			args: args{
				body: []byte(emptyJsonBody),
			},
			fields: fields{
				dataplaneKafkaService: &services.DataPlaneKafkaServiceMock{
					UpdateDataPlaneKafkaServiceFunc: func(ctx context.Context, clusterId string, status []*dbapi.DataPlaneKafkaStatus) *errors.ServiceError {
						return errors.GeneralError("update failed")
					},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should successfully update Kafka Statuses",
			args: args{
				body: []byte(emptyJsonBody),
			},
			fields: fields{
				dataplaneKafkaService: &services.DataPlaneKafkaServiceMock{
					UpdateDataPlaneKafkaServiceFunc: func(ctx context.Context, clusterId string, status []*dbapi.DataPlaneKafkaStatus) *errors.ServiceError {
						return nil
					},
				},
			},
			wantStatusCode: http.StatusOK,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDataPlaneKafkaHandler(tt.fields.dataplaneKafkaService, tt.fields.kafkaService)

			req, rw := GetHandlerParams("GET", "/{id}", bytes.NewBuffer(tt.args.body))
			req = mux.SetURLVars(req, map[string]string{"id": testId})

			h.UpdateKafkaStatuses(rw, req)
			resp := rw.Result()
			Expect(resp.StatusCode).To(Equal(tt.wantStatusCode))
			resp.Body.Close()
		})
	}
}

func Test_GetAll(t *testing.T) {
	type fields struct {
		dataplaneKafkaService services.DataPlaneKafkaService
		kafkaService          services.KafkaService
	}

	type args struct {
		clusterId string
	}

	tests := []struct {
		name           string
		args           args
		fields         fields
		wantStatusCode int
	}{
		{
			name:           "empty cluster ID should fail validation",
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should fail when GetManagedKafkaByClusterID fails",
			args: args{
				clusterId: testId,
			},
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetManagedKafkaByClusterIDFunc: func(clusterID string) ([]v1.ManagedKafka, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to get kafka by cluster id")
					},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should successfully return ManagedKafkaList",
			args: args{
				clusterId: testId,
			},
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetManagedKafkaByClusterIDFunc: func(clusterID string) ([]v1.ManagedKafka, *errors.ServiceError) {
						return []v1.ManagedKafka{
							{
								Id: testId,
							},
						}, nil
					},
				},
			},
			wantStatusCode: http.StatusOK,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewDataPlaneKafkaHandler(tt.fields.dataplaneKafkaService, tt.fields.kafkaService)

			req, rw := GetHandlerParams("GET", "/{id}", nil)
			req = mux.SetURLVars(req, map[string]string{"id": tt.args.clusterId})

			h.GetAll(rw, req)
			resp := rw.Result()
			Expect(resp.StatusCode).To(Equal(tt.wantStatusCode))
			resp.Body.Close()
		})
	}
}
