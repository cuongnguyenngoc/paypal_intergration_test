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
	Connect(merchantID string) string
	ExchangeAuthCode(ctx context.Context, code string) (*model.PayPalToken, error)
	Pay(ctx context.Context, userID string, items []*dto.Item) (*dto.PayResponse, error)
	PayAgain(ctx context.Context, userID string, items []*dto.Item) (*dto.PayResponse, error)
	CaptureOrder(ctx context.Context, orderID string) error
	HandleWebhook(ctx context.Context, headers http.Header, body []byte) error
	CheckUserHaveSavedPayment(ctx context.Context, userID string) (bool, error)
	SubscribeSubscription(ctx context.Context, userID string, productCode string) (approveURL string, err error)
}

type paypalServiceImpl struct {
	db               *gorm.DB
	paypalClient     client.PaypalClient
	serviceBaseUrl   string
	merchantRepo     repository.MerchantRepository
	productRepo      repository.ProductRepository
	orderRepo        repository.OrderRepository
	webhookEventRepo repository.WebhookEventRepository
	inventoryRepo    repository.InventoryRepository
	vaultRepo        repository.VaultRepository
	subscriptionRepo repository.SubscriptionRepository
}

func NewPaypalService(
	db *gorm.DB,
	paypalClient client.PaypalClient,
	serviceBaseUrl string,
	merchantRepo repository.MerchantRepository,
	productRepo repository.ProductRepository,
	orderRepo repository.OrderRepository,
	webhookEventRepo repository.WebhookEventRepository,
	inventoryRepo repository.InventoryRepository,
	vaultRepo repository.VaultRepository,
	subscriptionRepo repository.SubscriptionRepository,
) PaypalService {
	return &paypalServiceImpl{
		db:               db,
		paypalClient:     paypalClient,
		serviceBaseUrl:   serviceBaseUrl,
		merchantRepo:     merchantRepo,
		productRepo:      productRepo,
		orderRepo:        orderRepo,
		webhookEventRepo: webhookEventRepo,
		inventoryRepo:    inventoryRepo,
		vaultRepo:        vaultRepo,
		subscriptionRepo: subscriptionRepo,
	}
}

func (s *paypalServiceImpl) Connect(merchantID string) string {
	return s.paypalClient.BuildConnectURL(merchantID)
}

func (s *paypalServiceImpl) ExchangeAuthCode(ctx context.Context, code string) (*model.PayPalToken, error) {
	return s.paypalClient.ExchangeAuthCode(ctx, code)
}

