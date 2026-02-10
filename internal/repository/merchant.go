package repository

import (
	"context"
	"paypal-integration-demo/internal/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MerchantRepository interface {
	Upsert(ctx context.Context, merchant *model.Merchant) error
	Get(ctx context.Context, merchantID string) (*model.Merchant, error)
}

type merchantRepoImpl struct {
	db *gorm.DB
}

func NewMerchantRepository(db *gorm.DB) MerchantRepository {
	return &merchantRepoImpl{
		db: db,
	}
}

func (r *merchantRepoImpl) Upsert(ctx context.Context, merchant *model.Merchant) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		DoUpdates: clause.Assignments(map[string]interface{}{
			"paypal_merchant_id":   merchant.PayPalMerchantID,
			"paypal_access_token":  merchant.PayPalAccessToken,
			"paypal_refresh_token": merchant.PayPalRefreshToken,
			"token_expires_at":     merchant.TokenExpiresAt,
			"updated_at":           time.Now(),
		}),
	}).Create(&merchant).Error
}

func (r *merchantRepoImpl) Get(ctx context.Context, merchantID string) (*model.Merchant, error) {
	var merchant model.Merchant
	err := r.db.WithContext(ctx).
		Where("id = ?", merchantID).
		First(&merchant).Error
	if err != nil {
		return nil, err
	}

	return &merchant, nil
}
