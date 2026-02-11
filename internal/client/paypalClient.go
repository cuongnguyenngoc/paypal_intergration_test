package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"paypal-integration-demo/internal/config"
	"paypal-integration-demo/internal/model"
	"strings"
	"time"

	"github.com/google/uuid"
)

type PaypalClient interface {
	BuildConnectURL(merchantID string) string
	ExchangeAuthCode(ctx context.Context, code string) (*model.PayPalToken, error)
	RefreshMerchantToken(ctx context.Context, refreshToken string) (*model.PayPalToken, error)

	CreateOrderForApproval(ctx context.Context, serviceBaseUrl string, userID string, currency string, cost int32, merchantToken string) (*HandleOrderResponse, error)
	CreateOrderWithVault(ctx context.Context, userID string, vaultID string, currency string, cost int32, merchantToken string) (string, error)
	CaptureOrder(ctx context.Context, orderID string, merchantToken string) (*HandleOrderResponse, error)
	VerifyWebhookSignature(ctx context.Context, headers http.Header, body []byte) error
	CreateUserSubscription(ctx context.Context, serviceBaseUrl string, planID string, userID string, merchantAccessToken string) (subscriptionID string, approveURL string, err error)

	CreateSubscriptionProduct(ctx context.Context, merchantToken string, product *model.Product) (string, error)
	CreateSubscriptionPlan(ctx context.Context, merchantToken string, paypalProductID string, product *model.Product) (string, error)
}

type paypalClientImpl struct {
	httpClient         *http.Client
	baseApiURL         string
	paypalClientID     string
	paypalClientSecret string
	paypalWebhookID    string
	paypalRedirectURL  string
}

type HandleOrderResponse struct {
	OrderID    string
	ApproveURL string
	Status     string
	PayerID    string
}

type ConnectResponse struct {
	RedirectURL string
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
		paypalRedirectURL:  paypalCfg.RedirectURL,
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

func (c *paypalClientImpl) BuildConnectURL(merchantID string) string {
	// scopes := "openid https://uri.paypal.com/services/payments"
	scopes := "openid profile email https://uri.paypal.com/services/paypalattributes"
	return fmt.Sprintf(
		"https://www.sandbox.paypal.com/connect?flowEntry=static&client_id=%s&scope=%s&redirect_uri=%s&state=%s&response_type=code",
		c.paypalClientID,
		url.QueryEscape(scopes),
		url.QueryEscape(c.paypalRedirectURL),
		merchantID,
	)
}

func (c *paypalClientImpl) ExchangeAuthCode(ctx context.Context, code string) (*model.PayPalToken, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)

	req, _ := http.NewRequestWithContext(
		ctx,
		"POST",
		c.baseApiURL+"/v1/oauth2/token",
		strings.NewReader(data.Encode()),
	)

	req.SetBasicAuth(c.paypalClientID, c.paypalClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var token model.PayPalToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
}

func (c *paypalClientImpl) RefreshMerchantToken(ctx context.Context, refreshToken string) (*model.PayPalToken, error) {
	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", refreshToken)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseApiURL+"/v1/oauth2/token",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.paypalClientID, c.paypalClientSecret)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("paypal refresh token failed: %s", b)
	}

	var token model.PayPalToken
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, err
	}

	return &token, nil
}

func (c *paypalClientImpl) CreateOrderForApproval(ctx context.Context, serviceBaseUrl string, userID string, currency string, cost int32, merchantToken string) (*HandleOrderResponse, error) {
	payload := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"custom_id": userID,
				"amount": map[string]string{
					"currency_code": currency,
					"value":         fmt.Sprintf("%.2f", float64(cost)),
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

	result, err := c.createOrder(payload, merchantToken)
	if err != nil {
		return nil, err
	}

	approveURL := _extractApproveURL(result.Links)

	return &HandleOrderResponse{
		OrderID:    result.ID,
		ApproveURL: approveURL,
	}, nil
}

func (c *paypalClientImpl) CreateOrderWithVault(ctx context.Context, userID string, vaultID string, currency string, cost int32, merchantToken string) (string, error) {
	payload := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"custom_id": userID,
				"amount": map[string]string{
					"currency_code": currency,
					"value":         fmt.Sprintf("%.2f", float64(cost)),
				},
			},
		},
		"payment_source": map[string]interface{}{
			"paypal": map[string]string{
				"vault_id": vaultID,
			},
		},
	}

	result, err := c.createOrder(payload, merchantToken)
	if err != nil {
		return "", err
	}

	return result.ID, nil
}

