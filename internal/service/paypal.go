package service

import (
	"context"
	"paypal-integration-demo/internal/client"
	"paypal-integration-demo/internal/db"
	"paypal-integration-demo/internal/model"
)

type PaypalService interface {
	Pay(ctx context.Context, email string, items []*model.Item) (*client.CreateOrderResponse, error)
	CaptureOrder(ctx context.Context, orderID string) error
}

type paypalServiceImpl struct {
	paypalClient   client.PaypalClient
	serviceBaseUrl string
}

func NewPaypalService(paypalClient client.PaypalClient, serviceBaseUrl string) PaypalService {
	return &paypalServiceImpl{
		paypalClient:   paypalClient,
		serviceBaseUrl: serviceBaseUrl,
	}
}

func (s *paypalServiceImpl) Pay(ctx context.Context, email string, items []*model.Item) (*client.CreateOrderResponse, error) {
	order := model.Order{Email: email, Status: "CREATED"}
	db.DB.Create(&order)

	total := 0.0
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
		db.DB.Create(&model.OrderItem{
			OrderID:  order.ID,
			Type:     item.Type,
			Price:    item.Price,
			Quantity: item.Quantity,
		})
	}
	order.TotalAmount = total
	db.DB.Save(&order)

	return s.paypalClient.CreateOrder(ctx, s.serviceBaseUrl)
}

func (s *paypalServiceImpl) CaptureOrder(ctx context.Context, orderID string) error {
	return s.paypalClient.CaptureOrder(ctx, orderID)
}
