package handler

import (
	"fmt"
	"net/http"
	"paypal-integration-demo/internal/dto"
	"paypal-integration-demo/internal/model"
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
	ctx := c.Request().Context()

	var event model.PayPalWebhookEvent
	if err := c.Bind(&event); err != nil {
		return fmt.Errorf("decode webhook event payload: %w", err)
	}

	err := h.paypalService.HandleWebhook(ctx, &event)
	if err != nil {
		return fmt.Errorf("handle webhook: %w", err)
	}

	return c.NoContent(http.StatusOK)
}
