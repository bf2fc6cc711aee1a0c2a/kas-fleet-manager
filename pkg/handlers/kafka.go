package handlers

import (
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"net/http"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
)

//var _ RestHandler = &kafkaHandler{}

type kafkaHandler struct {
	service services.KafkaService
	clusterService services.ClusterService
}

func NewKafkaHandler(service services.KafkaService, clusterService services.ClusterService) *kafkaHandler {
	return &kafkaHandler{
		service: service,
		clusterService: clusterService,
	}
}

func (h kafkaHandler) Create(w http.ResponseWriter, r *http.Request) {
	var kafka openapi.Kafka
	cfg := &handlerConfig{
		MarshalInto: &kafka,
		Validate: []validate{
			validateEmpty(&kafka.Id, "id"),
			validateNotEmpty(&kafka.Region, "region"),
			validateNotEmpty(&kafka.CloudProvider, "cloud_provider"),
			validateNotEmpty(&kafka.Name, "cluster_name"),
		},
		Action: func() (interface{}, *errors.ServiceError) {
			convKafka := presenters.ConvertKafka(kafka)
			err := h.service.RegisterKafkaJob(convKafka)
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

// TODO: Temporary endpoint for verification only!!!!
// Jira: https://issues.redhat.com/browse/MGDSTRM-22
func (h kafkaHandler) ClusterCreate(w http.ResponseWriter, r *http.Request) {
	cluster := api.Cluster{
		CloudProvider: "aws",
		Region:        "eu-west-1",
	}
	cfg := &handlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			if _, err := h.clusterService.Create(&cluster); err != nil {
				return nil, err
			}
			return nil, nil
		},
		ErrorHandler: handleError,
	}

	// return 201 status created
	handle(w, r, cfg, http.StatusCreated)
}