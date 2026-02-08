package service

import (
	"context"
	"paypal-integration-demo/internal/model"
	"paypal-integration-demo/internal/repository"
)

type UserService interface {
	GetInventories(ctx context.Context) ([]*model.UserInventory, error)
}

type userServiceImpl struct {
	inventoryRepo repository.InventoryRepository
}

func NewUserService(
	inventoryRepo repository.InventoryRepository,
) UserService {
	return &userServiceImpl{
		inventoryRepo: inventoryRepo,
	}
}

func (s *userServiceImpl) GetInventories(ctx context.Context) ([]*model.UserInventory, error) {
	return s.inventoryRepo.Get(ctx)
}
