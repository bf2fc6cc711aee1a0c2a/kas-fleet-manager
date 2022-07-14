package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	v1 "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/managedkafkas.managedkafka.bf2.org/v1"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/onsi/gomega"
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			h := NewDataPlaneKafkaHandler(tt.fields.dataplaneKafkaService, tt.fields.kafkaService)

			req, rw := GetHandlerParams("GET", "/{id}", bytes.NewBuffer(tt.args.body), t)
			req = mux.SetURLVars(req, map[string]string{"id": testId})

			h.UpdateKafkaStatuses(rw, req)
			resp := rw.Result()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
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
		wantKafkaIDs   []string
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
			name: "should fail when generation of reserved managed kafka instances fails",
			args: args{
				clusterId: testId,
			},
			fields: fields{
				kafkaService: &services.KafkaServiceMock{
					GetManagedKafkaByClusterIDFunc: func(clusterID string) ([]v1.ManagedKafka, *errors.ServiceError) {
						return []v1.ManagedKafka{{Id: testId}}, nil
					},
					GenerateReservedManagedKafkasByClusterIDFunc: func(clusterID string) ([]v1.ManagedKafka, *errors.ServiceError) {
						return nil, errors.GeneralError("test fail")
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
								ObjectMeta: metav1.ObjectMeta{
									Annotations: map[string]string{
										"bf2.org/id": testId,
									},
								},
							},
						}, nil
					},
					GenerateReservedManagedKafkasByClusterIDFunc: func(clusterID string) ([]v1.ManagedKafka, *errors.ServiceError) {
						return []v1.ManagedKafka{
							v1.ManagedKafka{
								Id: "reserved-kafka-test-1",
								ObjectMeta: metav1.ObjectMeta{
									Annotations: map[string]string{
										"bf2.org/id": "reserved-kafka-test-1",
									},
								},
							},
						}, nil
					},
				},
			},
			wantStatusCode: http.StatusOK,
			wantKafkaIDs:   []string{testId, "reserved-kafka-test-1"},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewDataPlaneKafkaHandler(tt.fields.dataplaneKafkaService, tt.fields.kafkaService)

			req, rw := GetHandlerParams("GET", "/{id}", nil, t)
			req = mux.SetURLVars(req, map[string]string{"id": tt.args.clusterId})

			h.GetAll(rw, req)
			resp := rw.Result()
			var respBody private.ManagedKafkaList
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			decodeErr := json.NewDecoder(resp.Body).Decode(&respBody)
			g.Expect(decodeErr).NotTo(gomega.HaveOccurred())
			g.Expect(respBody.Items).Should(gomega.HaveLen(len(tt.wantKafkaIDs)))
			for idx, managedkafka := range respBody.Items {
				g.Expect(managedkafka.Id).To(gomega.Equal(tt.wantKafkaIDs[idx]))
			}

			resp.Body.Close()
		})
	}
}
