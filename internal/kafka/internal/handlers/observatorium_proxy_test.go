package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"

	stderr "errors"
)

func contextWithClientIDinClaims(clientID string) context.Context {
	return auth.SetTokenInContext(context.TODO(), &jwt.Token{
		Claims: jwt.MapClaims{
			"clientId": clientID,
		},
	})
}

func Test_ValidateTokenAndExternalClusterID(t *testing.T) {
	type fields struct {
		clusterService services.ClusterService
	}

	type args struct {
		ctx        context.Context
		externalID string
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{

		{
			name: "should fail if external cluster ID is not valid",
			args: args{
				externalID: "invalid",
				ctx:        context.TODO(),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should fail if the context is empty",
			args: args{
				externalID: validFormatExternalClusterId,
				ctx:        context.TODO(),
			},
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name: "should fail if FindCluster returns an error",
			args: args{
				externalID: validFormatExternalClusterId,
				ctx:        contextWithClientIDinClaims("test"),
			},
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, error) {
						return nil, errors.GeneralError("failed to perform this action")
					},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should return not found error if cluster cannot be found",
			args: args{
				externalID: validFormatExternalClusterId,
				ctx:        contextWithClientIDinClaims("test"),
			},
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, error) {
						return nil, nil
					},
				},
			},
			wantStatusCode: http.StatusNotFound,
		},
		{
			name: "should return forbidden error if there is a mismatch between client ID in the claims and in the clusters table",
			args: args{
				externalID: validFormatExternalClusterId,
				ctx:        contextWithClientIDinClaims("test"),
			},
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, error) {
						return &api.Cluster{
							ExternalID: validFormatExternalClusterId,
							ClientID:   "some-client-id",
						}, nil
					},
				},
			},
			wantStatusCode: http.StatusForbidden,
		},
		{
			name: "successful validation of client ID from the claims and cluster external ID from the url param",
			args: args{
				externalID: validFormatExternalClusterId,
				ctx:        contextWithClientIDinClaims("test"),
			},
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, error) {
						return &api.Cluster{
							ExternalID: validFormatExternalClusterId,
							ClientID:   "test",
						}, nil
					},
				},
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "successful validation should not depend on cluster multi-az value",
			args: args{
				externalID: validFormatExternalClusterId,
				ctx:        contextWithClientIDinClaims("test"),
			},
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterFunc: func(criteria services.FindClusterCriteria) (*api.Cluster, error) {
						if criteria.MultiAZ != false {
							return nil, stderr.New("multi-az criteria must not be set")
						}

						return &api.Cluster{
							ExternalID: validFormatExternalClusterId,
							ClientID:   "test",
						}, nil
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
			h := NewObservatoriumProxyHandler(tt.fields.clusterService)
			req, rw := GetHandlerParams("POST", "/{cluster_external_id}", nil, t)
			req = req.WithContext(tt.args.ctx)
			req = mux.SetURLVars(req, map[string]string{"cluster_external_id": tt.args.externalID})
			h.ValidateTokenAndExternalClusterID(rw, req)
			resp := rw.Result()
			defer resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}

}
