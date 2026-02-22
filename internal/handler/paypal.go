package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"paypal-integration-demo/internal/dto"
	"paypal-integration-demo/internal/service"

	"github.com/labstack/echo/v4"
)

// for demo purpose: user who receive items from merchant
const userID = "demo-user-001"

type PaypalHandler struct {
	paypalService   service.PaypalService
	merchantService service.MerchantService
}

func NewPaypalHandler(paypalService service.PaypalService, merchantService service.MerchantService) *PaypalHandler {
	return &PaypalHandler{
		paypalService:   paypalService,
		merchantService: merchantService,
	}
}

func merchantIDFromHeader(c echo.Context) (string, error) {
	merchantID := c.Request().Header.Get("X-Merchant-Id")
	if merchantID == "" {
		return "", echo.NewHTTPError(http.StatusBadRequest, "missing X-Merchant-Id header")
	}
	return merchantID, nil
}

func (h *PaypalHandler) ConnectMerchant(c echo.Context) error {
	merchantID := c.Param("merchantID")

	url := h.paypalService.Connect(merchantID)

	return c.Redirect(http.StatusFound, url)
}

func (h *PaypalHandler) OAuthCallback(c echo.Context) error {
	ctx := c.Request().Context()
	code := c.QueryParam("code")
	merchantID := c.QueryParam("state")

	if code == "" || merchantID == "" {
		return c.String(http.StatusBadRequest, "invalid oauth callback")
	}

	token, err := h.paypalService.ExchangeAuthCode(ctx, code)
	if err != nil {
		return err
	}

	err = h.merchantService.UpdatePaypalTokens(ctx, merchantID, token)
	if err != nil {
		return err
	}

	err = h.paypalService.SetExistingProductsSubPlan(ctx, merchantID, token.AccessToken) // silent setup subscription products for merchant when they connect to their paypal business account
	if err != nil {
		log.Println("set existing plans for merchant: %w", err)
	}
	return c.String(http.StatusOK, "PayPal connected successfully")
}

func (h *PaypalHandler) Pay(c echo.Context) error {
	ctx := c.Request().Context()

	merchantID, err := merchantIDFromHeader(c)
	if err != nil {
		return err
	}

	var req dto.PayRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid req body")
	}

	result, err := h.paypalService.Pay(ctx, merchantID, userID, req.Items)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

func (h *PaypalHandler) PayAgain(c echo.Context) error {
	ctx := c.Request().Context()

	merchantID, err := merchantIDFromHeader(c)
	if err != nil {
		return err
	}

	var req dto.PayRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	result, err := h.paypalService.PayAgain(ctx, merchantID, userID, req.Items)
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
		<p>We are processing your payment and grant items to you if success</p>
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

func (h *PaypalHandler) CheckUserHaveSavedPayment(c echo.Context) error {
	ctx := c.Request().Context()

	haveSaved, err := h.paypalService.CheckUserHaveSavedPayment(ctx, userID)
	if err != nil {
		return fmt.Errorf("check user have saved payment: %w", err)
	}

	return c.JSON(http.StatusOK, map[string]bool{
		"has_saved_payment": haveSaved,
	})
}

func (h *PaypalHandler) SubscribeSubscription(c echo.Context) error {
	ctx := c.Request().Context()

	merchantID, err := merchantIDFromHeader(c)
	if err != nil {
		return err
	}

	var req dto.SubscribeRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	approveURL, err := h.paypalService.SubscribeSubscription(ctx, userID, req.ProductID, merchantID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, &dto.SubscribeResponse{
		ApprovalURL: approveURL,
	})
}

func (h *PaypalHandler) HandleSubscriptionSuccess(c echo.Context) error {
	ctx := c.Request().Context()

	subscriptionID := c.QueryParam("subscription_id")
	if subscriptionID == "" {
		return c.String(400, "missing subscription id")
	}

	err := h.paypalService.HandleSubscriptionSuccess(ctx, subscriptionID)
	if err != nil {
		return err
	}

	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<meta charset="utf-8">
		<title>Subscription Processing</title>
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
		<h2>Subscription approved</h2>
		<p>We are processing your product vip_monthly subscription</p>
		<p>Redirecting to homepage in <span class="countdown" id="countdown">5</span> seconds…</p>

		<script>
			let seconds = 5;
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

func (h *PaypalHandler) GetSubscriptionStatus(c echo.Context) error {
	ctx := c.Request().Context()

	merchantID, err := merchantIDFromHeader(c)
	if err != nil {
		return err
	}

	active, err := h.paypalService.HasActiveSubscription(ctx, userID, merchantID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]bool{
		"active": active,
	})
}

func (h *PaypalHandler) CancelSubscription(c echo.Context) error {
	ctx := c.Request().Context()

	merchantID, err := merchantIDFromHeader(c)
	if err != nil {
		return err
	}

	if err := h.paypalService.CancelSubscription(ctx, userID, merchantID); err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "cancelled",
	})
}

func (h *PaypalHandler) ProcessCheckout(c echo.Context) error {
	ctx := c.Request().Context()

	// Parse the JSON body from the frontend
	var req struct {
		Nonce string `json:"nonce"`
	}
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request payload",
		})
	}

	// // Extract the user ID from the token/session
	// userID, _ := c.Get("user_id").(string)

	// Call your service layer to execute the Vault -> Charge -> Subscribe flow
	// (This calls the logic we wrote in the previous step)
	result, err := h.paypalService.BraintreeProcessCheckout(ctx, req.Nonce)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Checkout failed: " + err.Error(),
		})
	}

	// Return success to the frontend
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":         "Checkout Complete!",
		"transaction_id":  result.TransactionID,
		"subscription_id": result.SubscriptionID,
	})
}

func (h *PaypalHandler) ProcessCheckoutWithSavedCard(c echo.Context) error {
	ctx := c.Request().Context()

	result, err := h.paypalService.BraintreeProcessCheckoutWithSavedCard(ctx)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Checkout failed: " + err.Error(),
		})
	}

	// Return success to the frontend
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message":         "Checkout Complete!",
		"transaction_id":  result.TransactionID,
		"subscription_id": result.SubscriptionID,
	})
}
