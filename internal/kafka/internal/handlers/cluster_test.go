package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	mocks "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/auth"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/golang-jwt/jwt/v4"
	"github.com/onsi/gomega"
)

const (
	invalidParam                 = "invalid"
	validLengthClusterId         = "1234abcd1234abcd1234abcd1234abcd"
	validFormatExternalClusterId = "69d631de-9b7f-4bc2-bf4f-4d3295a7b25e"
	validDnsName                 = "apps.enterprise-aws.awdk.s1.devshift.org"
)

var (
	ctxWithClaims = auth.SetTokenInContext(context.TODO(), &jwt.Token{
		Claims: jwt.MapClaims{
			"username":     "test-user",
			"org_id":       mocks.DefaultOrganisationId,
			"is_org_admin": true,
		},
	})
)

func Test_RegisterEnterpriseCluster(t *testing.T) {
	type fields struct {
		kasFleetshardOperatorAddon services.KasFleetshardOperatorAddon
		clusterService             services.ClusterService
	}

	type args struct {
		body []byte
		ctx  context.Context
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
		want           *public.EnterpriseCluster
	}{
		{
			name: "should return an error if body is empty",
			args: args{
				body: []byte(`{}`),
				ctx:  context.TODO(),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return an error if cluster_id is invalid",
			args: args{
				body: []byte(fmt.Sprintf(`{"cluster_id": "%s"}`, invalidParam)),
				ctx:  context.TODO(),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return an error if cluster_external_id is empty",
			args: args{
				body: []byte(fmt.Sprintf(`{"cluster_id": "%s"}`, validLengthClusterId)),
				ctx:  context.TODO(),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return an error if cluster_external_id is invalid",
			args: args{
				body: []byte(fmt.Sprintf(`{"cluster_id": "%s", "cluster_external_id": "%s"}`, validLengthClusterId, invalidParam)),
				ctx:  context.TODO(),
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return an error if FindClusterByID returns error other than cluster not found",
			args: args{
				body: []byte(fmt.Sprintf(`{"cluster_id": "%s", "cluster_external_id": "%s"}`, validLengthClusterId, validFormatExternalClusterId)),
				ctx:  context.TODO(),
			},
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, errors.GeneralError("unexpected error")
					},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should return an error if cluster_ingress_dns_name is empty",
			args: args{
				body: []byte(fmt.Sprintf(`{"cluster_id": "%s", "cluster_external_id": "%s"}`, validLengthClusterId, validFormatExternalClusterId)),
				ctx:  context.TODO(),
			},
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return an error if cluster_dns_name is invalid",
			args: args{
				body: []byte(fmt.Sprintf(`{"cluster_id": "%s", "cluster_external_id": "%s", "cluster_ingress_dns_name": "%s"}`, validLengthClusterId, validFormatExternalClusterId, invalidParam)),
				ctx:  context.TODO(),
			},
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return an error if claims cant be obtained from context",
			args: args{
				body: []byte(fmt.Sprintf(`{"cluster_id": "%s", "cluster_external_id": "%s", "cluster_ingress_dns_name": "%s"}`, validLengthClusterId, validFormatExternalClusterId, validDnsName)),
				ctx:  context.TODO(),
			},
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, nil
					},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should return an error if GetAddonParams returns an error",
			args: args{
				body: []byte(fmt.Sprintf(`{"cluster_id": "%s", "cluster_external_id": "%s", "cluster_ingress_dns_name": "%s"}`, validLengthClusterId, validFormatExternalClusterId, validDnsName)),
				ctx:  ctxWithClaims,
			},
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					GetAddonParamsFunc: func(cluster *api.Cluster) (services.ParameterList, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to get addons")
					},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should return an error if RegisterClusterJob returns an error",
			args: args{
				body: []byte(fmt.Sprintf(`{"cluster_id": "%s", "cluster_external_id": "%s", "cluster_ingress_dns_name": "%s"}`, validLengthClusterId, validFormatExternalClusterId, validDnsName)),
				ctx:  ctxWithClaims,
			},
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, nil
					},
					RegisterClusterJobFunc: func(clusterRequest *api.Cluster) *errors.ServiceError {
						return errors.GeneralError("failed to register cluster")
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					GetAddonParamsFunc: func(cluster *api.Cluster) (services.ParameterList, *errors.ServiceError) {
						return services.ParameterList{
							{
								Id:    "some-id",
								Value: "value",
							},
						}, nil
					},
				},
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should successfully register enterprise cluster",
			args: args{
				body: []byte(fmt.Sprintf(`{"cluster_id": "%s", "cluster_external_id": "%s", "cluster_ingress_dns_name": "%s"}`, validLengthClusterId, validFormatExternalClusterId, validDnsName)),
				ctx:  ctxWithClaims,
			},
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, nil
					},
					RegisterClusterJobFunc: func(clusterRequest *api.Cluster) *errors.ServiceError {
						return nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					GetAddonParamsFunc: func(cluster *api.Cluster) (services.ParameterList, *errors.ServiceError) {
						return services.ParameterList{
							{
								Id:    "some-id",
								Value: "value",
							},
						}, nil
					},
				},
			},
			wantStatusCode: http.StatusOK,
			want: &public.EnterpriseCluster{
				Status:    api.ClusterAccepted.String(),
				ClusterId: validLengthClusterId,
				FleetshardParameters: []public.FleetshardParameter{
					{
						Id:    "some-id",
						Value: "value",
					},
				},
			},
		},
		{
			name: "should successfully register enterprise cluster if FindClusterByID returns cluster not found error",
			args: args{
				body: []byte(fmt.Sprintf(`{"cluster_id": "%s", "cluster_external_id": "%s", "cluster_ingress_dns_name": "%s"}`, validLengthClusterId, validFormatExternalClusterId, validDnsName)),
				ctx:  ctxWithClaims,
			},
			fields: fields{
				clusterService: &services.ClusterServiceMock{
					FindClusterByIDFunc: func(clusterID string) (*api.Cluster, *errors.ServiceError) {
						return nil, errors.GeneralError("failed to find cluster")
					},
					RegisterClusterJobFunc: func(clusterRequest *api.Cluster) *errors.ServiceError {
						return nil
					},
				},
				kasFleetshardOperatorAddon: &services.KasFleetshardOperatorAddonMock{
					GetAddonParamsFunc: func(cluster *api.Cluster) (services.ParameterList, *errors.ServiceError) {
						return services.ParameterList{
							{
								Id:    "some-id",
								Value: "value",
							},
						}, nil
					},
				},
			},
			wantStatusCode: http.StatusOK,
			want: &public.EnterpriseCluster{
				Status:    api.ClusterAccepted.String(),
				ClusterId: validLengthClusterId,
				FleetshardParameters: []public.FleetshardParameter{
					{
						Id:    "some-id",
						Value: "value",
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			h := NewClusterHandler(tt.fields.kasFleetshardOperatorAddon, tt.fields.clusterService)
			req, rw := GetHandlerParams("POST", "", bytes.NewBuffer(tt.args.body), t)
			req = req.WithContext(tt.args.ctx)
			h.RegisterEnterpriseCluster(rw, req)
			resp := rw.Result()
			defer resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
			if tt.wantStatusCode == http.StatusOK {
				cluster := &public.EnterpriseCluster{}
				err := json.NewDecoder(resp.Body).Decode(&cluster)
				g.Expect(err).NotTo(gomega.HaveOccurred())
				g.Expect(cluster).To(gomega.Equal(tt.want))
			}
		})
	}
}
