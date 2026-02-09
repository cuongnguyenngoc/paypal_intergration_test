package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"paypal-integration-demo/internal/client"
	"paypal-integration-demo/internal/config"
	"paypal-integration-demo/internal/repository"
	"paypal-integration-demo/internal/server"
	"paypal-integration-demo/internal/service"
	"syscall"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

func main() {
	// load .env into os.Environ
	if err := godotenv.Load("../../.env"); err != nil {
		fmt.Println("No .env file found (ok in prod)")
	}

	cfg := &config.Config{}
	if err := env.Parse(cfg); err != nil {
		fmt.Printf("Failed to parse config: %v\n", err)
		os.Exit(1)
	}

	db := client.InitMysqlClient(cfg.DatabaseURL)
	paypalClient := client.NewPaypalClient(&cfg.Paypal)

	productRepo := repository.NewProductRepository(db)
	err := productRepo.Seed()
	if err != nil {
		log.Fatal("seed some products data into db")
	}

	orderRepo := repository.NewOrderRepository(db)
	webhookEventRepo := repository.NewWebhookEventRepository(db)
	inventoryRepo := repository.NewInventoryRepository(db)
	vaultRepo := repository.NewVaultRepository(db)

	paypalService := service.NewPaypalService(
		paypalClient, cfg.BaseURL,
		productRepo,
		orderRepo,
		webhookEventRepo,
		inventoryRepo,
		vaultRepo,
	)
	userService := service.NewUserService(inventoryRepo)

	serverAddr := cfg.HTTP.Host + ":" + cfg.HTTP.Port

	// Init HTTP server
	srv := server.NewServer(paypalService, userService)

	log.Println("Starting HTTP server on", serverAddr)
	go func() {
		if err := srv.Start(serverAddr); err != nil && err != http.ErrServerClosed {
			fmt.Println("err", err)
			log.Fatal("HTTP server error", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	<-sigChan
	log.Println("Signal received, starting graceful shutdown...")

	_, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(); err != nil {
		log.Fatal("HTTP server shutdown error")
	}
}
