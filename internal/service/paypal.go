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
	Pay(ctx context.Context, items []*dto.Item) (*dto.PayResponse, error)
	CaptureOrder(ctx context.Context, orderID string) error
	HandleWebhook(ctx context.Context, eventPayload *model.PayPalWebhookEvent) error
}

type paypalServiceImpl struct {
	paypalClient     client.PaypalClient
	serviceBaseUrl   string
	productRepo      repository.ProductRepository
	orderRepo        repository.OrderRepository
	webhookEventRepo repository.WebhookEventRepository
	inventoryRepo    repository.InventoryRepository
	vaultRepo        repository.VaultRepository
}

func NewPaypalService(
	paypalClient client.PaypalClient,
	serviceBaseUrl string,
	productRepo repository.ProductRepository,
	orderRepo repository.OrderRepository,
	webhookEventRepo repository.WebhookEventRepository,
	inventoryRepo repository.InventoryRepository,
	vaultRepo repository.VaultRepository,
) PaypalService {
	return &paypalServiceImpl{
		paypalClient:     paypalClient,
		serviceBaseUrl:   serviceBaseUrl,
		productRepo:      productRepo,
		orderRepo:        orderRepo,
		webhookEventRepo: webhookEventRepo,
		inventoryRepo:    inventoryRepo,
		vaultRepo:        vaultRepo,
	}
}

func (s *paypalServiceImpl) Pay(ctx context.Context, items []*dto.Item) (*dto.PayResponse, error) {
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
	})
	if err != nil {
		return nil, fmt.Errorf("store order in db: %w", err)
	}

	err = s.orderRepo.CreateOrderItems(orderItems)
	if err != nil {
		return nil, fmt.Errorf("store order items in db: %w", err)
	}

	return &dto.PayResponse{
		OrderID:          resp.OrderID,
		OrderApprovalURL: resp.ApproveURL,
	}, nil
}

func (s *paypalServiceImpl) CaptureOrder(ctx context.Context, orderID string) error {
	resp, err := s.paypalClient.CaptureOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("paypal api capture order: %w", err)
	}

	orderDetail, err := s.paypalClient.GetOrderDetails(ctx, orderID)
	if err != nil {
		return fmt.Errorf("paypal api get order: %w", err)
	}

	err = s.vaultRepo.Create(ctx, &model.VaultedPaymentMethod{
		UserID:   orderDetail.PaymentSource.PayPal.PayerID,
		VaultID:  orderDetail.PaymentSource.PayPal.VaultID,
		Provider: "paypal",
	})
	if err != nil {
		return fmt.Errorf("store user vault info to db: %w", err)
	}

	return s.orderRepo.MarkCompleted(orderID, resp.PayerID)
}

func (s *paypalServiceImpl) HandleWebhook(ctx context.Context, eventPayload *model.PayPalWebhookEvent) error {
	switch eventPayload.EventType {
	case "PAYMENT.CAPTURE.COMPLETED":
		// mark order as paid
		fmt.Println("payment completed")
		return s.handleOrderPaid(ctx, eventPayload)
	case "BILLING.SUBSCRIPTION.ACTIVATED":
		// activate subscription
		fmt.Println("subscription activated")
	}

	return nil
}

func (s *paypalServiceImpl) handleOrderPaid(ctx context.Context, eventPayload *model.PayPalWebhookEvent) error {
	orderID := eventPayload.Resource.SupplementaryData.RelatedIDs.OrderID
	if orderID == "" {
		return fmt.Errorf("could not find order_id in webhook payload")
	}

	orderInfo, err := s.orderRepo.MarkPaid(orderID)
	if err != nil {
		return fmt.Errorf("mark order paid: %w", err)
	}

	orderItems, err := s.orderRepo.GetOrderItems(orderID)
	if err != nil {
		return fmt.Errorf("get order items: %w", err)
	}

	// grant items to user inventory
	for _, item := range orderItems {
		err = s.inventoryRepo.Upsert(ctx, &model.UserInventory{
			UserID:    orderInfo.PayerID,
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
		})
		if err != nil {
			return fmt.Errorf("update user inventory: %w", err)
		}
	}

	return nil
}
