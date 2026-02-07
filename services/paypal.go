package services

import (
	"context"
	"paypal-integration-demo/client"
	"paypal-integration-demo/db"
	"paypal-integration-demo/models"
)

type PaypalService interface {
	Pay(ctx context.Context, email string, items []*models.Item) (map[string]interface{}, error)
}

type paypalServiceImpl struct {
	paypalClient client.PaypalClient
}

func NewPaypalService(paypalClient client.PaypalClient) PaypalService {
	return &paypalServiceImpl{
		paypalClient: paypalClient,
	}
}

func (s *paypalServiceImpl) Pay(ctx context.Context, email string, items []*models.Item) (map[string]interface{}, error) {
	order := models.Order{Email: email, Status: "CREATED"}
	db.DB.Create(&order)

	total := 0.0
	for _, item := range items {
		total += item.Price * float64(item.Quantity)
		db.DB.Create(&models.OrderItem{
			OrderID:  order.ID,
			Type:     item.Type,
			Price:    item.Price,
			Quantity: item.Quantity,
		})
	}
	order.TotalAmount = total
	db.DB.Save(&order)

	return s.paypalClient.Pay(ctx)
}
