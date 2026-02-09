package repository

import (
	"context"
	"paypal-integration-demo/internal/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InventoryRepository interface {
	Upsert(ctx context.Context, tx *gorm.DB, inventory *model.UserInventory) error
	Get(ctx context.Context) ([]*model.UserInventory, error)
}

type inventoryRepoImpl struct {
	db *gorm.DB
}

func NewInventoryRepository(db *gorm.DB) InventoryRepository {
	return &inventoryRepoImpl{
		db: db,
	}
}

func (r *inventoryRepoImpl) Upsert(ctx context.Context, tx *gorm.DB, inventory *model.UserInventory) error {
	return tx.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "product_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"quantity":   gorm.Expr("user_inventories.quantity + ?", inventory.Quantity),
			"updated_at": time.Now(),
		}),
	}).Create(&inventory).Error
}

func (r *inventoryRepoImpl) Get(ctx context.Context) ([]*model.UserInventory, error) {
	var inventories []*model.UserInventory

	err := r.db.WithContext(ctx).Find(&inventories).Error
	if err != nil {
		return nil, err
	}

	return inventories, nil
}
