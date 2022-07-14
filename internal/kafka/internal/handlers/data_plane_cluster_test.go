package handlers

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"

	dataplanemocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/data_plane"
)

var (
	fullStatusRequest = `{"Conditions":[{"Type":"Ready","Status":"True"}],"Total":{"IngressEgressThroughputPerSec":"12345Mi","Connections":500,"DataRetentionSize":"12345Mi","Partitions":50},"NodeInfo":{"Ceiling":12,"Floor":3,"Current":6,"CurrentWorkLoadMinimum":6},"Remaining":{"Connections":400,"Partitions":40,"IngressEgressThroughputPerSec":"1234Mi","DataRetentionSize":"1234Mi"},"ResizeInfo":{"NodeDelta":6,"Delta":{"Connections":100,"Partitions":10,"IngressEgressThroughputPerSec":"1234Mi","DataRetentionSize":"1234Mi"}}}`
)

func Test_UpdateDataPlaneClusterStatus(t *testing.T) {
	type fields struct {
		dataplaneClusterService services.DataPlaneClusterService
	}

	type args struct {
		body []byte
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should return success status if passing correct and populated DataPlaneClusterStatus",
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{
					UpdateDataPlaneClusterStatusFunc: func(ctx context.Context, clusterID string, status *dbapi.DataPlaneClusterStatus) *errors.ServiceError {
						return nil
					},
				},
			},
			args: args{
				body: []byte(fullStatusRequest),
			},
			wantStatusCode: http.StatusNoContent,
		},
		{
			name: "should return an error if updating dataplane cluster status fails in the service",
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{
					UpdateDataPlaneClusterStatusFunc: func(ctx context.Context, clusterID string, status *dbapi.DataPlaneClusterStatus) *errors.ServiceError {
						return errors.GeneralError("failed to update dataplane status")
					},
				},
			},
			args: args{
				body: []byte(fullStatusRequest),
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewDataPlaneClusterHandler(tt.fields.dataplaneClusterService)
			req, rw := GetHandlerParams("GET", "/{id}", bytes.NewBuffer(tt.args.body), t)
			req = mux.SetURLVars(req, map[string]string{"id": "test-id"})
			h.UpdateDataPlaneClusterStatus(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_GetDataPlaneClusterConfig(t *testing.T) {
	type fields struct {
		dataplaneClusterService services.DataPlaneClusterService
	}

	tests := []struct {
		name           string
		fields         fields
		wantStatusCode int
	}{
		{
			name: "should return dataplane cluster config without an error",
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{
					GetDataPlaneClusterConfigFunc: func(ctx context.Context, clusterID string) (*dbapi.DataPlaneClusterConfig, *errors.ServiceError) {
						return &dbapi.DataPlaneClusterConfig{}, nil
					},
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should return an error if getting dataplane cluster config returns an error",
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{
					GetDataPlaneClusterConfigFunc: func(ctx context.Context, clusterID string) (*dbapi.DataPlaneClusterConfig, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to get dataplane cluster config")
					},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewDataPlaneClusterHandler(tt.fields.dataplaneClusterService)
			req, rw := GetHandlerParams("GET", "/{id}", nil, t)
			req = mux.SetURLVars(req, map[string]string{"id": "test-id"})
			h.GetDataPlaneClusterConfig(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_validateBody(t *testing.T) {
	type fields struct {
		dataplaneClusterService services.DataPlaneClusterService
	}

	type args struct {
		request *private.DataPlaneClusterUpdateStatusRequest
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "should return no error with valid dataplane cluster status when validating body",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(nil), //&private.DataPlaneClusterUpdateStatusRequest{}, //SampleDataPlaneclusterStatusRequestWithAvailableCapacity(),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: false,
		},
		{
			name: "should return an error with invalid Strimzi when validating body",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Strimzi = []private.DataPlaneClusterUpdateStatusRequestStrimzi{
						{
							Ready:            true,
							Version:          "",
							KafkaVersions:    []string{},
							KafkaIbpVersions: []string{},
						},
					}
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewDataPlaneClusterHandler(tt.fields.dataplaneClusterService)
			err := h.validateBody(tt.args.request)()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func Test_validateStrimzi(t *testing.T) {
	type fields struct {
		dataplaneClusterService services.DataPlaneClusterService
	}

	type args struct {
		request *private.DataPlaneClusterUpdateStatusRequest
	}

	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "should return no error with valid Strimzi in the dataplane cluster status",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(nil),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: false,
		},
		{
			name: "should return an error with empty Strimzi Version",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Strimzi = []private.DataPlaneClusterUpdateStatusRequestStrimzi{
						{
							Ready:            true,
							Version:          "",
							KafkaVersions:    []string{"2.8.0"},
							KafkaIbpVersions: []string{"2.8.0"},
						},
					}
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with empty KafkaVersions",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Strimzi = []private.DataPlaneClusterUpdateStatusRequestStrimzi{
						{
							Ready:            true,
							Version:          "2.8.0",
							KafkaVersions:    []string{},
							KafkaIbpVersions: []string{"2.8.0"},
						},
					}
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with empty KafkaIbpVersions",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Strimzi = []private.DataPlaneClusterUpdateStatusRequestStrimzi{
						{
							Ready:            true,
							Version:          "2.8.0",
							KafkaVersions:    []string{"2.8.0"},
							KafkaIbpVersions: []string{},
						},
					}
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewDataPlaneClusterHandler(tt.fields.dataplaneClusterService)
			g.Expect(h.validateStrimzi(tt.args.request) != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
