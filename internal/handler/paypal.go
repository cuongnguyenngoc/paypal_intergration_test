package handler

import (
	"fmt"
	"io"
	"net/http"
	"paypal-integration-demo/internal/dto"
	"paypal-integration-demo/internal/service"

	"github.com/labstack/echo/v4"
)

const userID = "RR837NYEPXP36"

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

	result, err := h.paypalService.Pay(ctx, req.Items)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

func (h *PaypalHandler) Payagain(c echo.Context) error {
	ctx := c.Request().Context()

	var req dto.PayRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	result, err := h.paypalService.PayAgain(ctx, userID, req.Items)
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

	err := h.paypalService.CaptureOrder(ctx, orderID)
	if err != nil {
		return err
	}

	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="utf-8">
		<title>Payment Processing</title>
		<style>
			body {
				font-family: Arial, sans-serif;
				text-align: center;
				margin-top: 80px;
			}
			.countdown {
				font-size: 24px;
				font-weight: bold;
			}
		</style>
	</head>
	<body>
		<h2>Payment approved</h2>
		<p>We are processing your payment.</p>
		<p>Redirecting to homepage in <span class="countdown" id="countdown">15</span> seconds…</p>

		<script>
			let seconds = 15;
			const el = document.getElementById("countdown");

			const timer = setInterval(function () {
				seconds--;
				el.textContent = seconds;

				if (seconds <= 0) {
					clearInterval(timer);
					window.location.href = "/";
				}
			}, 1000);
		</script>
	</body>
	</html>
	`

	return c.HTML(http.StatusOK, html)
}

func (h *PaypalHandler) PayPalWebhook(c echo.Context) error {
	ctx := c.Request().Context()

	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	err = h.paypalService.HandleWebhook(ctx, c.Request().Header, body)
	if err != nil {
		return fmt.Errorf("handle webhook: %w", err)
	}

	return c.NoContent(http.StatusOK)
}
