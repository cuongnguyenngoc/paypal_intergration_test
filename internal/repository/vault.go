package repository

import (
	"context"
	"errors"
	"fmt"
	"paypal-integration-demo/internal/model"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type VaultRepository interface {
	Create(ctx context.Context, tx *gorm.DB, vault *model.UserVault) error
	GetVaultID(ctx context.Context, userID string) (string, error)
}

type vaultRepoImpl struct {
	db *gorm.DB
}

func NewVaultRepository(db *gorm.DB) VaultRepository {
	return &vaultRepoImpl{
		db: db,
	}
}

func (r *vaultRepoImpl) Create(ctx context.Context, tx *gorm.DB, vault *model.UserVault) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "user_id"}, {Name: "vault_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"updated_at": time.Now(),
		}),
	}).Create(&vault).Error
}

func (r *vaultRepoImpl) GetVaultID(ctx context.Context, userID string) (string, error) {
	var vaultID string
	err := r.db.WithContext(ctx).
		Model(&model.UserVault{}).
		Where("user_id = ?", userID).
		Pluck("vault_id", &vaultID).
		Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("no vault id found for user %s", userID)
		}
		return "", err
	}

	return vaultID, nil
}
