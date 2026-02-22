package server

import (
	"paypal-integration-demo/internal/handler"
	"paypal-integration-demo/internal/service"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	echo            *echo.Echo
	paypalHandler   *handler.PaypalHandler
	userHandler     *handler.UserHandler
	merchantHandler *handler.MerchantHandler
}

func NewServer(paypalService service.PaypalService, userService service.UserService, merchantService service.MerchantService) *Server {
	e := echo.New()

	e.File("/", "../../web/index.html")
	e.File("/braintree", "../../web/braintree.html")

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	paypalHandler := handler.NewPaypalHandler(paypalService, merchantService)
	userHandler := handler.NewUserHandler(userService)
	merchantHandler := handler.NewMerchantHandler(merchantService)

	s := &Server{
		echo:            e,
		paypalHandler:   paypalHandler,
		userHandler:     userHandler,
		merchantHandler: merchantHandler,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	api := s.echo.Group("/api")

	api.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	api.GET("/inventories", s.userHandler.GetUsersInventory)
	api.POST("/merchants/create", s.merchantHandler.CreateMerchant)
	api.GET("/merchants/:merchantID/paypal/connect", s.paypalHandler.ConnectMerchant)
	api.GET("/merchants/:merchantID/paypal/status", s.merchantHandler.PayPalStatus)
	api.POST("/merchants/:merchantID/paypal/disconnect", s.merchantHandler.DisconnectPayPal)

	// -------- paypal --------
	paypal := api.Group("/paypal")
	paypal.GET("/oauth/callback", s.paypalHandler.OAuthCallback)
	paypal.POST("/pay", s.paypalHandler.Pay)
	paypal.POST("/pay-again", s.paypalHandler.PayAgain)
	paypal.GET("/have-saved-payment", s.paypalHandler.CheckUserHaveSavedPayment)
	// -------- paypal webhooks / callbacks --------
	paypal.GET("/success", s.paypalHandler.HandleSuccess)
	paypal.POST("/webhook", s.paypalHandler.PayPalWebhook)

	subscription := paypal.Group("/subscription")
	subscription.POST("/subscribe", s.paypalHandler.SubscribeSubscription)
	subscription.GET("/success", s.paypalHandler.HandleSubscriptionSuccess)
	subscription.GET("/status", s.paypalHandler.GetSubscriptionStatus)
	subscription.POST("/cancel", s.paypalHandler.CancelSubscription)

	// -------- braintree --------
	api.POST("/braintree/checkout", s.paypalHandler.ProcessCheckout)
	api.POST("/braintree/checkoutWithSavedCard", s.paypalHandler.ProcessCheckoutWithSavedCard)
}

func (s *Server) Start(address string) error {
	return s.echo.Start(address)
}

func (s *Server) Shutdown() error {
	return s.echo.Close()
}
