package handlers

import (
	"net/http"
	"paypal-integration-demo/services"

	"github.com/labstack/echo/v4"
)

type PaypalHandler struct {
	paypalService services.PaypalService
}

func NewPaypalHandler(paypalService services.PaypalService) *PaypalHandler {
	return &PaypalHandler{
		paypalService: paypalService,
	}
}

type PayRequest struct {
	Email string `json:"email"`
	Items []struct {
		Type     string  `json:"type"`
		Price    float64 `json:"price"`
		Quantity int     `json:"quantity"`
	} `json:"items"`
}

func (h *PaypalHandler) Pay(c echo.Context) error {
	// ctx := c.Request().Context()

	var req PayRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message": "Pay success",
	})
}
