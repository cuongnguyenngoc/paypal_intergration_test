package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"paypal-integration-demo/client"
	"paypal-integration-demo/config"
	"paypal-integration-demo/db"
	"paypal-integration-demo/models"
	"paypal-integration-demo/server"
	"paypal-integration-demo/services"
	"syscall"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/joho/godotenv"
)

func main() {
	// load .env into os.Environ
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found (ok in prod)")
	}

	db.Init()
	db.DB.AutoMigrate(&models.Order{}, &models.OrderItem{}, &models.Vault{})

	cfg := &config.Config{}
	if err := env.Parse(cfg); err != nil {
		fmt.Printf("Failed to parse config: %v\n", err)
		os.Exit(1)
	}

	paypalClient := client.NewPaypalClient(&cfg.Paypal)
	paypalService := services.NewPaypalService(paypalClient, cfg.BaseURL)

	serverAddr := cfg.HTTP.Host + ":" + cfg.HTTP.Port

	// Init HTTP server
	srv := server.NewServer(paypalService)

	log.Println("Starting HTTP server on", serverAddr)
	go func() {
		if err := srv.Start(serverAddr); err != nil && err != http.ErrServerClosed {
			log.Fatal("HTTP server error")
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
