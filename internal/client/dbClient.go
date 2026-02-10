package client

import (
	"log"
	"paypal-integration-demo/internal/model"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func InitMysqlClient(databaseURL string) *gorm.DB {
	var err error
	db, err := gorm.Open(mysql.Open(databaseURL), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal(err)
	}

	// Connection pool (important for webhooks)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(50)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := db.AutoMigrate(
		&model.Product{},
		&model.Merchant{},
		&model.Order{},
		&model.OrderItem{},
		&model.UserVault{},
		&model.WebhookEvent{},
		&model.UserInventory{},
		&model.PayPalPlan{},
		&model.UserSubscription{},
	); err != nil {
		log.Fatal(err)
	}

	return db
}