func (c *paypalClientImpl) createOrder(payload map[string]interface{}, merchantToken string) (*model.PaypalResult, error) {
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

	req.Header.Set("Authorization", "Bearer "+merchantToken)
	req.Header.Set("Content-Type", "application/json")

	// REQUIRED for vault charge
	req.Header.Set("PayPal-Request-Id", uuid.NewString())

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

func (c *paypalClientImpl) CaptureOrder(ctx context.Context, orderID string, merchantToken string) (*HandleOrderResponse, error) {
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

	req.Header.Set("Authorization", "Bearer "+merchantToken)
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

	return &HandleOrderResponse{
		OrderID: result.ID,
		PayerID: result.Payer.PayerID,
		Status:  result.Status,
	}, nil
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

func (c *paypalClientImpl) CreateUserSubscription(ctx context.Context, serviceBaseUrl string, planID string, userID string, merchantAccessToken string) (subscriptionID string, approveURL string, err error) {
	payload := map[string]interface{}{
		"plan_id":   planID,
		"custom_id": userID,
		"application_context": map[string]interface{}{
			"user_action": "SUBSCRIBE_NOW",
			"return_url":  fmt.Sprintf("%s/api/paypal/subscription/success", serviceBaseUrl),
			"cancel_url":  serviceBaseUrl,
		},
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseApiURL+"/v1/billing/subscriptions",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return "", "", err
	}

	req.Header.Set("Authorization", "Bearer "+merchantAccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("paypal subscription error: %s", b)
	}

	var result model.PaypalResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	approveURL = _extractApproveURL(result.Links)
	if approveURL == "" {
		return result.ID, "", fmt.Errorf("approve url not found")
	}

	return result.ID, approveURL, nil
}

func (c *paypalClientImpl) CreateSubscriptionProduct(ctx context.Context, merchantToken string, product *model.Product) (string, error) {
	body := map[string]interface{}{
		"name":        product.Name,
		"description": product.Description,
		"type":        "SERVICE", // usually SERVICE for subscriptions
		"category":    "SOFTWARE",
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseApiURL+"/v1/catalogs/products",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+merchantToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("paypal create product failed: %s", string(b))
	}

	var result struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

func (c *paypalClientImpl) CreateSubscriptionPlan(ctx context.Context, merchantToken string, paypalProductID string, product *model.Product) (string, error) {

	body := map[string]interface{}{
		"product_id":  paypalProductID,
		"name":        product.Name,
		"description": product.Description,
		"billing_cycles": []map[string]interface{}{
			{
				"frequency": map[string]interface{}{
					"interval_unit":  "MONTH",
					"interval_count": 1,
				},
				"tenure_type":  "REGULAR",
				"sequence":     1,
				"total_cycles": 0, // 0 = infinite
				"pricing_scheme": map[string]interface{}{
					"fixed_price": map[string]interface{}{
						"value":         fmt.Sprintf("%.2f", float64(product.Price)),
						"currency_code": product.Currency,
					},
				},
			},
		},
		"payment_preferences": map[string]interface{}{
			"auto_bill_outstanding": true,
			"setup_fee": map[string]interface{}{
				"value":         "0",
				"currency_code": product.Currency,
			},
			"setup_fee_failure_action":  "CONTINUE",
			"payment_failure_threshold": 3,
		},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseApiURL+"/v1/billing/plans",
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+merchantToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("paypal create plan failed: %s", string(b))
	}

	var result struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.ID, nil
}

func _extractApproveURL(links []model.PaypalLink) string {
	for _, link := range links {
		if link.Rel == "approve" || link.Rel == "payer-action" {
			return link.Href
		}
	}
	return ""
}
