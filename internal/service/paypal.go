package service

import (
	"context"
	"fmt"
	"paypal-integration-demo/internal/client"
	"paypal-integration-demo/internal/model"
	"paypal-integration-demo/internal/repository"
)

type PaypalService interface {
	Pay(ctx context.Context, email string, items []*model.Item) (*client.CreateOrderResponse, error)
	CaptureOrder(ctx context.Context, orderID string) error
}

type paypalServiceImpl struct {
	paypalClient     client.PaypalClient
	serviceBaseUrl   string
	orderRepo        repository.OrderRepository
	captureRepo      repository.CaptureRepository
	webhookEventRepo repository.WebhookEventRepository
}

func NewPaypalService(
	paypalClient client.PaypalClient,
	serviceBaseUrl string,
	orderRepo repository.OrderRepository,
	captureRepo repository.CaptureRepository,
	webhookEventRepo repository.WebhookEventRepository,
) PaypalService {
	return &paypalServiceImpl{
		paypalClient:     paypalClient,
		serviceBaseUrl:   serviceBaseUrl,
		orderRepo:        orderRepo,
		captureRepo:      captureRepo,
		webhookEventRepo: webhookEventRepo,
	}
}

func (s *paypalServiceImpl) Pay(ctx context.Context, email string, items []*model.Item) (*client.CreateOrderResponse, error) {
	// db.DB.Create(&order)

	// total := 0.0
	// for _, item := range items {
	// 	total += item.Price * float64(item.Quantity)
	// 	db.DB.Create(&model.OrderItem{
	// 		OrderID:  order.ID,
	// 		Type:     item.Type,
	// 		Price:    item.Price,
	// 		Quantity: item.Quantity,
	// 	})
	// }
	// order.TotalAmount = total
	// db.DB.Save(&order)

	resp, err := s.paypalClient.CreateOrder(ctx, s.serviceBaseUrl)
	if err != nil {
		return nil, fmt.Errorf("paypal api create order: %w", err)
	}

	order := model.Order{
		OrderID:  resp.OrderID,
		Status:   "CREATED",
		Amount:   1,
		Currency: "USD",
		PayerID:  resp.PaypalAccount.AccountID,
	}
	err = s.orderRepo.Create(&order)
	if err != nil {
		return nil, fmt.Errorf("store order in db: %w", err)
	}

	return resp, nil
}

func (s *paypalServiceImpl) CaptureOrder(ctx context.Context, orderID string) error {
	return s.paypalClient.CaptureOrder(ctx, orderID)
}
