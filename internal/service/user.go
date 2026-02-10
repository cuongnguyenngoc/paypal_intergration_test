package service

import (
	"context"
	"paypal-integration-demo/internal/model"
	"paypal-integration-demo/internal/repository"
)

type UserService interface {
	GetInventory(ctx context.Context, userID string) ([]*model.UserInventory, error)
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

func (s *userServiceImpl) GetInventory(ctx context.Context, userID string) ([]*model.UserInventory, error) {
	return s.inventoryRepo.Get(ctx, userID)
}
