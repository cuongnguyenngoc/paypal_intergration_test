package server

import (
	"paypal-integration-demo/handlers"
	"paypal-integration-demo/services"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	echo          *echo.Echo
	paypalHandler *handlers.PaypalHandler
}

func NewServer(paypalService services.PaypalService) *Server {
	e := echo.New()

	e.File("/", "web/index.html")

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	paypalHandler := handlers.NewPaypalHandler(paypalService)

	s := &Server{
		echo:          e,
		paypalHandler: paypalHandler,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	api := s.echo.Group("/api")

	api.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"status": "ok",
		})
	})

	paypal := api.Group("/paypal")

	paypal.POST("/pay", s.paypalHandler.Pay)
	paypal.GET("/success", s.paypalHandler.HandleSuccess)
	paypal.POST("/webhook", s.paypalHandler.PayPalWebhook)
}

func (s *Server) Start(address string) error {
	return s.echo.Start(address)
}

func (s *Server) Shutdown() error {
	return s.echo.Close()
}
