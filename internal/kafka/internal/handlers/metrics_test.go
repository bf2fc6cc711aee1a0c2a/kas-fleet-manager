package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/client/observatorium"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/gorilla/mux"
	"github.com/onsi/gomega"
)

func TestNewMetricsHandler(t *testing.T) {
	type args struct {
		service services.ObservatoriumService
	}
	tests := []struct {
		name string
		args args
		want *metricsHandler
	}{
		{
			name: "should return the New Metrics Handler",
			args: args{
				service: &services.ObservatoriumServiceMock{},
			},
			want: &metricsHandler{
				service: &services.ObservatoriumServiceMock{},
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			g.Expect(NewMetricsHandler(tt.args.service)).To(gomega.Equal(tt.want))
		})
	}
}

func Test_metricsHandler_FederateMetrics(t *testing.T) {
	type fields struct {
		service services.ObservatoriumService
	}
	type args struct {
		url string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should return status code 200 if the metrics are federated",
			fields: fields{
				service: &services.ObservatoriumServiceMock{
					GetMetricsByKafkaIdFunc: func(ctx context.Context, csMetrics *observatorium.KafkaMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError) {
						return "", nil
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/kafkas/{id}/metrics/federate",
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should return status code 400 if the id is missing from the url",
			fields: fields{
				service: &services.ObservatoriumServiceMock{
					GetMetricsByKafkaIdFunc: func(ctx context.Context, csMetrics *observatorium.KafkaMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError) {
						return "", nil
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/kafkas/{id}/metrics/federate",
			},
			wantStatusCode: http.StatusBadRequest,
		},
		{
			name: "should return status code 500 if it fails to get metrics by Kafka Id",
			fields: fields{
				service: &services.ObservatoriumServiceMock{
					GetMetricsByKafkaIdFunc: func(ctx context.Context, csMetrics *observatorium.KafkaMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError) {
						return "", errors.GeneralError("Failed to get metrics by kafka Id")
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/kafkas/{id}/metrics/federate",
			},
			wantStatusCode: http.StatusInternalServerError,
		},
		{
			name: "should return status code 404 if the Kafka is not found",
			fields: fields{
				service: &services.ObservatoriumServiceMock{
					GetMetricsByKafkaIdFunc: func(ctx context.Context, csMetrics *observatorium.KafkaMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError) {
						return "", errors.New(errors.ErrorNotFound, "Failed to get metrics by kafka Id")
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/kafkas/{id}/metrics/federate",
			},
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			req, rw := GetHandlerParams("GET", tt.args.url, nil, t)

			if tt.wantStatusCode != http.StatusBadRequest {
				req = mux.SetURLVars(req, map[string]string{"id": "id"})
			}

			h := NewMetricsHandler(tt.fields.service)
			h.FederateMetrics(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_metricsHandler_GetMetricsByRangeQuery(t *testing.T) {
	type fields struct {
		service services.ObservatoriumService
	}
	type args struct {
		url string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should return status code 200 if it retrieves the metrics",
			fields: fields{
				service: &services.ObservatoriumServiceMock{
					GetMetricsByKafkaIdFunc: func(ctx context.Context, csMetrics *observatorium.KafkaMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError) {
						return "", nil
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/kafkas/{id}/metrics/query_range",
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should return status code 404 if the kafka is not found",
			fields: fields{
				service: &services.ObservatoriumServiceMock{
					GetMetricsByKafkaIdFunc: func(ctx context.Context, csMetrics *observatorium.KafkaMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError) {
						return "", errors.New(errors.ErrorNotFound, "Failed to get metrics by kafka Id")
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/kafkas/{id}/metrics/query_range",
			},
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			req, rw := GetHandlerParams("GET", tt.args.url, nil, t)
			req = mux.SetURLVars(req, map[string]string{"id": "id"})

			parameters := req.URL.Query()
			parameters.Add("duration", "30")
			parameters.Add("interval", "5")
			req.URL.RawQuery = parameters.Encode()

			h := NewMetricsHandler(tt.fields.service)
			h.GetMetricsByRangeQuery(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}

func Test_metricsHandler_GetMetricsByInstantQuery(t *testing.T) {
	type fields struct {
		service services.ObservatoriumService
	}
	type args struct {
		url string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantStatusCode int
	}{
		{
			name: "should return status code 200 if it retrieves the metrics",
			fields: fields{
				service: &services.ObservatoriumServiceMock{
					GetMetricsByKafkaIdFunc: func(ctx context.Context, csMetrics *observatorium.KafkaMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError) {
						return "", nil
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/kafkas/{id}/metrics/query",
			},
			wantStatusCode: http.StatusOK,
		},
		{
			name: "should return status code 404 if the kafka is not found",
			fields: fields{
				service: &services.ObservatoriumServiceMock{
					GetMetricsByKafkaIdFunc: func(ctx context.Context, csMetrics *observatorium.KafkaMetrics, id string, query observatorium.MetricsReqParams) (string, *errors.ServiceError) {
						return "", errors.New(errors.ErrorNotFound, "Failed to get metrics by kafka Id")
					},
				},
			},
			args: args{
				url: "/api/kafkas_mgmt/v1/kafkas/{id}/metrics/query_range",
			},
			wantStatusCode: http.StatusNotFound,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			req, rw := GetHandlerParams("GET", tt.args.url, nil, t)
			req = mux.SetURLVars(req, map[string]string{"id": "id"})

			h := NewMetricsHandler(tt.fields.service)
			h.GetMetricsByInstantQuery(rw, req)
			resp := rw.Result()
			resp.Body.Close()
			g.Expect(resp.StatusCode).To(gomega.Equal(tt.wantStatusCode))
		})
	}
}
