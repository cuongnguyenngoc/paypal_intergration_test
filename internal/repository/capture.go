package repository

import (
	"paypal-integration-demo/internal/model"

	"gorm.io/gorm"
)

type CaptureRepository interface {
	Create(capture *model.Capture) error
	Exists(captureID string) (bool, error)
}

type captureRepositoryImpl struct {
	db *gorm.DB
}

func NewCaptureRepository(db *gorm.DB) CaptureRepository {
	return &captureRepositoryImpl{
		db: db,
	}
}

func (r *captureRepositoryImpl) Create(capture *model.Capture) error {
	return r.db.Create(capture).Error
}

func (r *captureRepositoryImpl) Exists(captureID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.Capture{}).
		Where("capture_id = ?", captureID).
		Count(&count).Error

	return count > 0, err
}
