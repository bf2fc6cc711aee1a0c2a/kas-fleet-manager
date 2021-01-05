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

type metricrHandler struct {
	service services.ObservatoriumService
}

var _ RestHandler = metricrHandler{}

func NewMetricsHandler(service services.ObservatoriumService) *metricrHandler {
	return &metricrHandler{
		service: service,
	}
}

func (h metricrHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]
	params := observatorium.RangeQuery{}
	query := r.URL.Query()
	cfg := &handlerConfig{
		Validate: []validate{
			validatQueryParam(query, "duration"),
			validatQueryParam(query, "interval"),
		},
		Action: func() (i interface{}, serviceError *errors.ServiceError) {
			ctx := r.Context()
			extractMetricsQueryParams(r, &params)
			kafkaMetrics := &observatorium.KafkaMetrics{}
			foundKafkaId, err := h.service.GetMetricsByKafkaId(ctx, kafkaMetrics, id, params)
			if err != nil {
				return nil, err
			}
			metricList := openapi.MetricsList{
				Kind: "Metrics",
				Id:   foundKafkaId,
			}
			metrics, err := presenters.PresentMetrics(kafkaMetrics)
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
func (h metricrHandler) List(w http.ResponseWriter, r *http.Request) {
	handleError(r.Context(), w, errors.NotImplemented("list"))
}

func (h metricrHandler) Create(w http.ResponseWriter, r *http.Request) {
	handleError(r.Context(), w, errors.NotImplemented("create"))
}

func (h metricrHandler) Patch(w http.ResponseWriter, r *http.Request) {
	handleError(r.Context(), w, errors.NotImplemented("patch"))
}

func (h metricrHandler) Delete(w http.ResponseWriter, r *http.Request) {
	handleError(r.Context(), w, errors.NotImplemented("delete"))
}
func extractMetricsQueryParams(r *http.Request, q *observatorium.RangeQuery) {
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
