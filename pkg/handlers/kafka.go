package handlers

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"net/http"
)

//var _ RestHandler = &kafkaHandler{}

type kafkaHandler struct {
	service services.KafkaService
}

func NewKafkaHandler(service services.KafkaService) *kafkaHandler {
	return &kafkaHandler{
		service: service,
	}
}

func (h kafkaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var kafka openapi.Kafka
	cfg := &handlerConfig{
		MarshalInto: &kafka,
		Validate: []validate{
			validateEmpty(&kafka.Id, "id"),
			//validateNotEmpty(&kafka.ClusterID, "cluster_id"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			convKafka := presenters.ConvertKafka(kafka)
			err := h.service.Create(ctx, convKafka)
			if err != nil {
				return nil, err
			}
			return presenters.PresentKafka(convKafka), nil
		},
		ErrorHandler: handleError,
	}

	// return 202 status accepted
	handle(w, r, cfg, http.StatusAccepted)
}
