package server

import (
	"paypal-integration-demo/internal/handler"
	"paypal-integration-demo/internal/service"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	echo          *echo.Echo
	paypalHandler *handler.PaypalHandler
}

func NewServer(paypalService service.PaypalService) *Server {
	e := echo.New()

	e.File("/", "../../web/index.html")

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	paypalHandler := handler.NewPaypalHandler(paypalService)

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
