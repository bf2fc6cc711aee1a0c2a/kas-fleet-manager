package kafka_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type KafkaRoutesCNAMEManager struct {
	workers.BaseWorker
	kafkaService services.KafkaService
	kafkaConfig  *config.KafkaConfig
}

var _ workers.Worker = &KafkaRoutesCNAMEManager{}

func NewKafkaCNAMEManager(kafkaService services.KafkaService, kafkaConfig *config.KafkaConfig, reconciler workers.Reconciler) *KafkaRoutesCNAMEManager {
	return &KafkaRoutesCNAMEManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "kafka_dns",
			Reconciler: reconciler,
		},
		kafkaService: kafkaService,
		kafkaConfig:  kafkaConfig,
	}
}

func (k *KafkaRoutesCNAMEManager) Start() {
	k.StartWorker(k)
}

func (k *KafkaRoutesCNAMEManager) Stop() {
	k.StopWorker(k)
}

func (k *KafkaRoutesCNAMEManager) Reconcile() []error {
	glog.Infoln("reconciling DNS for kafkas")
	var errs []error

	kafkas, listErr := k.kafkaService.ListKafkasWithRoutesNotCreated()
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list kafkas whose routes are not created"))
	} else {
		glog.Infof("kafkas need routes created count = %d", len(kafkas))
	}

	for _, kafka := range kafkas {
		if k.kafkaConfig.EnableKafkaCNAMERegistration {
			if kafka.RoutesCreationId == "" {
				glog.Infof("creating CNAME records for kafka %s", kafka.ID)

				changeOutput, err := k.kafkaService.ChangeKafkaCNAMErecords(kafka, services.KafkaRoutesActionCreate)

				if err != nil {
					errs = append(errs, err)
					continue
				}

				kafka.RoutesCreationId = *changeOutput.ChangeInfo.Id
				kafka.RoutesCreated = *changeOutput.ChangeInfo.Status == "INSYNC"
			} else {
				recordStatus, err := k.kafkaService.GetCNAMERecordStatus(kafka)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				kafka.RoutesCreated = *recordStatus.Status == "INSYNC"
			}
		} else {
			glog.Infof("external certificate is disabled, skip CNAME creation for Kafka %s", kafka.ID)
			kafka.RoutesCreated = true
		}

		if err := k.kafkaService.Update(kafka); err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return errs
}
