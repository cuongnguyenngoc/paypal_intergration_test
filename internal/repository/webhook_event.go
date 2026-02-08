package repository

import (
	"paypal-integration-demo/internal/model"
	"time"

	"gorm.io/gorm"
)

type WebhookEventRepository interface {
	Exists(payPalEventID string) (bool, error)
	MarkProcessed(eventID, eventType string) error
}

type webhookEventRepositoryIml struct {
	db *gorm.DB
}

func NewWebhookEventRepository(db *gorm.DB) WebhookEventRepository {
	return &webhookEventRepositoryIml{db: db}
}

func (r *webhookEventRepositoryIml) Exists(payPalEventID string) (bool, error) {
	var count int64
	err := r.db.Model(&model.WebhookEvent{}).
		Where("event_id = ?", payPalEventID).
		Count(&count).Error

	return count > 0, err
}

func (r *webhookEventRepositoryIml) MarkProcessed(eventID string, eventType string) error {
	return r.db.Create(&model.WebhookEvent{
		EventID:     eventID,
		EventType:   eventType,
		ProcessedAt: time.Now(),
	}).Error
}
