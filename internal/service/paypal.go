package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"paypal-integration-demo/internal/client"
	"paypal-integration-demo/internal/dto"
	"paypal-integration-demo/internal/model"
	"paypal-integration-demo/internal/repository"

	"gorm.io/gorm"
)

type PaypalService interface {
	Pay(ctx context.Context, items []*dto.Item) (*dto.PayResponse, error)
	PayAgain(ctx context.Context, userID string, items []*dto.Item) (*dto.PayResponse, error)
	CaptureOrder(ctx context.Context, orderID string) error
	HandleWebhook(ctx context.Context, headers http.Header, body []byte) error
	CheckUserHaveSavedPayment(ctx context.Context, userID string) (bool, error)
}

type paypalServiceImpl struct {
	db               *gorm.DB
	paypalClient     client.PaypalClient
	serviceBaseUrl   string
	productRepo      repository.ProductRepository
	orderRepo        repository.OrderRepository
	webhookEventRepo repository.WebhookEventRepository
	inventoryRepo    repository.InventoryRepository
	vaultRepo        repository.VaultRepository
}

func NewPaypalService(
	db *gorm.DB,
	paypalClient client.PaypalClient,
	serviceBaseUrl string,
	productRepo repository.ProductRepository,
	orderRepo repository.OrderRepository,
	webhookEventRepo repository.WebhookEventRepository,
	inventoryRepo repository.InventoryRepository,
	vaultRepo repository.VaultRepository,
) PaypalService {
	return &paypalServiceImpl{
		db:               db,
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

	resp, err := s.paypalClient.CreateOrderForApproval(ctx, s.serviceBaseUrl)
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

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err = s.orderRepo.Create(tx, &model.Order{
			OrderID:  resp.OrderID,
			Status:   "CREATED",
			Amount:   totalAmount,
			Currency: "USD",
		})
		if err != nil {
			return fmt.Errorf("store order in db: %w", err)
		}

		err = s.orderRepo.CreateOrderItems(tx, orderItems)
		if err != nil {
			return fmt.Errorf("store order items in db: %w", err)
		}
		return nil
	})

	return &dto.PayResponse{
		OrderID:          resp.OrderID,
		OrderApprovalURL: resp.ApproveURL,
	}, nil
}

func (s *paypalServiceImpl) PayAgain(ctx context.Context, userID string, items []*dto.Item) (*dto.PayResponse, error) {
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
		return nil, fmt.Errorf("get products: %w", err)
	}
	if len(products) != len(items) {
		return nil, fmt.Errorf("some products not found")
	}

	vaultID, err := s.vaultRepo.GetVaultID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("no vaulted payment method")
	}

	orderID, err := s.paypalClient.CreateOrderWithVault(ctx, vaultID)
	if err != nil {
		return nil, fmt.Errorf("paypal create order with vault: %w", err)
	}
	fmt.Println("orderID vault", orderID)

	totalAmount := int32(0)
	orderItems := make([]*model.OrderItem, len(products))

	for i, product := range products {
		qty := itemQuantityMap[product.ID]
		totalAmount += product.Price * qty

		orderItems[i] = &model.OrderItem{
			OrderID:   orderID,
			ProductID: product.ID,
			Quantity:  qty,
			UnitPrice: product.Price,
			Currency:  product.Currency,
		}
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.orderRepo.Create(tx, &model.Order{
			OrderID:  orderID,
			PayerID:  userID,
			Status:   "COMPLETED", // paypal auto capture order when create order with vault so order status should be compeleted
			Amount:   totalAmount,
			Currency: "USD",
		}); err != nil {
			return err
		}

		if err := s.orderRepo.CreateOrderItems(tx, orderItems); err != nil {
			return err
		}

		return nil
	})

	return &dto.PayResponse{
		OrderID: orderID,
	}, nil
}

func (s *paypalServiceImpl) CaptureOrder(ctx context.Context, orderID string) error {
	resp, err := s.paypalClient.CaptureOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("paypal api capture order: %w", err)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.orderRepo.MarkCompleted(tx, orderID, resp.PayerID)
	})
}

func (s *paypalServiceImpl) CheckUserHaveSavedPayment(ctx context.Context, userID string) (bool, error) {
	vaultID, err := s.vaultRepo.GetVaultID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("no vaulted payment method")
	}
	fmt.Println("vaultID", vaultID)

	return vaultID != "", nil
}

func (s *paypalServiceImpl) HandleWebhook(ctx context.Context, headers http.Header, body []byte) error {
	err := s.paypalClient.VerifyWebhookSignature(ctx, headers, body)
	if err != nil {
		return fmt.Errorf("verify webhook signature: %w", err)
	}

	var eventPayload model.PayPalWebhookEvent
	if err := json.Unmarshal(body, &eventPayload); err != nil {
		return fmt.Errorf("decode webhook payload: %w", err)
	}
	fmt.Println("eventPayload", eventPayload)
	fmt.Println("eventPayload.EventType", eventPayload.EventType)

	switch eventPayload.EventType {
	case "PAYMENT.CAPTURE.COMPLETED":
		// mark order as paid
		fmt.Println("payment completed")
		return s.handleOrderPaid(ctx, &eventPayload)
	case "VAULT.PAYMENT-TOKEN.CREATED":
		return s.handlePaymentTokenCreated(ctx, &eventPayload)
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

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		orderInfo, err := s.orderRepo.MarkPaid(tx, orderID)
		if err != nil {
			return fmt.Errorf("mark order paid: %w", err)
		}

		orderItems, err := s.orderRepo.GetOrderItems(tx, orderID)
		if err != nil {
			return fmt.Errorf("get order items: %w", err)
		}

		// grant items to user inventory
		for _, item := range orderItems {
			err = s.inventoryRepo.Upsert(ctx, tx, &model.UserInventory{
				UserID:    orderInfo.PayerID,
				ProductID: item.ProductID,
				Quantity:  item.Quantity,
			})
			if err != nil {
				return fmt.Errorf("update user inventory: %w", err)
			}
		}

		return nil
	})
}

func (s *paypalServiceImpl) handlePaymentTokenCreated(ctx context.Context, event *model.PayPalWebhookEvent) error {
	resource := event.Resource
	fmt.Println("resource", resource)
	if resource.ID == "" {
		return fmt.Errorf("missing vault_id in PAYMENT.TOKEN.CREATED")
	}

	userID := event.Resource.PaymentResource.PayPal.PayerID
	if userID == "" {
		return fmt.Errorf("missing payer_id in PAYMENT.TOKEN.CREATED")
	}

	fmt.Println("userID", userID)

	// Upsert user vault info
	err := s.vaultRepo.Create(ctx, &model.UserVault{
		UserID:  userID,
		VaultID: resource.ID,
	})
	if err != nil {
		return fmt.Errorf("save user paypal vault: %w", err)
	}

	return nil
}
