package services

import (
	"context"

	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/api"
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/db"
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/errors"
	"gitlab.cee.redhat.com/service/sdb-ocm-example-service/pkg/logger"
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
		connectionFactory: connectionFactory,
	}
}

var _ DinosaurService = &sqlDinosaurService{}

type sqlDinosaurService struct {
	connectionFactory *db.ConnectionFactory
}

func (s *sqlDinosaurService) Get(ctx context.Context, id string) (*api.Dinosaur, *errors.ServiceError) {
	gorm := s.connectionFactory.New()

	var dinosaur api.Dinosaur
	if err := gorm.First(&dinosaur, "id =? ", id).Error; err != nil {
		return nil, handleGetError("Dinosaur", "id", id, err)
	}
	return &dinosaur, nil
}

func (s *sqlDinosaurService) List(ctx context.Context, listArgs *ListArguments) (api.DinosaurList, *api.PagingMeta, *errors.ServiceError) {
	gorm := s.connectionFactory.New()
	ulog := logger.NewUHCLogger(ctx)
	pagingMeta := api.PagingMeta{
		Page: listArgs.Page,
	}

	// Unbounded list operations should be discouraged, as they can result in very long API operations
	if listArgs.Size < 0 {
		ulog.Warningf("A query with an unbound size was requested.")
	}

	// Get the total number of records
	gorm.Model(api.Dinosaur{}).Count(&pagingMeta.Total)

	// Set the order by arguments
	for _, orderByArg := range listArgs.OrderBy {
		gorm = gorm.Order(orderByArg)
	}

	// TODO Search

	// Get the full list, using page/size to limit the result set
	var dinosaurs api.DinosaurList
	if err := gorm.Offset((listArgs.Page - 1) * listArgs.Size).Limit(listArgs.Size).Find(&dinosaurs).Error; err != nil {
		return dinosaurs, &pagingMeta, errors.GeneralError("Unable to list dinosaurs: %s", err)
	}

	// Set the proper size, as the result set total may be less than the requested size
	pagingMeta.Size = len(dinosaurs)

	return dinosaurs, &pagingMeta, nil
}

func (s *sqlDinosaurService) Create(ctx context.Context, dinosaur *api.Dinosaur) (*api.Dinosaur, *errors.ServiceError) {
	gorm := s.connectionFactory.New()

	if err := gorm.Create(dinosaur).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, handleCreateError("Dinosaur", err)
	}
	return dinosaur, nil
}

func (s *sqlDinosaurService) Replace(ctx context.Context, dinosaur *api.Dinosaur) (*api.Dinosaur, *errors.ServiceError) {
	gorm := s.connectionFactory.New()

	if err := gorm.Save(dinosaur).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return nil, handleUpdateError("Dinosaur", err)
	}
	return dinosaur, nil
}

func (s *sqlDinosaurService) Delete(ctx context.Context, id string) *errors.ServiceError {
	gorm := s.connectionFactory.New()

	if err := gorm.Delete(&api.Dinosaur{Meta: api.Meta{ID: id}}).Error; err != nil {
		db.MarkForRollback(ctx, err)
		return errors.GeneralError("Unable to delete dinosaur: %s", err)
	}
	return nil
}

func (s *sqlDinosaurService) FindByIDs(ctx context.Context, ids []string) (api.DinosaurList, *errors.ServiceError) {
	gorm := s.connectionFactory.New()

	results := api.DinosaurList{}
	if err := gorm.Where("id in (?)", ids).Find(&results).Error; err != nil {
		return nil, errors.GeneralError("Unable to get all dinosaurs: %s", err)
	}

	return results, nil
}

func (s *sqlDinosaurService) FindBySpecies(ctx context.Context, species string) (api.DinosaurList, *errors.ServiceError) {
	gorm := s.connectionFactory.New()

	results := api.DinosaurList{}
	if err := gorm.Where("species = ?", species).Find(&results).Error; err != nil {
		return nil, handleGetError("Dinosaur", "species", species, err)
	}
	return results, nil
}

func (s *sqlDinosaurService) All(ctx context.Context) (api.DinosaurList, *errors.ServiceError) {
	gorm := s.connectionFactory.New()

	results := api.DinosaurList{}
	if err := gorm.Find(&results).Error; err != nil {
		return nil, errors.GeneralError("Unable to get all dinosaurs: %s", err)
	}

	return results, nil
}
