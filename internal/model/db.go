package model

import "time"

type ProductType string

const (
	SUBSCRIPTION ProductType = "SUBSCRIPTION"
	ONE_TIME     ProductType = "ONE_TIME"
)

type Product struct {
	ID          string `gorm:"primaryKey;size:64;not null"` // product sku
	Name        string
	Description string
	Price       int32  `gorm:"not null"`
	Currency    string `gorm:"size:8;not null"`
	Type        string `gorm:"size:32;index;not null"` // ONE_TIME, SUBSCRIPTION
}

type Order struct {
	OrderID    string `gorm:"primaryKey;size:64;not null"` // paypal order id
	Status     string `gorm:"size:32;index;not null"`      // CREATED, APPROVED, PAID, FAILED
	UserID     string `gorm:"size:32;index"`
	Amount     int32  `gorm:"not null"` // total amount (sum of items)
	Currency   string `gorm:"size:8;not null"`
	MerchantID string `gorm:"not null"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type OrderItem struct {
	ID uint `gorm:"primaryKey"`
	// FK → order.order_id
	OrderID string `gorm:"size:64;index;not null"`
	// FK → product.id
	ProductID string `gorm:"index;not null"`
	Quantity  int32  `gorm:"not null"`
	UnitPrice int32  `gorm:"not null"`
	Currency  string `gorm:"size:8;not null"`

	CreatedAt time.Time
}

type WebhookEvent struct {
	EventID     string `gorm:"primaryKey;size:128;uniqueIndex;not null"`
	EventType   string `gorm:"size:64;index"`
	ProcessedAt time.Time
	CreatedAt   time.Time
}

type UserInventory struct {
	UserID    string `gorm:"primaryKey;size:32;"`
	ProductID string `gorm:"primaryKey;index;not null"`
	Quantity  int32  `gorm:"not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserVault struct {
	UserID   string `gorm:"primaryKey;not null"`
	VaultID  string `gorm:"primaryKey;uniqueIndex;not null"`
	Provider string

	// IsActive  bool `gorm:"not null;default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type SubscriptionPlan struct {
	MerchantID      string `gorm:"primaryKey"`
	ProductID       string `gorm:"primaryKey"` // vip_monthly
	PayPalProductID string
	PayPalPlanID    string
}

type UserSubscription struct {
	ID                   uint   `gorm:"primaryKey"`
	UserID               string `gorm:"index"`
	ProductID            string
	MerchantID           string
	PayPalSubscriptionID string `gorm:"size:64;uniqueIndex"`
	Status               string // ACTIVE, CANCELLED, SUSPENDED
	StartTime            time.Time
	NextBillingTime      *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Merchant struct {
	ID   string `gorm:"primaryKey"`
	Name string

	PayPalMerchantID   string
	PayPalAccessToken  string
	PayPalRefreshToken string
	TokenExpiresAt     *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time
}
