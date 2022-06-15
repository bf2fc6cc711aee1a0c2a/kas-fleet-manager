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
	. "github.com/onsi/gomega"

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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			h := NewDataPlaneClusterHandler(tt.fields.dataplaneClusterService)
			req, rw := GetHandlerParams("GET", "/{id}", bytes.NewBuffer(tt.args.body))
			req = mux.SetURLVars(req, map[string]string{"id": "test-id"})
			h.UpdateDataPlaneClusterStatus(rw, req)
			resp := rw.Result()
			Expect(resp.StatusCode).To(Equal(tt.wantStatusCode))
			resp.Body.Close()
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			h := NewDataPlaneClusterHandler(tt.fields.dataplaneClusterService)
			req, rw := GetHandlerParams("GET", "/{id}", nil)
			req = mux.SetURLVars(req, map[string]string{"id": "test-id"})
			h.GetDataPlaneClusterConfig(rw, req)
			resp := rw.Result()
			Expect(resp.StatusCode).To(Equal(tt.wantStatusCode))
			resp.Body.Close()
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
			name: "should return an error with invalid NodeInfo when validating body",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.NodeInfo.Ceiling = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid ResizeInfo when validating body",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.ResizeInfo.NodeDelta = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid Total when validating body",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Total.Connections = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid Remaining when validating body",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Remaining.Connections = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			h := NewDataPlaneClusterHandler(tt.fields.dataplaneClusterService)
			err := h.validateBody(tt.args.request)()
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_validateNodeInfo(t *testing.T) {
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
			name: "should return no error with valid NodeInfo in the dataplane cluster status",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(nil),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: false,
		},
		{
			name: "should return an error with invalid Ceiling in the NodeInfo",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.NodeInfo.Ceiling = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid Current in the NodeInfo",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.NodeInfo.Current = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid CurrentWorkLoadMinimum in the NodeInfo",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.NodeInfo.CurrentWorkLoadMinimum = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid Floor in the NodeInfo",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.NodeInfo.Floor = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			h := NewDataPlaneClusterHandler(tt.fields.dataplaneClusterService)
			Expect(h.validateNodeInfo(tt.args.request) != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_validateResizeInfo(t *testing.T) {
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
			name: "should return no error with valid ResizeInfo in the dataplane cluster status",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(nil),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: false,
		},
		{
			name: "should return an error with invalid NodeDelta in the ResizeInfo",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.ResizeInfo.NodeDelta = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid Delta in the ResizeInfo",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.ResizeInfo.Delta = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid Delta.Connections in the ResizeInfo",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.ResizeInfo.Delta.Connections = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid Delta.DataRetentionSize in the ResizeInfo",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.ResizeInfo.Delta.DataRetentionSize = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid Delta.IngressEgressThroughputPerSec in the ResizeInfo",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.ResizeInfo.Delta.IngressEgressThroughputPerSec = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid Delta.Partitions in the ResizeInfo",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.ResizeInfo.Delta.Partitions = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			h := NewDataPlaneClusterHandler(tt.fields.dataplaneClusterService)
			Expect(h.validateResizeInfo(tt.args.request) != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_validateTotal(t *testing.T) {
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
			name: "should return no error with valid Total in the dataplane cluster status",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(nil),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: false,
		},
		{
			name: "should return an error with invalid DataRetentionSize in the Total",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Total.DataRetentionSize = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid Connections in the Total",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Total.Connections = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid IngressEgressThroughputPerSec in the Total",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Total.IngressEgressThroughputPerSec = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid Partitions in the Total",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Total.Partitions = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			h := NewDataPlaneClusterHandler(tt.fields.dataplaneClusterService)
			Expect(h.validateTotal(tt.args.request) != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_validateRemaining(t *testing.T) {
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
			name: "should return no error with valid Remaining in the dataplane cluster status in the Remaining",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(nil),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: false,
		},
		{
			name: "should return an error with invalid DataRetentionSize in the Remaining",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Remaining.DataRetentionSize = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid Connections in the Remaining",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Remaining.Connections = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid IngressEgressThroughputPerSec in the Remaining",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Remaining.IngressEgressThroughputPerSec = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
		{
			name: "should return an error with invalid Partitions in the Remaining",
			args: args{
				request: dataplanemocks.BuildValidDataPlaneClusterUpdateStatusRequest(func(statusRequest *private.DataPlaneClusterUpdateStatusRequest) {
					statusRequest.Remaining.Partitions = nil
				}),
			},
			fields: fields{
				dataplaneClusterService: &services.DataPlaneClusterServiceMock{},
			},
			wantErr: true,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			h := NewDataPlaneClusterHandler(tt.fields.dataplaneClusterService)
			Expect(h.validateRemaining(tt.args.request) != nil).To(Equal(tt.wantErr))
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			h := NewDataPlaneClusterHandler(tt.fields.dataplaneClusterService)
			Expect(h.validateStrimzi(tt.args.request) != nil).To(Equal(tt.wantErr))
		})
	}
}
