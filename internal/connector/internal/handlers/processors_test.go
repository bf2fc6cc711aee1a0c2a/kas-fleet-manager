package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/authz"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	mocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	coreService "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
	"github.com/openshift-online/ocm-sdk-go/authentication"
	"gorm.io/gorm"
	"io"
	"testing"
	"time"
)

type fields struct {
	processorsService services.ProcessorsService
	namespaceService  services.ConnectorNamespaceService
	vaultService      vault.VaultService
	authZService      authz.AuthZService
	processorsConfig  *config.ProcessorsConfig
}
type assertion func(body *[]byte, g *gomega.WithT, fields fields)

func toError(body *[]byte, g *gomega.WithT) api.Error {
	var e api.Error
	if err := json.Unmarshal(*body, &e); err != nil {
		g.Fail("Failed to unmarshall response.")
	}
	g.Expect(e).ToNot(gomega.BeNil())
	return e
}

func toProcessor(body *[]byte, g *gomega.WithT) *public.Processor {
	var p public.Processor
	if err := json.Unmarshal(*body, &p); err != nil {
		g.Fail("Failed to unmarshall response.")
	}
	g.Expect(p).ToNot(gomega.BeNil())
	return &p
}

func toProcessorList(body *[]byte, g *gomega.WithT) *public.ProcessorList {
	var p public.ProcessorList
	if err := json.Unmarshal(*body, &p); err != nil {
		g.Fail("Failed to unmarshall response.")
	}
	g.Expect(p).ToNot(gomega.BeNil())
	return &p
}

var (
	ctx = auth.SetTokenInContext(context.TODO(), &jwt.Token{
		Claims: jwt.MapClaims{
			"username":     "test-user",
			"org_id":       mocks.DefaultOrganisationId,
			"is_org_admin": true,
		},
	})
)

func Test_ProcessorsHandler_Get(t *testing.T) {
	type args struct {
		id string
	}

	type test struct {
		name           string
		args           args
		fields         fields
		wantStatusCode int
		assertion
	}

	tests := []test{
		{
			args:           args{id: ""},
			name:           "Processor Id not valid",
			wantStatusCode: 400,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			args: args{id: "not-found-processor-id"},
			name: "Processor Id not found",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
						return nil, errors.NotFound("Not found")
					},
				},
			},
			wantStatusCode: 404,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("7"))
			},
		},
		{
			args: args{id: "processor-id"},
			name: "Get:: Success",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
						return mockProcessorWithConditions(map[string]interface{}{"id": "processor-id"}), nil
					},
				},
			},
			wantStatusCode: 200,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toProcessor(body, g).Id).To(gomega.Equal("processor-id"))
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewProcessorsHandler(tt.fields.processorsService, tt.fields.namespaceService, tt.fields.vaultService, tt.fields.authZService, tt.fields.processorsConfig)
			req, rw := GetHandlerParams("GET", fmt.Sprintf("/%s", tt.args.id), nil, t)
			req = mux.SetURLVars(req, map[string]string{"processor_id": tt.args.id})
			h.Get(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			body, err := io.ReadAll(rw.Body)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(body).ToNot(gomega.BeNil())
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			tt.assertion(&body, g, tt.fields)
		})
	}
}

