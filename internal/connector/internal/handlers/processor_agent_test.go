package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/private"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreService "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
	"io"
	"testing"
)

type processorDeploymentTestFields struct {
	connectorClusterService services.ConnectorClusterService
	processorsService       services.ProcessorsService
}
type processorDeploymentTestAssertion func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields)

func toProcessorDeployment(body *[]byte, g *gomega.WithT) *private.ProcessorDeployment {
	var p private.ProcessorDeployment
	if err := json.Unmarshal(*body, &p); err != nil {
		g.Fail("Failed to unmarshall response.")
	}
	g.Expect(p).ToNot(gomega.BeNil())
	return &p
}

func toProcessorDeploymentList(body *[]byte, g *gomega.WithT) *private.ProcessorDeploymentList {
	var p private.ProcessorDeploymentList
	if err := json.Unmarshal(*body, &p); err != nil {
		g.Fail("Failed to unmarshall response.")
	}
	g.Expect(p).ToNot(gomega.BeNil())
	return &p
}

func Test_ProcessorAgent_ListProcessorDeployments(t *testing.T) {
	type args struct {
		clusterId string
	}

	type test struct {
		name           string
		args           args
		fields         processorDeploymentTestFields
		wantStatusCode int
		processorDeploymentTestAssertion
	}

	tests := []test{
		{
			name: "ClusterId::Not valid",
			args: args{
				clusterId: "",
			},
			wantStatusCode: 400,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			name: "Cluster::Unknown",
			args: args{
				clusterId: "unknown",
			},
			fields: processorDeploymentTestFields{
				connectorClusterService: &services.ConnectorClusterServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ConnectorCluster, *errors.ServiceError) {
						return nil, errors.NotFound("Not found")
					},
				},
			},
			wantStatusCode: 404,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("7"))
			},
		},
		{
			name: "ClusterId::Valid::With ProcessorDeployments",
			args: args{
				clusterId: "cluster-id",
			},
			fields: processorDeploymentTestFields{
				connectorClusterService: &services.ConnectorClusterServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ConnectorCluster, *errors.ServiceError) {
						return &dbapi.ConnectorCluster{}, nil
					},
					ListProcessorDeploymentsFunc: func(ctx context.Context, clusterId string, filterChannelUpdates bool, filterOperatorUpdates bool, includeDanglingDeploymentsOnly bool, listArgs *coreService.ListArguments, gtVersion int64) (dbapi.ProcessorDeploymentList, *api.PagingMeta, *errors.ServiceError) {
						processorDeployments := dbapi.ProcessorDeploymentList{*mockProcessorDeployment(map[string]interface{}{
							"cluster_id":              "cluster-id",
							"processor_id":            "processor-id",
							"processor_deployment_id": "processor-deployment-id"})}
						return processorDeployments, &api.PagingMeta{
							Page:  1,
							Size:  1,
							Total: 1,
						}, nil
					},
				},
				processorsService: &services.ProcessorsServiceMock{
					ResolveProcessorRefsWithBase64SecretsFunc: func(resource *dbapi.Processor) (bool, *errors.ServiceError) {
						return false, nil
					},
				}},
			wantStatusCode: 200,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				processorDeployments := toProcessorDeploymentList(body, g)
				g.Expect(processorDeployments.Size).To(gomega.Equal(int32(1)))
				g.Expect(processorDeployments.Items[0].Id).To(gomega.Equal("processor-deployment-id"))
				g.Expect(processorDeployments.Items[0].Spec.ProcessorId).To(gomega.Equal("processor-id"))
				g.Expect(processorDeployments.Items[0].Spec.ShardMetadata["key"]).To(gomega.Equal("value"))
			},
		},
		{
			name: "ClusterId::Valid::Without ProcessorDeployments",
			args: args{
				clusterId: "cluster-id",
			},
			fields: processorDeploymentTestFields{
				connectorClusterService: &services.ConnectorClusterServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ConnectorCluster, *errors.ServiceError) {
						return &dbapi.ConnectorCluster{}, nil
					},
					ListProcessorDeploymentsFunc: func(ctx context.Context, clusterId string, filterChannelUpdates bool, filterOperatorUpdates bool, includeDanglingDeploymentsOnly bool, listArgs *coreService.ListArguments, gtVersion int64) (dbapi.ProcessorDeploymentList, *api.PagingMeta, *errors.ServiceError) {
						return dbapi.ProcessorDeploymentList{}, &api.PagingMeta{
							Page:  1,
							Size:  0,
							Total: 0,
						}, nil
					},
				},
			},
			wantStatusCode: 200,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				g.Expect(toProcessorDeploymentList(body, g).Size).To(gomega.Equal(int32(0)))
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			cch := ConnectorClusterHandler{
				Service:    tt.fields.connectorClusterService,
				Processors: tt.fields.processorsService,
			}
			h := NewConnectorClusterHandler(cch)
			req, rw := GetHandlerParams("GET", "/", nil, t)
			req = mux.SetURLVars(req, map[string]string{"connector_cluster_id": tt.args.clusterId})
			h.ListProcessorDeployments(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			body, err := io.ReadAll(rw.Body)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(body).ToNot(gomega.BeNil())
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			tt.processorDeploymentTestAssertion(&body, g, tt.fields)
		})
	}
}

