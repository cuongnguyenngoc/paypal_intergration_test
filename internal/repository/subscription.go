package repository

import (
	"context"
	"paypal-integration-demo/internal/model"
	"time"

	"gorm.io/gorm"
)

type SubscriptionRepository interface {
	GetPaypalPlanByProductCode(ctx context.Context, productCode string) (*model.PayPalPlan, error)

	CreateSubscription(ctx context.Context, sub *model.UserSubscription) error
	ActivateSubscription(ctx context.Context, subscriptionID string, start time.Time, next *time.Time) error
	CancelSubscription(ctx context.Context, subscriptionID string) error
	GetBySubscriptionID(ctx context.Context, subscriptionID string) (*model.UserSubscription, error)
}

type subscriptionRepoImpl struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &subscriptionRepoImpl{
		db: db,
	}
}

func (r *subscriptionRepoImpl) GetPaypalPlanByProductCode(ctx context.Context, productCode string) (*model.PayPalPlan, error) {
	var plan model.PayPalPlan
	err := r.db.WithContext(ctx).
		Where("product_code = ?", productCode).
		First(&plan).
		Error

	if err != nil {
		return nil, err
	}

	return &plan, nil
}

func (r *subscriptionRepoImpl) CreateSubscription(ctx context.Context, sub *model.UserSubscription) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *subscriptionRepoImpl) ActivateSubscription(ctx context.Context, subscriptionID string, start time.Time, next *time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.UserSubscription{}).
		Where("paypal_subscription_id = ?", subscriptionID).
		Updates(map[string]interface{}{
			"status":            "ACTIVE",
			"start_time":        start,
			"next_billing_time": next,
		}).Error
}

func (r *subscriptionRepoImpl) CancelSubscription(ctx context.Context, subscriptionID string) error {
	return r.db.WithContext(ctx).
		Model(&model.UserSubscription{}).
		Where("paypal_subscription_id = ?", subscriptionID).
		Update("status", "CANCELLED").
		Error
}

func (r *subscriptionRepoImpl) GetBySubscriptionID(ctx context.Context, subscriptionID string) (*model.UserSubscription, error) {
	var sub model.UserSubscription
	err := r.db.WithContext(ctx).
		Where("paypal_subscription_id = ?", subscriptionID).
		First(&sub).
		Error

	if err != nil {
		return nil, err
	}

	return &sub, nil
}