func Test_ProcessorsHandler_Delete(t *testing.T) {
	type args struct {
		id string
	}

	type test struct {
		name           string
		args           args
		fields         fields
		wantStatusCode int
		assertion
	}

	tests := []test{
		{
			args:           args{id: ""},
			name:           "Processor Id not valid",
			wantStatusCode: 400,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			args: args{id: "not-found-processor-id"},
			name: "Processor Id not found",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
						return nil, errors.NotFound("Not found")
					},
				},
			},
			wantStatusCode: 404,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("7"))
			},
		},
		{
			args: args{id: "processor-id"},
			name: "Delete:: Success",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
						return mockProcessorWithConditions(map[string]interface{}{
								"id":           "processor-id",
								"status":       dbapi.ProcessorStatusPhaseReady,
								"desiredState": dbapi.ProcessorReady}),
							nil
					},
					SaveStatusFunc: func(ctx context.Context, resource dbapi.ProcessorStatus) *errors.ServiceError {
						return nil
					},
					UpdateFunc: func(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {
						return nil
					},
				},
				namespaceService: &services.ConnectorNamespaceServiceMock{
					GetFunc: func(ctx context.Context, namespaceID string) (*dbapi.ConnectorNamespace, *errors.ServiceError) {
						return &dbapi.ConnectorNamespace{
							Status: dbapi.ConnectorNamespaceStatus{Phase: dbapi.ConnectorNamespacePhaseReady},
						}, nil
					},
				},
			},
			wantStatusCode: 204,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				f, ok := fields.processorsService.(*services.ProcessorsServiceMock)
				if !ok {
					g.Fail("Failed to get reference to ProcessorsServiceMock")
				}
				saveCalls := f.SaveStatusCalls()
				g.Expect(len(saveCalls)).To(gomega.Equal(1))
				g.Expect(saveCalls[0].Resource.Phase).To(gomega.Equal(dbapi.ProcessorStatusPhaseDeprovisioning))

				updates := f.UpdateCalls()
				g.Expect(len(updates)).To(gomega.Equal(1))
				g.Expect(updates[0].Resource.ID).To(gomega.Equal("processor-id"))
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewProcessorsHandler(tt.fields.processorsService, tt.fields.namespaceService, tt.fields.vaultService, tt.fields.authZService, tt.fields.processorsConfig)
			req, rw := GetHandlerParams("DELETE", fmt.Sprintf("/%s", tt.args.id), nil, t)
			req = mux.SetURLVars(req, map[string]string{"processor_id": tt.args.id})
			h.Delete(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			body, err := io.ReadAll(rw.Body)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(body).ToNot(gomega.BeNil())
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			tt.assertion(&body, g, tt.fields)
		})
	}
}

func Test_ProcessorsHandler_List(t *testing.T) {
	type args struct {
		url string
	}

	type test struct {
		name           string
		args           args
		fields         fields
		wantStatusCode int
		assertion
	}

	tests := []test{
		{
			name: "Successful list:: Empty results",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					ListFunc: func(ctx context.Context, listArgs *coreService.ListArguments, clusterId string) (dbapi.ProcessorWithConditionsList, *api.PagingMeta, *errors.ServiceError) {
						list := dbapi.ProcessorWithConditionsList{}
						meta := &api.PagingMeta{}
						return list, meta, nil
					},
				},
			},
			wantStatusCode: 200,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(len(toProcessorList(body, g).Items)).To(gomega.Equal(0))
			},
		},
		{
			name: "Successful list:: Basic results",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					ListFunc: func(ctx context.Context, listArgs *coreService.ListArguments, clusterId string) (dbapi.ProcessorWithConditionsList, *api.PagingMeta, *errors.ServiceError) {
						list := dbapi.ProcessorWithConditionsList{
							mockProcessorWithConditions(map[string]interface{}{
								"id": "processor-id1"}),
							mockProcessorWithConditions(map[string]interface{}{
								"id": "processor-id2"}),
						}
						meta := &api.PagingMeta{
							Page:  0,
							Size:  2,
							Total: 2,
						}
						return list, meta, nil
					},
				},
			},
			wantStatusCode: 200,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				list := toProcessorList(body, g)
				g.Expect(list.Page).To(gomega.Equal(int32(0)))
				g.Expect(list.Total).To(gomega.Equal(int32(2)))
				g.Expect(len(list.Items)).To(gomega.Equal(2))
			},
		},
		{
			name: "Successful list:: Criteria",
			args: args{
				url: "?page=1&size=2&search=search&orderBy=orderBy",
			},
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					ListFunc: func(ctx context.Context, listArgs *coreService.ListArguments, clusterId string) (dbapi.ProcessorWithConditionsList, *api.PagingMeta, *errors.ServiceError) {
						list := dbapi.ProcessorWithConditionsList{}
						meta := &api.PagingMeta{}
						return list, meta, nil
					},
				},
			},
			wantStatusCode: 200,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				f, ok := fields.processorsService.(*services.ProcessorsServiceMock)
				if !ok {
					g.Fail("Failed to get reference to ProcessorsServiceMock")
				}
				listCalls := f.ListCalls()
				g.Expect(len(listCalls)).To(gomega.Equal(1))
				g.Expect(listCalls[0].ListArgs.Page).To(gomega.Equal(1))
				g.Expect(listCalls[0].ListArgs.Size).To(gomega.Equal(2))
				g.Expect(listCalls[0].ListArgs.Search).To(gomega.Equal("search"))
				g.Expect(len(listCalls[0].ListArgs.OrderBy)).To(gomega.Equal(1))
				g.Expect(listCalls[0].ListArgs.OrderBy[0]).To(gomega.Equal("orderBy"))
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewProcessorsHandler(tt.fields.processorsService, tt.fields.namespaceService, tt.fields.vaultService, tt.fields.authZService, tt.fields.processorsConfig)
			req, rw := GetHandlerParams("GET", fmt.Sprintf("/%s", tt.args.url), nil, t)
			h.List(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			body, err := io.ReadAll(rw.Body)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(body).ToNot(gomega.BeNil())
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			tt.assertion(&body, g, tt.fields)
		})
	}
}

