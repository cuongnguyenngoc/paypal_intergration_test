package repository

import (
	"paypal-integration-demo/internal/model"
	"time"

	"gorm.io/gorm"
)

type OrderRepository interface {
	Create(order *model.Order) error
	FindByOrderID(orderID string) (*model.Order, error)
	MarkCompleted(orderID string, payerID string) error
	MarkPaid(orderID string) (*model.Order, error)
	IsPaid(orderID string) (bool, error)
	CreateOrderItems(items []*model.OrderItem) error
	GetOrderItems(orderID string) ([]*model.OrderItem, error)
}

type orderRepoImpl struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepoImpl{
		db: db,
	}
}

func (r *orderRepoImpl) Create(order *model.Order) error {
	return r.db.Create(order).Error
}

func (r *orderRepoImpl) FindByOrderID(orderID string) (*model.Order, error) {
	var order model.Order
	err := r.db.
		Where("order_id = ?", orderID).
		First(&order).Error

	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (r *orderRepoImpl) MarkCompleted(orderID string, payerID string) error {
	return r.db.Model(&model.Order{}).
		Where(`
			order_id = ?
			AND status IN ?
			AND (payer_id = '' OR payer_id IS NULL)
		`,
			orderID,
			[]string{"CREATED", "APPROVED"},
		).
		Updates(map[string]interface{}{
			"status":     "COMPLETED",
			"payer_id":   payerID,
			"updated_at": time.Now(),
		}).Error
}

func (r *orderRepoImpl) MarkPaid(orderID string) (*model.Order, error) {
	var order model.Order
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// Update the record
		result := tx.Model(&order).
			Where("order_id = ? AND status = ?", orderID, "COMPLETED").
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

func (r *orderRepoImpl) IsPaid(orderID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Order{}).
		Where("order_id = ?", orderID).
		Where("status = ?", "PAID").
		Count(&count).Error

	return count > 0, err
}

func (r *orderRepoImpl) CreateOrderItems(items []*model.OrderItem) error {
	return r.db.Create(&items).Error
}

func (r *orderRepoImpl) GetOrderItems(orderID string) ([]*model.OrderItem, error) {
	var items []*model.OrderItem
	err := r.db.Where("order_id = ?", orderID).
		Find(&items).Error

	if err != nil {
		return nil, err
	}

	return items, nil
}
