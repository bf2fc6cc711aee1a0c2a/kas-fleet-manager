package services

import (
	"context"

	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/api"
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/db"
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/errors"
)

type DinosaurService interface {
	Get(ctx context.Context, id string) (*api.Dinosaur, *errors.ServiceError)
	List(ctx context.Context, listArgs *ListArguments) (api.DinosaurList, *api.PagingMeta, *errors.ServiceError)
	Create(ctx context.Context, dinosaur *api.Dinosaur) (*api.Dinosaur, *errors.ServiceError)
	Replace(ctx context.Context, dinosaur *api.Dinosaur) (*api.Dinosaur, *errors.ServiceError)
	Delete(ctx context.Context, id string) *errors.ServiceError
	All(ctx context.Context) (api.DinosaurList, *errors.ServiceError)

	FindBySpecies(ctx context.Context, species string) (api.DinosaurList, *errors.ServiceError)
	FindByIDs(ctx context.Context, ids []string) (api.DinosaurList, *errors.ServiceError)
}

func NewDinosaurService(connectionFactory *db.ConnectionFactory) DinosaurService {
	return &sqlDinosaurService{
		ConnectionFactory: connectionFactory,
	}
}

var _ DinosaurService = &sqlDinosaurService{}

type sqlDinosaurService struct {
	ConnectionFactory *db.ConnectionFactory
}

func (s *sqlDinosaurService) Get(ctx context.Context, id string) (*api.Dinosaur, *errors.ServiceError) {
	gorm := s.ConnectionFactory.New()

	var dinosaur api.Dinosaur
	if err := gorm.First(&dinosaur, "id =? ", id).Error; err != nil {
		return nil, handleGetError("Dinosaur", "id", id, err)
	}
	return &dinosaur, nil
}

func (s *sqlDinosaurService) List(ctx context.Context, listArgs *ListArguments) (api.DinosaurList, *api.PagingMeta, *errors.ServiceError) {
	// TODO
	return api.DinosaurList{}, &api.PagingMeta{}, errors.NotImplemented("")
}

func (s *sqlDinosaurService) Create(ctx context.Context, dinosaur *api.Dinosaur) (*api.Dinosaur, *errors.ServiceError) {
	gorm := s.ConnectionFactory.New()

	if err := gorm.Create(dinosaur).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, handleCreateError("Dinosaur", err)
	}
	return dinosaur, nil
}

func (s *sqlDinosaurService) Replace(ctx context.Context, dinosaur *api.Dinosaur) (*api.Dinosaur, *errors.ServiceError) {
	gorm := s.ConnectionFactory.New()

	if err := gorm.Save(dinosaur).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, handleUpdateError("Dinosaur", err)
	}
	return dinosaur, nil
}

func (s *sqlDinosaurService) Delete(ctx context.Context, id string) *errors.ServiceError {
	gorm := s.ConnectionFactory.New()

	if err := gorm.Delete(&api.Dinosaur{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return errors.GeneralError("Unable to delete dinosaur: %s", err)
	}
	return nil
}

func (s *sqlDinosaurService) FindByIDs(ctx context.Context, ids []string) (api.DinosaurList, *errors.ServiceError) {
	gorm := s.ConnectionFactory.New()

	results := api.DinosaurList{}
	if err := gorm.Where("id in (?)", ids).Find(&results).Error; err != nil {
		return nil, errors.GeneralError("Unable to get all dinosaurs: %s", err)
	}

	return results, nil
}

func (s *sqlDinosaurService) FindBySpecies(ctx context.Context, species string) (api.DinosaurList, *errors.ServiceError) {
	gorm := s.ConnectionFactory.New()

	results := api.DinosaurList{}
	if err := gorm.Where("species = ?", species).Find(&results).Error; err != nil {
		return nil, handleGetError("Dinosaur", "species", species, err)
	}
	return results, nil
}

func (s *sqlDinosaurService) All(ctx context.Context) (api.DinosaurList, *errors.ServiceError) {
	gorm := s.ConnectionFactory.New()

	results := api.DinosaurList{}
	if err := gorm.Find(&results).Error; err != nil {
		return nil, errors.GeneralError("Unable to get all dinosaurs: %s", err)
	}

	return results, nil
}
