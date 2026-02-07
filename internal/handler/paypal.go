package handler

import (
	"fmt"
	"net/http"
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

type PayRequest struct {
	Email string        `json:"email"`
	Items []*model.Item `json:"items"`
	Vault bool          `json:"vault"`
}

type PayResponse struct {
	OrderID          string `json:"order_id"`
	OrderApprovalURL string `json:"order_approval_url"`
}

func (h *PaypalHandler) Pay(c echo.Context) error {
	ctx := c.Request().Context()

	var req PayRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	result, err := h.paypalService.Pay(ctx, req.Email, req.Items)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &PayResponse{
		OrderID:          result.OrderID,
		OrderApprovalURL: result.ApproveURL,
	})
}

func (h *PaypalHandler) HandleSuccess(c echo.Context) error {
	ctx := c.Request().Context()

	orderID := c.QueryParam("token")
	if orderID == "" {
		return c.String(400, "missing order token")
	}

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
