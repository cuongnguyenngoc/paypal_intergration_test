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

	// -------- paypal --------
	paypal := api.Group("/paypal")
	paypal.POST("/connect", s.paypalHandler.ConnectMerchant)
	paypal.POST("/pay", s.paypalHandler.Pay)
	paypal.POST("/pay-again", s.paypalHandler.PayAgain)
	paypal.GET("/have-saved-payment", s.paypalHandler.CheckUserHaveSavedPayment)
	paypal.POST("/subscribe", s.paypalHandler.SubscribeSubscription)

	// -------- paypal webhooks / callbacks --------
	paypal.GET("/success", s.paypalHandler.HandleSuccess)
	paypal.POST("/webhook", s.paypalHandler.PayPalWebhook)
}

func (s *Server) Start(address string) error {
	return s.echo.Start(address)
}

func (s *Server) Shutdown() error {
	return s.echo.Close()
}
