package service

import (
	"context"
	"paypal-integration-demo/internal/model"
	"paypal-integration-demo/internal/repository"
	"time"

	"github.com/google/uuid"
)

type MerchantService interface {
	CreateMerchant(ctx context.Context, name string) (string, error)
	UpdatePaypalTokens(ctx context.Context, merchantID string, tokens *model.PayPalToken) error
	GetMerchant(ctx context.Context, id string) (*model.Merchant, error)
}

type merchantServiceImpl struct {
	merchantRepo repository.MerchantRepository
}

func NewMerchantService(
	merchantRepo repository.MerchantRepository,
) MerchantService {
	return &merchantServiceImpl{
		merchantRepo: merchantRepo,
	}
}

func (s *merchantServiceImpl) CreateMerchant(ctx context.Context, name string) (string, error) {
	merchant := &model.Merchant{
		ID:   uuid.NewString(),
		Name: name,
	}
	err := s.merchantRepo.Upsert(ctx, merchant)
	if err != nil {
		return "", err
	}

	return merchant.ID, nil
}

func (s *merchantServiceImpl) UpdatePaypalTokens(ctx context.Context, merchantID string, tokens *model.PayPalToken) error {
	expiresAt := time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)
	return s.merchantRepo.Upsert(ctx, &model.Merchant{
		ID:                 merchantID,
		PayPalAccessToken:  tokens.AccessToken,
		PayPalRefreshToken: tokens.RefreshToken,
		TokenExpiresAt:     &expiresAt,
	})
}

func (s *merchantServiceImpl) GetMerchant(ctx context.Context, id string) (*model.Merchant, error) {
	return s.merchantRepo.Get(ctx, id)
}
