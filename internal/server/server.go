package server

import (
	"paypal-integration-demo/internal/handler"
	"paypal-integration-demo/internal/service"

	internalMiddleware "paypal-integration-demo/internal/middleware"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	echo          *echo.Echo
	paypalHandler *handler.PaypalHandler
	userHandler   *handler.UserHandler
}

func NewServer(paypalService service.PaypalService, userService service.UserService) *Server {
	e := echo.New()

	e.File("/", "../../web/index.html")

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	paypalHandler := handler.NewPaypalHandler(paypalService)
	userHandler := handler.NewUserHandler(userService)

	s := &Server{
		echo:          e,
		paypalHandler: paypalHandler,
		userHandler:   userHandler,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	api := s.echo.Group("/api")

	// -------- public --------
	api.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// -------- authenticated --------
	auth := api.Group("", internalMiddleware.AuthMiddleware())

	auth.GET("/inventories", s.userHandler.GetUsersInventory)

	// -------- paypal --------
	paypal := auth.Group("/paypal")
	paypal.POST("/pay", s.paypalHandler.Pay)
	paypal.POST("/pay-again", s.paypalHandler.PayAgain)
	paypal.GET("/have-saved-payment", s.paypalHandler.CheckUserHaveSavedPayment)
	paypal.POST("/subscribe", s.paypalHandler.SubscribeSubscription)

	// -------- paypal webhooks / callbacks --------
	paypalPublic := api.Group("/paypal")
	paypalPublic.GET("/success", s.paypalHandler.HandleSuccess)
	paypalPublic.POST("/webhook", s.paypalHandler.PayPalWebhook)
}

func (s *Server) Start(address string) error {
	return s.echo.Start(address)
}

func (s *Server) Shutdown() error {
	return s.echo.Close()
}
