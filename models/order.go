package models

import "time"

type Order struct {
	ID            uint `gorm:"primaryKey"`
	Email         string
	Status        string
	PayPalOrderID string
	TotalAmount   float64
	CreatedAt     time.Time
}

type OrderItem struct {
	ID        uint `gorm:"primaryKey"`
	OrderID   uint
	Type      string // ONE_TIME | SUBSCRIPTION
	Price     float64
	Quantity  int
	PaypalSub string
}

type Vault struct {
	ID        uint `gorm:"primaryKey"`
	Email     string
	Token     string
	CreatedAt time.Time
}