func Test_ProcessorsHandler_Update(t *testing.T) {
	type args struct {
		id          string
		body        []byte
		contentType string
		ctx         context.Context
	}

	type test struct {
		name           string
		args           args
		fields         fields
		wantStatusCode int
		assertion
	}

	token, _ := authentication.TokenFromContext(ctx)
	username := token.Claims.(jwt.MapClaims)["username"].(string)
	vaultService, _ := vault.NewTmpVaultService()
	// Authorization Service that approves the User defined for these tests
	authorizingAuthService := authz.NewAuthZService(nil,
		&services.ConnectorNamespaceServiceMock{
			GetNamespaceTenantFunc: func(namespaceId string) (*dbapi.ConnectorNamespace, *errors.ServiceError) {
				return &dbapi.ConnectorNamespace{
					TenantUserId: &username,
				}, nil
			},
			CheckConnectorQuotaFunc: func(namespaceId string) *errors.ServiceError {
				return nil
			}},
		nil)
	nameSpaceService := &services.ConnectorNamespaceServiceMock{
		GetFunc: func(ctx context.Context, namespaceID string) (*dbapi.ConnectorNamespace, *errors.ServiceError) {
			return &dbapi.ConnectorNamespace{
				Status: dbapi.ConnectorNamespaceStatus{Phase: dbapi.ConnectorNamespacePhaseReady},
			}, nil
		}}

	existingProcessor := func(status dbapi.ProcessorStatusPhase, desiredState dbapi.ProcessorDesiredState) *dbapi.ProcessorWithConditions {
		return mockProcessorWithConditions(map[string]interface{}{
			"id":                "processor-id",
			"name":              "name",
			"namespace_id":      "namespace_id",
			"processor_type_id": "processor_type_id",
			"status":            status,
			"desiredState":      desiredState,
			"kafka": dbapi.KafkaConnectionSettings{
				KafkaID:         "kafka-id",
				BootstrapServer: "kafka-url",
			},
			"serviceAccount": dbapi.ServiceAccount{
				ClientId:        "clientId",
				ClientSecret:    "clientSecret",
				ClientSecretRef: "originalSecretRef",
			}})
	}

	tests := []test{
		{
			args: args{
				id:  "",
				ctx: ctx,
			},
			name:           "Processor Id not valid",
			wantStatusCode: 400,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			args: args{
				id:          "processor-id",
				ctx:         ctx,
				contentType: "application/xml",
			},
			name:           "Content-Type header not valid",
			wantStatusCode: 400,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("21"))
			},
		},
		{
			args: args{
				id:          "processor-id",
				ctx:         ctx,
				contentType: "application/json",
			},
			name: "Non-existent Processor",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
						return nil, errors.NotFound("Not found")
					},
				},
			},
			wantStatusCode: 404,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("7"))
			},
		},
		{
			args: args{
				id:          "processor-id",
				ctx:         ctx,
				body:        []byte(`{"namespace_id": "new-namespace_id"}`),
				contentType: "application/json",
			},
			name: "Immutable field:: namespace_id",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
						return existingProcessor(dbapi.ProcessorStatusPhaseReady, dbapi.ProcessorReady), nil
					},
					UpdateFunc: func(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {
						return nil
					},
				},
				authZService:     authorizingAuthService,
				namespaceService: nameSpaceService,
				vaultService:     vaultService,
				processorsConfig: &config.ProcessorsConfig{
					ProcessorsEnabled: true,
				},
			},
			wantStatusCode: 409,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("21"))
			},
		},
		{
			args: args{
				id:          "processor-id",
				ctx:         ctx,
				body:        []byte(`{"processor_type_id": "new-processor_type_id"}`),
				contentType: "application/json",
			},
			name: "Immutable field:: processor_type_id",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
						return existingProcessor(dbapi.ProcessorStatusPhaseReady, dbapi.ProcessorReady), nil
					},
					UpdateFunc: func(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {
						return nil
					},
				},
				authZService:     authorizingAuthService,
				namespaceService: nameSpaceService,
				vaultService:     vaultService,
				processorsConfig: &config.ProcessorsConfig{
					ProcessorsEnabled: true,
				},
			},
			wantStatusCode: 409,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("21"))
			},
		},
		{
			args: args{
				id:          "processor-id",
				ctx:         ctx,
				body:        []byte(`{"channel": "beta"}`),
				contentType: "application/json",
			},
			name: "Immutable field:: channel",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
						return existingProcessor(dbapi.ProcessorStatusPhaseReady, dbapi.ProcessorReady), nil
					},
					UpdateFunc: func(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {
						return nil
					},
				},
				authZService:     authorizingAuthService,
				namespaceService: nameSpaceService,
				vaultService:     vaultService,
				processorsConfig: &config.ProcessorsConfig{
					ProcessorsEnabled: true,
				},
			},
			wantStatusCode: 409,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("21"))
			},
		},
		{
			args: args{
				id:          "processor-id",
				ctx:         ctx,
				body:        []byte(`{"name": "new-name", "definition": { "from": { "uri": "kafka:my-topic", "steps": [] }} }`),
				contentType: APPLICATION_JSON,
			},
			name: "Update:: Successful:: Ready",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
						return existingProcessor(dbapi.ProcessorStatusPhaseReady, dbapi.ProcessorReady), nil
					},
					UpdateFunc: func(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {
						return nil
					},
					SaveStatusFunc: func(ctx context.Context, resource dbapi.ProcessorStatus) *errors.ServiceError {
						return nil
					},
				},
				authZService:     authorizingAuthService,
				namespaceService: nameSpaceService,
				vaultService:     vaultService,
				processorsConfig: &config.ProcessorsConfig{
					ProcessorsEnabled: true,
				},
			},
			wantStatusCode: 202,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				f, ok := fields.processorsService.(*services.ProcessorsServiceMock)
				if !ok {
					g.Fail("Failed to get reference to ProcessorsServiceMock")
				}
				saveStatusCalls := f.SaveStatusCalls()
				g.Expect(len(saveStatusCalls)).To(gomega.Equal(1))
				g.Expect(saveStatusCalls[0].Resource.Phase).To(gomega.Equal(dbapi.ProcessorStatusPhaseUpdating))
				updateCalls := f.UpdateCalls()
				g.Expect(len(updateCalls)).To(gomega.Equal(1))
				g.Expect(updateCalls[0].Resource.ServiceAccount.ClientSecretRef).ToNot(gomega.Equal("originalSecretRef"))

				p := toProcessor(body, g)
				g.Expect(p.Id).To(gomega.Equal("processor-id"))
				g.Expect(p.Name).To(gomega.Equal("new-name"))
				g.Expect(p.Definition).ToNot(gomega.BeNil())
				g.Expect(p.Definition["from"]).ToNot(gomega.BeNil())
				g.Expect(p.Status.State).To(gomega.Equal(public.PROCESSORSTATE_UPDATING))
				g.Expect(p.ServiceAccount.ClientSecret).To(gomega.Equal(""))
			},
		},
		{
			args: args{
				id:          "processor-id",
				ctx:         ctx,
				body:        []byte(`{"name": "new-name", "definition": { "from": { "uri": "kafka:my-topic", "steps": [] }} }`),
				contentType: APPLICATION_JSON,
			},
			name: "Update:: Successful:: Preparing",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
						return existingProcessor(dbapi.ProcessorStatusPhasePreparing, dbapi.ProcessorReady), nil
					},
					UpdateFunc: func(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {
						return nil
					},
					SaveStatusFunc: func(ctx context.Context, resource dbapi.ProcessorStatus) *errors.ServiceError {
						return nil
					},
				},
				authZService:     authorizingAuthService,
				namespaceService: nameSpaceService,
				vaultService:     vaultService,
				processorsConfig: &config.ProcessorsConfig{
					ProcessorsEnabled: true,
				},
			},
			wantStatusCode: 202,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				f, ok := fields.processorsService.(*services.ProcessorsServiceMock)
				if !ok {
					g.Fail("Failed to get reference to ProcessorsServiceMock")
				}
				saveStatusCalls := f.SaveStatusCalls()
				g.Expect(len(saveStatusCalls)).To(gomega.Equal(0))
				p := toProcessor(body, g)
				g.Expect(p.Status.State).To(gomega.Equal(public.PROCESSORSTATE_PREPARING))
			},
		},
		{
			args: args{
				id:          "processor-id",
				ctx:         ctx,
				body:        []byte(`{"name": "new-name", "definition": { "from": { "uri": "kafka:my-topic", "steps": [] }} }`),
				contentType: APPLICATION_JSON,
			},
			name: "Update:: Successful:: Deleting",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
						return existingProcessor(dbapi.ProcessorStatusPhaseDeleting, dbapi.ProcessorReady), nil
					},
					UpdateFunc: func(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {
						return nil
					},
					SaveStatusFunc: func(ctx context.Context, resource dbapi.ProcessorStatus) *errors.ServiceError {
						return nil
					},
				},
				authZService:     authorizingAuthService,
				namespaceService: nameSpaceService,
				vaultService:     vaultService,
				processorsConfig: &config.ProcessorsConfig{
					ProcessorsEnabled: true,
				},
			},
			wantStatusCode: 202,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				f, ok := fields.processorsService.(*services.ProcessorsServiceMock)
				if !ok {
					g.Fail("Failed to get reference to ProcessorsServiceMock")
				}
				saveStatusCalls := f.SaveStatusCalls()
				g.Expect(len(saveStatusCalls)).To(gomega.Equal(1))
				g.Expect(saveStatusCalls[0].Resource.Phase).To(gomega.Equal(dbapi.ProcessorStatusPhaseUpdating))
				p := toProcessor(body, g)
				g.Expect(p.Status.State).To(gomega.Equal(public.PROCESSORSTATE_UPDATING))
			},
		},
		{
			args: args{
				id:          "processor-id",
				ctx:         ctx,
				body:        []byte(`{"name": "new-name", "definition": { "from": { "uri": "kafka:my-topic", "steps": [] }} }`),
				contentType: APPLICATION_JSON,
			},
			name: "Update:: Successful:: Deleted",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
						return existingProcessor(dbapi.ProcessorStatusPhaseDeleted, dbapi.ProcessorReady), nil
					},
					UpdateFunc: func(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {
						return nil
					},
					SaveStatusFunc: func(ctx context.Context, resource dbapi.ProcessorStatus) *errors.ServiceError {
						return nil
					},
				},
				authZService:     authorizingAuthService,
				namespaceService: nameSpaceService,
				vaultService:     vaultService,
				processorsConfig: &config.ProcessorsConfig{
					ProcessorsEnabled: true,
				},
			},
			wantStatusCode: 202,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				f, ok := fields.processorsService.(*services.ProcessorsServiceMock)
				if !ok {
					g.Fail("Failed to get reference to ProcessorsServiceMock")
				}
				saveStatusCalls := f.SaveStatusCalls()
				g.Expect(len(saveStatusCalls)).To(gomega.Equal(1))
				g.Expect(saveStatusCalls[0].Resource.Phase).To(gomega.Equal(dbapi.ProcessorStatusPhaseUpdating))
				p := toProcessor(body, g)
				g.Expect(p.Status.State).To(gomega.Equal(public.PROCESSORSTATE_UPDATING))
			},
		},
		{
			args: args{
				id:          "processor-id",
				ctx:         ctx,
				body:        []byte(`{"name": "new-name", "definition": { "from": { "uri": "kafka:my-topic", "steps": [] }} }`),
				contentType: APPLICATION_JSON,
			},
			name: "Update:: Successful:: Deprovisioning",
			fields: fields{
				processorsService: &services.ProcessorsServiceMock{
					GetFunc: func(ctx context.Context, id string) (*dbapi.ProcessorWithConditions, *errors.ServiceError) {
						return existingProcessor(dbapi.ProcessorStatusPhaseDeprovisioning, dbapi.ProcessorReady), nil
					},
					UpdateFunc: func(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {
						return nil
					},
					SaveStatusFunc: func(ctx context.Context, resource dbapi.ProcessorStatus) *errors.ServiceError {
						return nil
					},
				},
				authZService:     authorizingAuthService,
				namespaceService: nameSpaceService,
				vaultService:     vaultService,
				processorsConfig: &config.ProcessorsConfig{
					ProcessorsEnabled: true,
				},
			},
			wantStatusCode: 202,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				f, ok := fields.processorsService.(*services.ProcessorsServiceMock)
				if !ok {
					g.Fail("Failed to get reference to ProcessorsServiceMock")
				}
				saveStatusCalls := f.SaveStatusCalls()
				g.Expect(len(saveStatusCalls)).To(gomega.Equal(1))
				g.Expect(saveStatusCalls[0].Resource.Phase).To(gomega.Equal(dbapi.ProcessorStatusPhaseUpdating))
				p := toProcessor(body, g)
				g.Expect(p.Status.State).To(gomega.Equal(public.PROCESSORSTATE_UPDATING))
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewProcessorsHandler(tt.fields.processorsService, tt.fields.namespaceService, tt.fields.vaultService, tt.fields.authZService, tt.fields.processorsConfig)
			req, rw := GetHandlerParams("PATCH", fmt.Sprintf("/%s", tt.args.id), bytes.NewBuffer(tt.args.body), t)
			req = req.WithContext(tt.args.ctx)
			req = mux.SetURLVars(req, map[string]string{"processor_id": tt.args.id})
			req.Header.Add("Content-Type", tt.args.contentType)
			h.Patch(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			body, err := io.ReadAll(rw.Body)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(body).ToNot(gomega.BeNil())
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			tt.assertion(&body, g, tt.fields)
		})
	}
}

