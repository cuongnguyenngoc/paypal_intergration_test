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
	ClearPayPalTokens(ctx context.Context, merchantID string) error
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
			"pay_pal_merchant_id":   merchant.PayPalMerchantID,
			"pay_pal_access_token":  merchant.PayPalAccessToken,
			"pay_pal_refresh_token": merchant.PayPalRefreshToken,
			"token_expires_at":      merchant.TokenExpiresAt,
			"updated_at":            time.Now(),
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

func (r *merchantRepoImpl) ClearPayPalTokens(ctx context.Context, merchantID string) error {
	result := r.db.
		WithContext(ctx).
		Model(&model.Merchant{}).
		Where("id = ?", merchantID).
		Updates(map[string]interface{}{
			"pay_pal_access_token":  "",
			"pay_pal_refresh_token": "",
			"token_expires_at":      nil,
		})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
