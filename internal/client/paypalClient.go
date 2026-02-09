package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"paypal-integration-demo/internal/config"
	"paypal-integration-demo/internal/model"
	"time"
)

type PaypalClient interface {
	CreateOrderForApproval(ctx context.Context, serviceBaseUrl string) (*HandleOrderResponse, error)
	CreateOrderWithVault(ctx context.Context, vaultID string) (string, error)
	CaptureOrder(ctx context.Context, orderID string) (*HandleOrderResponse, error)
	GetOrderDetails(ctx context.Context, orderID string) (*GetOrderResponse, error)
	VerifyWebhookSignature(ctx context.Context, headers http.Header, body []byte) error
}

type paypalClientImpl struct {
	httpClient         *http.Client
	baseApiURL         string
	paypalClientID     string
	paypalClientSecret string
	paypalWebhookID    string
}

type HandleOrderResponse struct {
	OrderID    string
	ApproveURL string
	Status     string
	PayerID    string
}

type PaymentSource struct {
	PayPal *PayPalPaymentSource `json:"paypal"`
}

type PayPalPaymentSource struct {
	VaultID string `json:"vault_id"`
	PayerID string `json:"payer_id"`
	Email   string `json:"email_address"`
}

type GetOrderResponse struct {
	ID            string        `json:"id"`
	Status        string        `json:"status"`
	PaymentSource PaymentSource `json:"payment_source"`
}

func NewPaypalClient(paypalCfg *config.Paypal) PaypalClient {
	return &paypalClientImpl{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseApiURL:         paypalCfg.BaseApiURL,
		paypalClientID:     paypalCfg.ClientID,
		paypalClientSecret: paypalCfg.ClientSecret,
		paypalWebhookID:    paypalCfg.WebhookID,
	}
}

func (c *paypalClientImpl) getAccessToken() (string, error) {
	auth := base64.StdEncoding.EncodeToString(
		[]byte(c.paypalClientID + ":" + c.paypalClientSecret),
	)

	req, err := http.NewRequest("POST", c.baseApiURL+"/v1/oauth2/token",
		bytes.NewBufferString("grant_type=client_credentials"))
	if err != nil {
		return "", fmt.Errorf("http new request: %w", err)
	}
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http client do: %w", err)
	}
	defer resp.Body.Close()

	var res struct {
		AccessToken string `json:"access_token"`
	}
	json.NewDecoder(resp.Body).Decode(&res)

	return res.AccessToken, nil
}

func (c *paypalClientImpl) CreateOrderForApproval(ctx context.Context, serviceBaseUrl string) (*HandleOrderResponse, error) {
	payload := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"amount": map[string]string{
					"currency_code": "USD",
					"value":         "100.00",
				},
			},
		},
		"payment_source": map[string]interface{}{
			"paypal": map[string]interface{}{
				"experience_context": map[string]interface{}{
					"return_url":   fmt.Sprintf("%s/api/paypal/success", serviceBaseUrl),
					"cancel_url":   fmt.Sprintf("%s", serviceBaseUrl),
					"landing_page": "LOGIN",
					"user_action":  "PAY_NOW",
				},
				"attributes": map[string]interface{}{
					"vault": map[string]interface{}{
						"store_in_vault": "ON_SUCCESS",
						"usage_type":     "MERCHANT",
						"customer_type":  "CONSUMER",
					},
				},
			},
		},
	}

	result, err := c.createOrder(payload)
	if err != nil {
		return nil, err
	}

	approveURL := _extractApproveURL(result.Links)

	return &HandleOrderResponse{
		OrderID:    result.ID,
		ApproveURL: approveURL,
	}, nil
}

func (c *paypalClientImpl) CreateOrderWithVault(ctx context.Context, vaultID string) (string, error) {
	payload := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"amount": map[string]string{
					"currency_code": "USD",
					"value":         "100.00",
				},
			},
		},
		"payment_source": map[string]interface{}{
			"paypal": map[string]string{
				"vault_id": vaultID,
			},
		},
	}

	result, err := c.createOrder(payload)
	if err != nil {
		return "", err
	}

	return result.ID, nil
}

func (c *paypalClientImpl) createOrder(payload map[string]interface{}) (*model.PaypalResult, error) {
	accessToken, err := c.getAccessToken()
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal req payload: %w", err)
	}
	req, err := http.NewRequest("POST",
		c.baseApiURL+"/v2/checkout/orders",
		bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("http new request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, _ := c.httpClient.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("paypal error %d: %s", resp.StatusCode, string(b))
	}

	var result model.PaypalResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode paypal response: %w", err)
	}

	return &result, nil
}

func (c *paypalClientImpl) CaptureOrder(ctx context.Context, orderID string) (*HandleOrderResponse, error) {
	accessToken, err := c.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get paypal access token: %w", err)
	}

	url := fmt.Sprintf(
		"%s/v2/checkout/orders/%s/capture",
		c.baseApiURL,
		orderID,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		url,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("create capture request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("paypal capture request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf(
			"paypal capture failed: status=%d body=%s",
			resp.StatusCode,
			string(b),
		)
	}

	var result model.PaypalResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode paypal response: %w", err)
	}
	fmt.Println("result", result)

	return &HandleOrderResponse{
		OrderID: result.ID,
		PayerID: result.Payer.PayerID,
		Status:  result.Status,
	}, nil
}

func (c *paypalClientImpl) GetOrderDetails(ctx context.Context, orderID string) (*GetOrderResponse, error) {
	accessToken, err := c.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get paypal access token: %w", err)
	}

	url := fmt.Sprintf("%s/v2/checkout/orders/%s", c.baseApiURL, orderID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create get order request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("paypal capture request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("paypal get orders failed: status=%d body=%s",
			resp.StatusCode,
			string(b),
		)
	}

	var result GetOrderResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode paypal get order response: %w", err)
	}

	return &result, nil
}

func (c *paypalClientImpl) VerifyWebhookSignature(ctx context.Context, headers http.Header, body []byte) error {
	accessToken, err := c.getAccessToken()
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"auth_algo":         headers.Get("PayPal-Auth-Algo"),
		"cert_url":          headers.Get("PayPal-Cert-Url"),
		"transmission_id":   headers.Get("PayPal-Transmission-Id"),
		"transmission_sig":  headers.Get("PayPal-Transmission-Sig"),
		"transmission_time": headers.Get("PayPal-Transmission-Time"),
		"webhook_id":        c.paypalWebhookID,
		"webhook_event":     json.RawMessage(body),
	}

	data, _ := json.Marshal(payload)

	req, _ := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseApiURL+"/v1/notifications/verify-webhook-signature",
		bytes.NewBuffer(data),
	)

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var res struct {
		VerificationStatus string `json:"verification_status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}

	if res.VerificationStatus != "SUCCESS" {
		return fmt.Errorf("invalid paypal webhook signature")
	}

	return nil
}

func _extractApproveURL(links []model.PaypalLink) string {
	for _, link := range links {
		if link.Rel == "approve" || link.Rel == "payer-action" {
			return link.Href
		}
	}
	return ""
}