func Test_ProcessorsHandler_Create(t *testing.T) {
	type args struct {
		url  string
		body []byte
		ctx  context.Context
	}

	type test struct {
		name           string
		args           args
		fields         fields
		wantStatusCode int
		assertion
	}

	token, _ := authentication.TokenFromContext(ctx)
	username := token.Claims.(jwt.MapClaims)["username"].(string)
	vaultService, _ := vault.NewTmpVaultService()
	// Authorization Service that approves the User defined for these tests
	authorizingAuthService := authz.NewAuthZService(nil,
		&services.ConnectorNamespaceServiceMock{
			GetNamespaceTenantFunc: func(namespaceId string) (*dbapi.ConnectorNamespace, *errors.ServiceError) {
				return &dbapi.ConnectorNamespace{
					TenantUserId: &username,
				}, nil
			},
			CheckConnectorQuotaFunc: func(namespaceId string) *errors.ServiceError {
				return nil
			}},
		nil)

	tests := []test{
		{
			name: "Async request",
			args: args{
				url:  "",
				body: []byte(`{}`),
				ctx:  ctx,
			},
			fields: fields{
				authZService: authz.NewAuthZService(nil, nil, nil),
			},
			wantStatusCode: 400,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("103"))
			},
		},
		{
			name: "Processor name invalid",
			args: args{
				url:  "?async=true",
				body: []byte(`{"name": "", "namespace_id": "namespace_id"}`),
				ctx:  ctx,
			},
			fields: fields{
				authZService: authorizingAuthService,
			},
			wantStatusCode: 400,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			name: "Kafka.Id invalid",
			args: args{
				url:  "?async=true",
				body: []byte(`{"name": "name", "namespace_id": "namespace_id", "kafka": {"id": "", "url": "kafka_url"}, "service_account": {"client_id": "client_id", "client_secret": "client_secret"} }`),
				ctx:  ctx,
			},
			fields: fields{
				authZService: authorizingAuthService,
			},
			wantStatusCode: 400,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			name: "Kafka.Url invalid",
			args: args{
				url:  "?async=true",
				body: []byte(`{"name": "name", "namespace_id": "namespace_id", "kafka": {"id": "kafka_id", "url": ""}, "service_account": {"client_id": "client_id", "client_secret": "client_secret"} }`),
				ctx:  ctx,
			},
			fields: fields{
				authZService: authorizingAuthService,
			},
			wantStatusCode: 400,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			name: "ServiceAccount.ClientId invalid",
			args: args{
				url:  "?async=true",
				body: []byte(`{"name": "name", "namespace_id": "namespace_id", "kafka": {"id": "kafka_id", "url": "kafka_url"}, "service_account": {"client_id": ""} }`),
				ctx:  ctx,
			},
			fields: fields{
				authZService: authorizingAuthService,
			},
			wantStatusCode: 400,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			name: "ServiceAccount.ClientSecret invalid",
			args: args{
				url:  "?async=true",
				body: []byte(`{"name": "name", "namespace_id": "namespace_id", "kafka": {"id": "kafka_id", "url": "kafka_url"}, "service_account": {"client_id": "client_id", "client_secret": ""} }`),
				ctx:  ctx,
			},
			fields: fields{
				authZService: authorizingAuthService,
			},
			wantStatusCode: 400,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			name: "DesiredState invalid",
			args: args{
				url:  "?async=true",
				body: []byte(`{"name": "name", "namespace_id": "namespace_id", "kafka": {"id": "kafka_id", "url": "kafka_url"}, "service_account": {"client_id": "client_id", "client_secret": "client_secret"}, "desired_state": "failed" }`),
				ctx:  ctx,
			},
			fields: fields{
				authZService: authorizingAuthService,
			},
			wantStatusCode: 400,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("21"))
			},
		},
		{
			name: "NamespaceId invalid",
			args: args{
				url:  "?async=true",
				body: []byte(`{"name": "name", "namespace_id": "", "kafka": {"id": "kafka_id", "url": "kafka_url"}, "service_account": {"client_id": "client_id", "client_secret": "client_secret"} }`),
				ctx:  ctx,
			},
			fields: fields{
				authZService: authorizingAuthService,
				processorsConfig: &config.ProcessorsConfig{
					ProcessorsEnabled: true,
				},
			},
			wantStatusCode: 400,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("33"))
			},
		},
		{
			name: "NamespaceId not authorised",
			args: args{
				url:  "?async=true",
				body: []byte(`{"name": "name", "namespace_id": "namespace_id", "kafka": {"id": "kafka_id", "url": "kafka_url"}, "service_account": {"client_id": "client_id", "client_secret": "client_secret"} }`),
				ctx:  ctx,
			},
			fields: fields{
				authZService: authz.NewAuthZService(nil,
					&services.ConnectorNamespaceServiceMock{
						GetNamespaceTenantFunc: func(namespaceId string) (*dbapi.ConnectorNamespace, *errors.ServiceError) {
							return nil, errors.Unauthorized("Unauthorised")
						}},
					nil),
			},
			wantStatusCode: 400,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("21"))
			},
		},
		{
			name: "NamespaceId no quota",
			args: args{
				url:  "?async=true",
				body: []byte(`{"name": "name", "namespace_id": "namespace_id", "kafka": {"id": "kafka_id", "url": "kafka_url"}, "service_account": {"client_id": "client_id", "client_secret": "client_secret"} }`),
				ctx:  ctx,
			},
			fields: fields{
				authZService: authz.NewAuthZService(nil,
					&services.ConnectorNamespaceServiceMock{
						GetNamespaceTenantFunc: func(namespaceId string) (*dbapi.ConnectorNamespace, *errors.ServiceError) {
							return &dbapi.ConnectorNamespace{
								TenantUserId: &username,
							}, nil
						},
						CheckConnectorQuotaFunc: func(namespaceId string) *errors.ServiceError {
							return errors.InsufficientQuotaError("Insufficient quota")
						}},
					nil),
			},
			wantStatusCode: 403,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				g.Expect(toError(body, g).ID).To(gomega.Equal("120"))
			},
		},
		{
			name: "Creation:: Success",
			args: args{
				url:  "?async=true",
				body: []byte(`{"name": "name", "namespace_id": "namespace_id", "kafka": {"id": "kafka_id", "url": "kafka_url"}, "service_account": {"client_id": "client_id", "client_secret": "client_secret"}, "definition": { "from": { "uri": "kafka:my-topic", "steps": [] }} }`),
				ctx:  ctx,
			},
			fields: fields{
				authZService: authorizingAuthService,
				processorsConfig: &config.ProcessorsConfig{
					ProcessorsEnabled: true,
				},
				namespaceService: &services.ConnectorNamespaceServiceMock{
					GetFunc: func(ctx context.Context, namespaceID string) (*dbapi.ConnectorNamespace, *errors.ServiceError) {
						return &dbapi.ConnectorNamespace{
							Status: dbapi.ConnectorNamespaceStatus{Phase: dbapi.ConnectorNamespacePhaseReady},
						}, nil
					},
				},
				processorsService: &services.ProcessorsServiceMock{
					CreateFunc: func(ctx context.Context, resource *dbapi.Processor) *errors.ServiceError {
						return nil
					},
				},
				vaultService: vaultService,
			},
			wantStatusCode: 202,
			assertion: func(body *[]byte, g *gomega.WithT, fields fields) {
				processor := toProcessor(body, g)
				g.Expect(processor).NotTo(gomega.BeNil())
				g.Expect(processor.Id).NotTo(gomega.BeNil())
				g.Expect(processor.Owner).To(gomega.Equal(username))
				g.Expect(processor.Name).To(gomega.Equal("name"))
				g.Expect(processor.NamespaceId).To(gomega.Equal("namespace_id"))
				g.Expect(processor.DesiredState).To(gomega.Equal(public.PROCESSORDESIREDSTATE_READY))
				g.Expect(len(processor.Annotations)).To(gomega.Equal(1))
				g.Expect(processor.Annotations[dbapi.ConnectorClusterOrgIdAnnotation]).To(gomega.Equal(mocks.DefaultOrganisationId))
				g.Expect(processor.ServiceAccount.ClientId).To(gomega.Equal("client_id"))
				g.Expect(processor.ServiceAccount.ClientSecret).To(gomega.Equal(""))
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			h := NewProcessorsHandler(tt.fields.processorsService, tt.fields.namespaceService, tt.fields.vaultService, tt.fields.authZService, tt.fields.processorsConfig)
			req, rw := GetHandlerParams("CREATE", tt.args.url, bytes.NewBuffer(tt.args.body), t)
			req = req.WithContext(tt.args.ctx)
			h.Create(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			body, err := io.ReadAll(rw.Body)
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(body).ToNot(gomega.BeNil())
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			tt.assertion(&body, g, tt.fields)
		})
	}
}

