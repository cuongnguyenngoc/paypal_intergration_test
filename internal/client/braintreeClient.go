package client

import (
	"context"
	"fmt"
	"paypal-integration-demo/internal/config"

	"github.com/braintree-go/braintree-go"
	"github.com/shopspring/decimal"
)

// --- INTERFACE ---

type BraintreeClient interface {
	// VaultPaymentMethod takes a frontend nonce and creates a customer, returning a permanent payment token
	VaultPaymentMethod(ctx context.Context, nonce, firstName, lastName, email string) (string, error)

	// ChargeOneTime charges a vaulted payment token for a specific amount
	ChargeOneTime(ctx context.Context, paymentToken string, amount string) (string, error)

	// CreateSubscription attaches a vaulted payment token to a billing plan
	CreateSubscription(ctx context.Context, paymentToken string, planID string) (string, error)

	// CancelSubscription cancels an active subscription
	CancelSubscription(ctx context.Context, subscriptionID string) error
}

// --- IMPLEMENTATION ---

type braintreeClientImpl struct {
	gateway *braintree.Braintree
}

// NewBraintreeClient initializes the Braintree SDK gateway
func NewBraintreeClient(cfg *config.Braintree) BraintreeClient {
	env := braintree.Sandbox
	if cfg.Environment == "production" {
		env = braintree.Production
	}

	gateway := braintree.New(
		env,
		cfg.MerchantID,
		cfg.PublicKey,
		cfg.PrivateKey,
	)

	return &braintreeClientImpl{
		gateway: gateway,
	}
}

// --- METHODS ---

func (c *braintreeClientImpl) VaultPaymentMethod(ctx context.Context, nonce, firstName, lastName, email string) (string, error) {
	req := &braintree.CustomerRequest{
		PaymentMethodNonce: nonce,
		FirstName:          firstName,
		LastName:           lastName,
		Email:              email,
	}

	customer, err := c.gateway.Customer().Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to vault payment method: %w", err)
	}

	if customer.DefaultPaymentMethod() == nil {
		return "", fmt.Errorf("no default payment method returned from vault")
	}

	return customer.DefaultPaymentMethod().GetToken(), nil
}

func (c *braintreeClientImpl) ChargeOneTime(ctx context.Context, paymentToken string, amount string) (string, error) {
	decAmount, err := decimal.NewFromString(amount)
	if err != nil {
		return "", fmt.Errorf("invalid amount format: %w", err)
	}

	// 2. Convert shopspring's Decimal to braintree's *Decimal format
	// Braintree expects NewDecimal(unscaled, scale). For 2 decimal places (like USD):
	// "50.00" * 100 = 5000 -> braintree.NewDecimal(5000, 2)
	cents := decAmount.Mul(decimal.NewFromInt(100)).IntPart()
	btAmount := braintree.NewDecimal(cents, 2)

	req := &braintree.TransactionRequest{
		Type:               "sale",
		Amount:             btAmount,
		PaymentMethodToken: paymentToken,
		Options: &braintree.TransactionOptions{
			SubmitForSettlement: true, // Captures the funds immediately
		},
	}

	tx, err := c.gateway.Transaction().Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("transaction creation failed: %w", err)
	}

	if tx.Status == braintree.TransactionStatusProcessorDeclined {
		return "", fmt.Errorf("transaction declined by processor: %s", tx.ProcessorResponseText)
	}

	return tx.Id, nil
}

func (c *braintreeClientImpl) CreateSubscription(ctx context.Context, paymentToken string, planID string) (string, error) {
	req := &braintree.SubscriptionRequest{
		PaymentMethodToken: paymentToken,
		PlanId:             planID,
	}

	sub, err := c.gateway.Subscription().Create(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to create subscription: %w", err)
	}

	return sub.Id, nil
}

func (c *braintreeClientImpl) CancelSubscription(ctx context.Context, subscriptionID string) error {
	_, err := c.gateway.Subscription().Cancel(ctx, subscriptionID)
	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}
	return nil
}
