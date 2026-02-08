package model

import "time"

type Order struct {
	OrderID   string `gorm:"primaryKey;size:64;uniqueIndex;not null"` // paypal order id
	Status    string `gorm:"size:32;index;not null"`                  // CREATED, APPROVED, PAID, FAILED
	PayerID   string `gorm:"size:32;index;not null"`                  // buyer
	Amount    int32  `gorm:"not null"`                                // total amount (sum of items)
	Currency  string `gorm:"size:8;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type OrderItem struct {
	ID uint `gorm:"primaryKey"`
	// FK → orders.order_id
	OrderID string `gorm:"size:64;index;not null"`
	// Product info (snapshot at purchase time)
	ProductID   string `gorm:"size:64;index;not null"`
	ProductType string `gorm:"size:32;not null"` // ONE_TIME, SUBSCRIPTION
	Quantity    int32  `gorm:"not null"`
	UnitPrice   int32  `gorm:"not null"` // price per item
	Currency    string `gorm:"size:8;not null"`
	CreatedAt   time.Time

	Order Order `gorm:"foreignKey:OrderID;references:OrderID;constraint:OnDelete:CASCADE;"`
}

type Capture struct {
	CaptureID string `gorm:"primaryKey;size:64;uniqueIndex;not null"`
	// FK → orders.order_id
	OrderID   string `gorm:"size:64;index;not null"`
	Amount    int32  `gorm:"not null"`
	Currency  string `gorm:"size:8;not null"`
	Status    string `gorm:"size:32"` // COMPLETED
	CreatedAt time.Time

	Order Order `gorm:"foreignKey:OrderID;references:OrderID;constraint:OnDelete:CASCADE;"`
}

type Subscription struct {
	SubscriptionID string `gorm:"primaryKey;size:64;uniqueIndex;not null"`
	PlanID         string `gorm:"size:64"`
	CustomerID     string `gorm:"size:64;index;not null"`
	Status         string `gorm:"size:32;not null"` // CREATED, ACTIVE, CANCELLED
	StartedAt      *time.Time
	EndedAt        *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type WebhookEvent struct {
	EventID     string `gorm:"primaryKey;size:128;uniqueIndex;not null"`
	EventType   string `gorm:"size:64;index"`
	ProcessedAt time.Time
	CreatedAt   time.Time
}

type Item struct {
	Sku      string  `json:"sku"`
	Type     string  `json:"type"`
	Price    float64 `json:"price"`
	Quantity int     `json:"quantity"`
}