func Test_ProcessorAgent_GetProcessorDeployment(t *testing.T) {
	type args struct {
		clusterId             string
		processorDeploymentId string
	}

	type test struct {
		name           string
		args           args
		fields         processorDeploymentTestFields
		wantStatusCode int
		processorDeploymentTestAssertion
	}

	tests := []test{
		{
			name: "ClusterId::Not valid",
			args: args{
				clusterId: "",
			},
			wantStatusCode: 400,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			name: "Cluster::Unknown",
			args: args{
				clusterId: "unknown",
			},
			fields: processorDeploymentTestFields{
				connectorClusterService: &services.ConnectorClusterServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ConnectorCluster, *errors.ServiceError) {
						return nil, errors.NotFound("Not found")
					},
				},
			},
			wantStatusCode: 404,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("7"))
			},
		},
		{
			name: "ProcessorDeploymentId::Not valid",
			args: args{
				clusterId:             "cluster-id",
				processorDeploymentId: "",
			},
			fields: processorDeploymentTestFields{
				connectorClusterService: &services.ConnectorClusterServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ConnectorCluster, *errors.ServiceError) {
						return &dbapi.ConnectorCluster{}, nil
					},
				},
			},
			wantStatusCode: 400,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			name: "ProcessorDeployment::Unknown",
			args: args{
				clusterId:             "cluster-id",
				processorDeploymentId: "unknown",
			},
			fields: processorDeploymentTestFields{
				connectorClusterService: &services.ConnectorClusterServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ConnectorCluster, *errors.ServiceError) {
						return &dbapi.ConnectorCluster{}, nil
					},
					GetProcessorDeploymentFunc: func(ctx context.Context, id string) (*dbapi.ProcessorDeployment, *errors.ServiceError) {
						return nil, errors.NotFound("Not found")
					},
				},
			},
			wantStatusCode: 404,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("7"))
			},
		},
		{
			name: "ProcessorDeployment::Success",
			args: args{
				clusterId:             "cluster-id",
				processorDeploymentId: "processor-deployment-id",
			},
			fields: processorDeploymentTestFields{
				connectorClusterService: &services.ConnectorClusterServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ConnectorCluster, *errors.ServiceError) {
						return &dbapi.ConnectorCluster{}, nil
					},
					GetProcessorDeploymentFunc: func(ctx context.Context, id string) (*dbapi.ProcessorDeployment, *errors.ServiceError) {
						return mockProcessorDeployment(map[string]interface{}{
							"cluster_id":              "cluster-id",
							"processor_id":            "processor-id",
							"processor_deployment_id": "processor-deployment-id"}), nil
					},
				},
				processorsService: &services.ProcessorsServiceMock{
					ResolveProcessorRefsWithBase64SecretsFunc: func(resource *dbapi.Processor) (bool, *errors.ServiceError) {
						return false, nil
					},
				},
			},
			wantStatusCode: 200,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				processorDeployment := toProcessorDeployment(body, g)
				g.Expect(processorDeployment.Id).To(gomega.Equal("processor-deployment-id"))
				g.Expect(processorDeployment.Spec.ProcessorId).To(gomega.Equal("processor-id"))
				g.Expect(processorDeployment.Spec.ShardMetadata["key"]).To(gomega.Equal("value"))
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			cch := ConnectorClusterHandler{
				Service:    tt.fields.connectorClusterService,
				Processors: tt.fields.processorsService,
			}
			h := NewConnectorClusterHandler(cch)
			req, rw := GetHandlerParams("GET", "/", nil, t)
			req = mux.SetURLVars(req, map[string]string{"connector_cluster_id": tt.args.clusterId, "processor_deployment_id": tt.args.processorDeploymentId})
			h.GetProcessorDeployment(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			body, err := io.ReadAll(rw.Body)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(body).ToNot(gomega.BeNil())
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			tt.processorDeploymentTestAssertion(&body, g, tt.fields)
		})
	}
}