func mockProcessorWithConditions(params map[string]interface{}) *dbapi.ProcessorWithConditions {
	processor := &dbapi.ProcessorWithConditions{
		Processor: dbapi.Processor{
			Model: db.Model{
				ID:        params["id"].(string),
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
				DeletedAt: gorm.DeletedAt{},
			},
			Owner:          "",
			OrganisationId: "",
			Version:        0,
			Annotations:    nil,
			Definition:     nil,
			ErrorHandler:   nil,
			Channel:        "",
			Kafka:          dbapi.KafkaConnectionSettings{},
			ServiceAccount: dbapi.ServiceAccount{},
			Status:         dbapi.ProcessorStatus{},
		},
		Conditions: nil,
	}
	if exists(params, "name") {
		processor.Name = extract(params, "name").(string)
	}
	if exists(params, "namespace_id") {
		processor.NamespaceId = extract(params, "namespace_id").(string)
	}
	if exists(params, "processor_type_id") {
		processor.ProcessorTypeId = extract(params, "processor_type_id").(string)
	}
	if exists(params, "status") {
		processor.Status.Phase = extract(params, "status").(dbapi.ProcessorStatusPhase)
	}
	if exists(params, "desiredState") {
		processor.DesiredState = extract(params, "desiredState").(dbapi.ProcessorDesiredState)
	}
	if exists(params, "kafka") {
		processor.Kafka = extract(params, "kafka").(dbapi.KafkaConnectionSettings)
	}
	if exists(params, "serviceAccount") {
		processor.ServiceAccount = extract(params, "serviceAccount").(dbapi.ServiceAccount)
	}
	return processor
}

func exists(params map[string]interface{}, key string) bool {
	return params[key] != nil
}

func extract(params map[string]interface{}, key string) interface{} {
	if value := params[key]; value != nil {
		switch t := value.(type) {
		default:
			fmt.Printf("unexpected type %T", t) // %T prints whatever type t has
		case string:
			return value
		case dbapi.ProcessorStatusPhase:
			return value
		case dbapi.ProcessorDesiredState:
			return value
		case dbapi.KafkaConnectionSettings:
			return value
		case dbapi.ServiceAccount:
			return value
		}
	}
	return nil
}
