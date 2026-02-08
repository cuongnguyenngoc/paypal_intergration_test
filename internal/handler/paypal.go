package handler

import (
	"fmt"
	"net/http"
	"paypal-integration-demo/internal/dto"
	"paypal-integration-demo/internal/service"

	"github.com/labstack/echo/v4"
)

type PaypalHandler struct {
	paypalService service.PaypalService
}

func NewPaypalHandler(paypalService service.PaypalService) *PaypalHandler {
	return &PaypalHandler{
		paypalService: paypalService,
	}
}

func (h *PaypalHandler) Pay(c echo.Context) error {
	ctx := c.Request().Context()

	var req dto.PayRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	result, err := h.paypalService.Pay(ctx, req.Email, req.Items)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

func (h *PaypalHandler) HandleSuccess(c echo.Context) error {
	ctx := c.Request().Context()

	orderID := c.QueryParam("token")
	if orderID == "" {
		return c.String(400, "missing order token")
	}
	fmt.Println("orderID", orderID)

	err := h.paypalService.CaptureOrder(ctx, orderID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message": "Payment approved. Processing...",
	})
}

func (h *PaypalHandler) PayPalWebhook(c echo.Context) error {
	var payload map[string]interface{}
	if err := c.Bind(&payload); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	eventType := payload["event_type"].(string)

	switch eventType {
	case "PAYMENT.CAPTURE.COMPLETED":
		// mark order as paid
		fmt.Println("payment completed")
	case "BILLING.SUBSCRIPTION.ACTIVATED":
		// activate subscription
		fmt.Println("subscription activated")
	}

	return c.NoContent(http.StatusOK)
}