func Test_ProcessorAgent_UpdateProcessorDeploymentStatus(t *testing.T) {
	type args struct {
		clusterId             string
		processorDeploymentId string
		body                  []byte
	}

	type test struct {
		name           string
		args           args
		fields         processorDeploymentTestFields
		wantStatusCode int
		processorDeploymentTestAssertion
	}

	tests := []test{
		{
			name: "ClusterId::Not valid",
			args: args{
				clusterId: "",
				body:      []byte("{}"),
			},
			wantStatusCode: 400,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			name: "Cluster::Unknown",
			args: args{
				clusterId: "unknown",
				body:      []byte("{}"),
			},
			fields: processorDeploymentTestFields{
				connectorClusterService: &services.ConnectorClusterServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ConnectorCluster, *errors.ServiceError) {
						return nil, errors.NotFound("Not found")
					},
				},
			},
			wantStatusCode: 404,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("7"))
			},
		},
		{
			name: "ProcessorDeploymentId::Not valid",
			args: args{
				clusterId:             "cluster-id",
				processorDeploymentId: "",
				body:                  []byte("{}"),
			},
			fields: processorDeploymentTestFields{
				connectorClusterService: &services.ConnectorClusterServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ConnectorCluster, *errors.ServiceError) {
						return &dbapi.ConnectorCluster{}, nil
					},
				},
			},
			wantStatusCode: 400,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			name: "ProcessorDeployment::Unknown",
			args: args{
				clusterId:             "cluster-id",
				processorDeploymentId: "unknown",
				body:                  []byte("{\"phase\": \"ready\"}"),
			},
			fields: processorDeploymentTestFields{
				connectorClusterService: &services.ConnectorClusterServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ConnectorCluster, *errors.ServiceError) {
						return &dbapi.ConnectorCluster{}, nil
					},
					UpdateProcessorDeploymentStatusFunc: func(ctx context.Context, status dbapi.ProcessorDeploymentStatus) *errors.ServiceError {
						return errors.NotFound("Not found")
					},
				},
			},
			wantStatusCode: 404,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("7"))
			},
		},
		{
			name: "ProcessorDeployment::Success",
			args: args{
				clusterId:             "cluster-id",
				processorDeploymentId: "processor-deployment-id",
				body:                  []byte("{\"phase\": \"ready\"}"),
			},
			fields: processorDeploymentTestFields{
				connectorClusterService: &services.ConnectorClusterServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ConnectorCluster, *errors.ServiceError) {
						return &dbapi.ConnectorCluster{}, nil
					},
					UpdateProcessorDeploymentStatusFunc: func(ctx context.Context, status dbapi.ProcessorDeploymentStatus) *errors.ServiceError {
						return nil
					},
				},
			},
			wantStatusCode: 204,
			processorDeploymentTestAssertion: func(body *[]byte, g *gomega.WithT, fields processorDeploymentTestFields) {
				f, ok := fields.connectorClusterService.(*services.ConnectorClusterServiceMock)
				if !ok {
					g.Fail("Failed to get reference to ProcessorsServiceMock")
				}
				updateStatusCalls := f.UpdateProcessorDeploymentStatusCalls()
				g.Expect(len(updateStatusCalls)).To(gomega.Equal(1))
				g.Expect(updateStatusCalls[0].Status.ID).To(gomega.Equal("processor-deployment-id"))
				g.Expect(updateStatusCalls[0].Status.Phase).To(gomega.Equal(dbapi.ProcessorStatusPhaseReady))
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			cch := ConnectorClusterHandler{
				Service:    tt.fields.connectorClusterService,
				Processors: tt.fields.processorsService,
			}
			h := NewConnectorClusterHandler(cch)
			req, rw := GetHandlerParams("PUT", "/", bytes.NewBuffer(tt.args.body), t)
			req = mux.SetURLVars(req, map[string]string{"connector_cluster_id": tt.args.clusterId, "processor_deployment_id": tt.args.processorDeploymentId})
			h.UpdateProcessorDeploymentStatus(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			body, err := io.ReadAll(rw.Body)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(body).ToNot(gomega.BeNil())
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			tt.processorDeploymentTestAssertion(&body, g, tt.fields)
		})
	}

}

func mockProcessorDeployment(params map[string]interface{}) *dbapi.ProcessorDeployment {
	processorDeployment := &dbapi.ProcessorDeployment{
		Model: db.Model{
			ID: params["processor_deployment_id"].(string),
		},
		ProcessorID: params["processor_id"].(string),
		ClusterID:   params["cluster_id"].(string),
		Processor: dbapi.Processor{
			Model: db.Model{
				ID: params["processor_id"].(string),
			},
		},
		ProcessorShardMetadata: dbapi.ProcessorShardMetadata{
			ID:              0,
			ProcessorTypeId: "processor_0.1",
			Channel:         "",
			Revision:        0,
			LatestRevision:  nil,
			ShardMetadata:   api.JSON([]byte("{\"key\":\"value\"}")),
		},
	}
	return processorDeployment
}
