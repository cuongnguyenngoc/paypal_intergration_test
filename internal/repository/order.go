package repository

import (
	"context"
	"paypal-integration-demo/internal/model"
	"time"

	"gorm.io/gorm"
)

type OrderRepository interface {
	Create(ctx context.Context, tx *gorm.DB, order *model.Order) error
	FindByOrderID(ctx context.Context, orderID string) (*model.Order, error)
	MarkCompleted(ctx context.Context, tx *gorm.DB, orderID string) error
	MarkPaid(ctx context.Context, tx *gorm.DB, orderID string) (*model.Order, error)
	IsPaid(ctx context.Context, orderID string) (bool, error)
	CreateOrderItems(ctx context.Context, tx *gorm.DB, items []*model.OrderItem) error
	GetOrderItems(ctx context.Context, tx *gorm.DB, orderID string) ([]*model.OrderItem, error)
}

type orderRepoImpl struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepoImpl{
		db: db,
	}
}

func (r *orderRepoImpl) Create(ctx context.Context, tx *gorm.DB, order *model.Order) error {
	return tx.WithContext(ctx).Create(order).Error
}

func (r *orderRepoImpl) FindByOrderID(ctx context.Context, orderID string) (*model.Order, error) {
	var order model.Order
	err := r.db.WithContext(ctx).
		Where("order_id = ?", orderID).
		First(&order).Error

	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *orderRepoImpl) MarkCompleted(ctx context.Context, tx *gorm.DB, orderID string) error {
	return tx.WithContext(ctx).Model(&model.Order{}).
		Where(`
			order_id = ?
			AND status IN ?
		`,
			orderID,
			[]string{"CREATED", "APPROVED"},
		).
		Updates(map[string]interface{}{
			"status":     "COMPLETED",
			"updated_at": time.Now(),
		}).Error
}

func (r *orderRepoImpl) MarkPaid(ctx context.Context, tx *gorm.DB, orderID string) (*model.Order, error) {
	var order model.Order
	err := tx.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Update the record
		result := tx.Model(&order).
			Where("order_id = ?", orderID).
			Updates(map[string]interface{}{
				"status":     "PAID",
				"updated_at": time.Now(),
			})

		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		// Fetch the updated record within the same transaction
		return tx.Where("order_id = ?", orderID).First(&order).Error
	})

	return &order, err
}

func (r *orderRepoImpl) IsPaid(ctx context.Context, orderID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.Order{}).
		Where("order_id = ?", orderID).
		Where("status = ?", "PAID").
		Count(&count).Error

	return count > 0, err
}

func (r *orderRepoImpl) CreateOrderItems(ctx context.Context, tx *gorm.DB, items []*model.OrderItem) error {
	return tx.WithContext(ctx).Create(&items).Error
}

func (r *orderRepoImpl) GetOrderItems(ctx context.Context, tx *gorm.DB, orderID string) ([]*model.OrderItem, error) {
	var items []*model.OrderItem
	err := tx.WithContext(ctx).Where("order_id = ?", orderID).
		Find(&items).Error

	if err != nil {
		return nil, err
	}

	return items, nil
}