func (s *paypalServiceImpl) Pay(ctx context.Context, userID string, items []*dto.Item) (*dto.PayResponse, error) {
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

	totalAmount := int32(0)
	orderItems := make([]*model.OrderItem, len(products))
	for i, product := range products {
		quantity := itemQuantityMap[product.ID]
		totalAmount += product.Price * quantity

		orderItems[i] = &model.OrderItem{
			ProductID: product.ID,
			Quantity:  quantity,
			UnitPrice: product.Price,
			Currency:  product.Currency,
		}
	}

	resp, err := s.paypalClient.CreateOrderForApproval(ctx, s.serviceBaseUrl, userID, "USD", totalAmount)
	if err != nil {
		return nil, fmt.Errorf("paypal api create order: %w", err)
	}

	for _, orderItem := range orderItems {
		orderItem.OrderID = resp.OrderID
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err = s.orderRepo.Create(tx, &model.Order{
			OrderID:  resp.OrderID,
			UserID:   userID,
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

	totalAmount := int32(0)
	orderItems := make([]*model.OrderItem, len(products))
	for i, product := range products {
		qty := itemQuantityMap[product.ID]
		totalAmount += product.Price * qty

		orderItems[i] = &model.OrderItem{
			ProductID: product.ID,
			Quantity:  qty,
			UnitPrice: product.Price,
			Currency:  product.Currency,
		}
	}

	orderID, err := s.paypalClient.CreateOrderWithVault(ctx, userID, vaultID, "USD", totalAmount)
	if err != nil {
		return nil, fmt.Errorf("paypal create order with vault: %w", err)
	}

	for _, orderItem := range orderItems {
		orderItem.OrderID = orderID
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.orderRepo.Create(tx, &model.Order{
			OrderID:  orderID,
			UserID:   userID,
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
	_, err := s.paypalClient.CaptureOrder(ctx, orderID)
	if err != nil {
		return fmt.Errorf("paypal api capture order: %w", err)
	}

	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.orderRepo.MarkCompleted(tx, orderID)
	})
}

func (s *paypalServiceImpl) CheckUserHaveSavedPayment(ctx context.Context, userID string) (bool, error) {
	vaultID, err := s.vaultRepo.GetVaultID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("no vaulted payment method")
	}

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

	switch eventPayload.EventType {
	case "PAYMENT.CAPTURE.COMPLETED":
		// mark order as paid, grant items to user
		return s.handleOrderPaid(ctx, &eventPayload)
	case "VAULT.PAYMENT-TOKEN.CREATED":
		return s.handlePaymentTokenCreated(ctx, &eventPayload)
	case "BILLING.SUBSCRIPTION.ACTIVATED":
		// activate subscription
		fmt.Println("subscription activated")
		return s.handleSubscriptionActivated(ctx, &eventPayload)
	case "BILLING.SUBSCRIPTION.CANCELLED":
		fmt.Println("subscription canceled")
		return s.handleSubscriptionCancelled(ctx, &eventPayload)
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
				UserID:    orderInfo.UserID,
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
	if resource.ID == "" {
		return fmt.Errorf("missing vault_id in PAYMENT.TOKEN.CREATED event payload")
	}

	orderID := resource.Metadata.OrderID
	if orderID == "" {
		return fmt.Errorf("missing user id in PAYMENT.TOKEN.CREATED event payload")
	}

	orderInfo, err := s.orderRepo.FindByOrderID(orderID)
	if err != nil {
		return fmt.Errorf("mark order paid: %w", err)
	}

	// Upsert user vault info
	err = s.vaultRepo.Create(ctx, &model.UserVault{
		UserID:   orderInfo.UserID,
		VaultID:  resource.ID,
		Provider: "paypal",
	})
	if err != nil {
		return fmt.Errorf("save user paypal vault: %w", err)
	}

	return nil
}

func (s *paypalServiceImpl) handleSubscriptionActivated(ctx context.Context, event *model.PayPalWebhookEvent) error {
	res := event.Resource.Subscription

	if res.ID == "" || res.CustomID == "" {
		return fmt.Errorf("invalid subscription webhook")
	}

	return s.subscriptionRepo.ActivateSubscription(ctx,
		res.ID,
		res.StartTime,
		res.BillingInfo.NextBillingTime,
	)
}

func (s *paypalServiceImpl) handleSubscriptionCancelled(ctx context.Context, event *model.PayPalWebhookEvent) error {
	subID := event.Resource.Subscription.ID
	if subID == "" {
		return nil
	}

	return s.subscriptionRepo.CancelSubscription(ctx, subID)
}

func (s *paypalServiceImpl) SubscribeSubscription(ctx context.Context, userID string, productCode string) (approveURL string, err error) {
	plan, err := s.subscriptionRepo.GetPaypalPlanByProductCode(ctx, productCode)
	if err != nil {
		return "", err
	}

	subID, approveURL, err := s.paypalClient.CreateUserSubscription(
		ctx,
		s.serviceBaseUrl,
		plan.PayPalPlanID,
		userID,
	)
	if err != nil {
		return "", err
	}

	// store as PENDING, will activate via webhook
	err = s.subscriptionRepo.CreateSubscription(ctx, &model.UserSubscription{
		UserID:               userID,
		ProductCode:          productCode,
		PayPalSubscriptionID: subID,
		Status:               "PENDING",
	})
	if err != nil {
		return "", err
	}

	return approveURL, nil
}
