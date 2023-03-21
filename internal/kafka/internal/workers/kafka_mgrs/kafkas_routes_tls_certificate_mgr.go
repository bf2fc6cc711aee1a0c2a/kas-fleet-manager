package kafka_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/logger"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// KafkasRoutesTLSCertificateManager represents a kafka manager that periodically reconciles kafka routes tls certificate.
type KafkasRoutesTLSCertificateManager struct {
	workers.BaseWorker
	kafkaService services.KafkaService
}

// NewKafkasRoutesTLSCertificateManager creates a new kafka manager to reconcile kafkas tls certificate.
func NewKafkasRoutesTLSCertificateManager(kafkaService services.KafkaService, reconciler workers.Reconciler) *KafkasRoutesTLSCertificateManager {
	return &KafkasRoutesTLSCertificateManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "kafka_routes_tls_certificate",
			Reconciler: reconciler,
		},
		kafkaService: kafkaService,
	}
}

// Start initializes the kafka manager to reconcile kafka routes tls certificate.
func (k *KafkasRoutesTLSCertificateManager) Start() {
	k.StartWorker(k)
}

// Stop causes the process for reconciling kafka routes tls certificate to stop.
func (k *KafkasRoutesTLSCertificateManager) Stop() {
	k.StopWorker(k)
}

func (k *KafkasRoutesTLSCertificateManager) Reconcile() []error {
	logger.Logger.Infof("reconciling kafka routes tls certificate")

	var encounteredErrors []error

	kafkas, err := k.kafkaService.ListAll()
	if err != nil {
		encounteredErrors = append(encounteredErrors, errors.Wrap(err, "failed to list kafkas"))
		return encounteredErrors
	}

	logger.Logger.Infof("kafkas count = %d", len(kafkas))

	for _, kafka := range kafkas {
		glog.V(10).Infof("reconciling tls certificate for kafka with id %q", kafka.ID)
		if err := k.kafkaService.ManagedKafkasRoutesTLSCertificate(kafka); err != nil {
			encounteredErrors = append(encounteredErrors, errors.Wrapf(err, "failed to reconcile kafka routes tls certificates%q", kafka.ID))
		}
	}

	return encounteredErrors
}
