package repository

import (
	"paypal-integration-demo/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type OrderRepository interface {
	CreateOrUpdate(order *model.Order) error
	FindByOrderID(orderID string) (*model.Order, error)
	MarkStatus(orderID string, status string) error
	IsPaid(orderID string) (bool, error)
	CreateOrderItems(items []*model.OrderItem) error
}

type orderRepoImpl struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepoImpl{
		db: db,
	}
}

func (r *orderRepoImpl) CreateOrUpdate(order *model.Order) error {
	return r.db.Clauses(
		clause.OnConflict{
			Columns: []clause.Column{
				{Name: "order_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"status",
				"payer_id",
				"amount",
				"currency",
				"updated_at",
			}),
		},
	).Create(order).Error
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

func (r *orderRepoImpl) MarkStatus(orderID string, status string) error {
	return r.db.Model(&model.Order{}).
		Where("order_id = ?", orderID).
		Update("status", status).Error
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
