package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/api"
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/api/presenters"
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/errors"
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/services"
)

var _ RestHandler = dinosaurHandler{}

type dinosaurHandler struct {
	service services.DinosaurService
}

func NewDinosaurHandler(service services.DinosaurService) *dinosaurHandler {
	return &dinosaurHandler{
		service: service,
	}
}

func (h dinosaurHandler) Create(w http.ResponseWriter, r *http.Request) {
	var dinosaur openapi.Dinosaur
	cfg := &handlerConfig{
		&dinosaur,
		[]validate{
			validateEmpty(&dinosaur.Id, "id"),
			validateNotEmpty(&dinosaur.Species, "species"),
		},
		func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			dino := presenters.ConvertDinosaur(dinosaur)
			dino, err := h.service.Create(ctx, dino)
			if err != nil {
				return nil, err
			}
			return presenters.PresentDinosaur(dino), nil
		},
		handleError,
	}

	handle(w, r, cfg, http.StatusCreated)
}

func (h dinosaurHandler) Patch(w http.ResponseWriter, r *http.Request) {
	var patch api.DinosaurPatchRequest

	cfg := &handlerConfig{
		&patch,
		[]validate{
			validateNotEmpty(patch.Species, "species"),
		},
		func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()
			id := mux.Vars(r)["id"]
			found, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			dino, err := h.service.Replace(ctx, found)
			if err != nil {
				return nil, err
			}
			return presenters.PresentDinosaur(dino), nil
		},
		handleError,
	}

	handle(w, r, cfg, http.StatusOK)
}

func (h dinosaurHandler) List(w http.ResponseWriter, r *http.Request) {
	cfg := &handlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			ctx := r.Context()

			listArgs := services.NewListArguments(r.URL.Query())
			dinosaurs, paging, err := h.service.List(ctx, listArgs)
			if err != nil {
				return nil, err
			}
			dinoList := openapi.DinosaurList{
				Kind:  "DinosaurList",
				Page:  int32(paging.Page),
				Size:  int32(paging.Size),
				Total: int32(paging.Total),
				Items: []openapi.Dinosaur{},
			}

			for _, dino := range dinosaurs {
				converted := presenters.PresentDinosaur(dino)
				dinoList.Items = append(dinoList.Items, converted)
			}
			return dinoList, nil
		},
	}

	handleList(w, r, cfg)
}

func (h dinosaurHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg := &handlerConfig{
		Action: func() (interface{}, *errors.ServiceError) {
			id := mux.Vars(r)["id"]
			ctx := r.Context()
			dinosaur, err := h.service.Get(ctx, id)
			if err != nil {
				return nil, err
			}

			return presenters.PresentDinosaur(dinosaur), nil
		},
	}

	handleGet(w, r, cfg)
}

func (h dinosaurHandler) Delete(w http.ResponseWriter, r *http.Request) {
	handleError(r.Context(), w, errors.NotImplemented("delete"))
}
