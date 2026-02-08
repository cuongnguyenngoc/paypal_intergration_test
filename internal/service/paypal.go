package service

import (
	"context"
	"fmt"
	"paypal-integration-demo/internal/client"
	"paypal-integration-demo/internal/dto"
	"paypal-integration-demo/internal/model"
	"paypal-integration-demo/internal/repository"
)

type PaypalService interface {
	Pay(ctx context.Context, email string, items []*dto.Item) (*client.CreateOrderResponse, error)
	CaptureOrder(ctx context.Context, orderID string) error
}

type paypalServiceImpl struct {
	paypalClient     client.PaypalClient
	serviceBaseUrl   string
	productRepo      repository.ProductRepository
	orderRepo        repository.OrderRepository
	captureRepo      repository.CaptureRepository
	webhookEventRepo repository.WebhookEventRepository
}

func NewPaypalService(
	paypalClient client.PaypalClient,
	serviceBaseUrl string,
	productRepo repository.ProductRepository,
	orderRepo repository.OrderRepository,
	captureRepo repository.CaptureRepository,
	webhookEventRepo repository.WebhookEventRepository,
) PaypalService {
	return &paypalServiceImpl{
		paypalClient:     paypalClient,
		serviceBaseUrl:   serviceBaseUrl,
		productRepo:      productRepo,
		orderRepo:        orderRepo,
		captureRepo:      captureRepo,
		webhookEventRepo: webhookEventRepo,
	}
}

func (s *paypalServiceImpl) Pay(ctx context.Context, email string, items []*dto.Item) (*client.CreateOrderResponse, error) {
	productIDs := make([]string, len(items))
	itemQuantityMap := make(map[string]int32)
	for i, item := range items {
		if item.Quantity <= 0 {
			return nil, fmt.Errorf("item quantity must be positive")
		}
		productIDs[i] = item.Sku

		itemQuantityMap[item.Sku] = item.Quantity
	}
	products, err := s.productRepo.FindMany(productIDs)
	if err != nil {
		return nil, fmt.Errorf("get many products by item ids")
	}

	if len(products) != len(items) {
		return nil, fmt.Errorf("some products not found")
	}

	resp, err := s.paypalClient.CreateOrder(ctx, s.serviceBaseUrl)
	if err != nil {
		return nil, fmt.Errorf("paypal api create order: %w", err)
	}

	totalAmount := int32(0)
	orderItems := make([]*model.OrderItem, len(products))
	for i, product := range products {
		quantity := itemQuantityMap[product.ID]
		totalAmount += product.Price * quantity

		orderItems[i] = &model.OrderItem{
			OrderID:   resp.OrderID,
			ProductID: product.ID,
			Quantity:  quantity,
			UnitPrice: product.Price,
			Currency:  product.Currency,
		}
	}

	err = s.orderRepo.Create(&model.Order{
		OrderID:  resp.OrderID,
		Status:   "CREATED",
		Amount:   totalAmount,
		Currency: "USD",
		PayerID:  resp.PaypalAccount.AccountID,
	})
	if err != nil {
		return nil, fmt.Errorf("store order in db: %w", err)
	}

	err = s.orderRepo.CreateOrderItems(orderItems)
	if err != nil {
		return nil, fmt.Errorf("store order items in db: %w", err)
	}

	return resp, nil
}

func (s *paypalServiceImpl) CaptureOrder(ctx context.Context, orderID string) error {
	return s.paypalClient.CaptureOrder(ctx, orderID)
}
