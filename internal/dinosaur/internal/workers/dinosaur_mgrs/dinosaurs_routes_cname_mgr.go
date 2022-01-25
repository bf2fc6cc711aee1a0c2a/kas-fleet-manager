package dinosaur_mgrs

import (
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/pkg/workers"
	"github.com/golang/glog"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type DinosaurRoutesCNAMEManager struct {
	workers.BaseWorker
	dinosaurService services.DinosaurService
	dinosaurConfig  *config.DinosaurConfig
}

var _ workers.Worker = &DinosaurRoutesCNAMEManager{}

func NewDinosaurCNAMEManager(dinosaurService services.DinosaurService, kafkfConfig *config.DinosaurConfig) *DinosaurRoutesCNAMEManager {
	return &DinosaurRoutesCNAMEManager{
		BaseWorker: workers.BaseWorker{
			Id:         uuid.New().String(),
			WorkerType: "dinosaur_dns",
			Reconciler: workers.Reconciler{},
		},
		dinosaurService: dinosaurService,
		dinosaurConfig:  kafkfConfig,
	}
}

func (k *DinosaurRoutesCNAMEManager) Start() {
	k.StartWorker(k)
}

func (k *DinosaurRoutesCNAMEManager) Stop() {
	k.StopWorker(k)
}

func (k *DinosaurRoutesCNAMEManager) Reconcile() []error {
	glog.Infoln("reconciling DNS for dinosaurs")
	var errs []error

	dinosaurs, listErr := k.dinosaurService.ListDinosaursWithRoutesNotCreated()
	if listErr != nil {
		errs = append(errs, errors.Wrap(listErr, "failed to list dinosaurs whose routes are not created"))
	} else {
		glog.Infof("dinosaurs need routes created count = %d", len(dinosaurs))
	}

	for _, dinosaur := range dinosaurs {
		if k.dinosaurConfig.EnableDinosaurExternalCertificate {
			if dinosaur.RoutesCreationId == "" {
				glog.Infof("creating CNAME records for dinosaur %s", dinosaur.ID)

				changeOutput, err := k.dinosaurService.ChangeDinosaurCNAMErecords(dinosaur, services.DinosaurRoutesActionCreate)

				if err != nil {
					errs = append(errs, err)
					continue
				}

				dinosaur.RoutesCreationId = *changeOutput.ChangeInfo.Id
				dinosaur.RoutesCreated = *changeOutput.ChangeInfo.Status == "INSYNC"
			} else {
				recordStatus, err := k.dinosaurService.GetCNAMERecordStatus(dinosaur)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				dinosaur.RoutesCreated = *recordStatus.Status == "INSYNC"
			}
		} else {
			glog.Infof("external certificate is disabled, skip CNAME creation for Dinosaur %s", dinosaur.ID)
			dinosaur.RoutesCreated = true
		}

		if err := k.dinosaurService.Update(dinosaur); err != nil {
			errs = append(errs, err)
			continue
		}
	}

	return errs
}
