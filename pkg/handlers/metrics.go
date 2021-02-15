package handlers

import (
	"github.com/gorilla/mux"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/client/observatorium"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"net/http"
	"strconv"
	"time"
)

type metricsHandler struct {
	service services.ObservatoriumService
}

func NewMetricsHandler(service services.ObservatoriumService) *metricsHandler {
	return &metricsHandler{
		service: service,
	}
}

func (h metricsHandler) GetMetricsByQueryRange(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	params := observatorium.MetricsReqParams{}
	query := r.URL.Query()
	cfg := &handlerConfig{
		Validate: []validate{
			validatQueryParam(query, "duration"),
			validatQueryParam(query, "interval"),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			ctx := r.Context()
			params.ResultType = observatorium.RangeQuery
			extractMetricsQueryParams(r, &params)
			kafkaMetrics := &observatorium.KafkaMetrics{}
			foundKafkaId, err := h.service.GetMetricsByKafkaId(ctx, kafkaMetrics, id, params)
			if err != nil {
				return nil, err
			}
			metricList := openapi.MetricsQueryRangeList{
				Kind: "Metrics",
				Id:   foundKafkaId,
			}
			metrics, err := presenters.PresentMetricsByQueryRange(kafkaMetrics)
			if err != nil {
				return nil, err
			}
			metricList.Items = metrics

			return metricList, nil
		},
		ErrorHandler: handleError,
	}
	handleGet(w, r, cfg)
}

func (h metricsHandler)  GetMetricsByQueryInstant(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	params := observatorium.MetricsReqParams{}
	cfg := &handlerConfig{
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			ctx := r.Context()
			params.ResultType = observatorium.Query
			extractMetricsQueryParams(r, &params)
			kafkaMetrics := &observatorium.KafkaMetrics{}
			foundKafkaId, err := h.service.GetMetricsByKafkaId(ctx, kafkaMetrics, id, params)
			if err != nil {
				return nil, err
			}
			metricList := openapi.MetricsQueryInstantList{
				Kind: "Metrics",
				Id:   foundKafkaId,
			}
			metrics, err := presenters.PresentMetricsByQueryInstant(kafkaMetrics)
			if err != nil {
				return nil, err
			}
			metricList.Items = metrics

			return metricList, nil
		},
		ErrorHandler: handleError,
	}
	handleGet(w, r, cfg)
}

func extractMetricsQueryParams(r *http.Request, q *observatorium.MetricsReqParams) {
	q.FillDefaults()
	queryParams := r.URL.Query()
	if dur := queryParams.Get("duration"); dur != "" {
		if num, err := strconv.ParseInt(dur, 10, 64); err == nil {
			duration := time.Duration(num) * time.Minute
			q.Start = q.End.Add(-duration)
		}
	}
	if step := queryParams.Get("interval"); step != "" {
		if num, err := strconv.Atoi(step); err == nil {
			q.Step = time.Duration(num) * time.Second
		}
	}
	if filters, ok := queryParams["filters"]; ok && len(filters) > 0 {
		q.Filters = filters
	}

}
