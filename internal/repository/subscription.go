package repository

import (
	"context"
	"errors"
	"paypal-integration-demo/internal/model"
	"time"

	"gorm.io/gorm"
)

type SubscriptionRepository interface {
	GetSubPlanByProductID(ctx context.Context, merchantID string, productID string) (*model.SubscriptionPlan, error)
	StoreSubPlan(ctx context.Context, plan *model.SubscriptionPlan) error

	CreateSubscription(ctx context.Context, sub *model.UserSubscription) error
	ActivateSubscription(ctx context.Context, subscriptionID string, start time.Time, next *time.Time) error
	CancelSubscription(ctx context.Context, subscriptionID string) error
	GetBySubscriptionID(ctx context.Context, subscriptionID string) (*model.UserSubscription, error)
	GetActiveByUser(ctx context.Context, userID string, merchantID string) (*model.UserSubscription, error)
}

type subscriptionRepoImpl struct {
	db *gorm.DB
}

func NewSubscriptionRepository(db *gorm.DB) SubscriptionRepository {
	return &subscriptionRepoImpl{
		db: db,
	}
}

func (r *subscriptionRepoImpl) GetSubPlanByProductID(ctx context.Context, merchantID string, productID string) (*model.SubscriptionPlan, error) {
	var plan model.SubscriptionPlan
	err := r.db.WithContext(ctx).
		Where("merchant_id = ? AND product_id = ?", merchantID, productID).
		First(&plan).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &plan, gorm.ErrRecordNotFound
		}
		return nil, err
	}

	return &plan, nil
}

func (r *subscriptionRepoImpl) StoreSubPlan(ctx context.Context, plan *model.SubscriptionPlan) error {
	return r.db.WithContext(ctx).Create(plan).Error
}

func (r *subscriptionRepoImpl) CreateSubscription(ctx context.Context, sub *model.UserSubscription) error {
	return r.db.WithContext(ctx).Create(sub).Error
}

func (r *subscriptionRepoImpl) ActivateSubscription(ctx context.Context, subscriptionID string, start time.Time, next *time.Time) error {
	return r.db.WithContext(ctx).
		Model(&model.UserSubscription{}).
		Where("pay_pal_subscription_id = ?", subscriptionID).
		Updates(map[string]interface{}{
			"status":            "ACTIVE",
			"start_time":        start,
			"next_billing_time": next,
		}).Error
}

func (r *subscriptionRepoImpl) CancelSubscription(ctx context.Context, subscriptionID string) error {
	return r.db.WithContext(ctx).
		Model(&model.UserSubscription{}).
		Where("pay_pal_subscription_id = ?", subscriptionID).
		Update("status", "CANCELLED").
		Error
}

func (r *subscriptionRepoImpl) GetBySubscriptionID(ctx context.Context, subscriptionID string) (*model.UserSubscription, error) {
	var sub model.UserSubscription
	err := r.db.WithContext(ctx).
		Where("pay_pal_subscription_id = ?", subscriptionID).
		First(&sub).
		Error

	if err != nil {
		return nil, err
	}

	return &sub, nil
}

func (r *subscriptionRepoImpl) GetActiveByUser(ctx context.Context, userID string, merchantID string) (*model.UserSubscription, error) {
	var sub model.UserSubscription
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND merchant_id = ? AND status = ?", userID, merchantID, "ACTIVE").
		First(&sub).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &sub, nil
	}

	return &sub, err
}
