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
	"time"
)

type PaypalClient interface {
	CreateOrder(ctx context.Context, serviceBaseUrl string) (*CreateOrderResponse, error)
	CaptureOrder(ctx context.Context, orderID string) error
}

type paypalClientImpl struct {
	httpClient         *http.Client
	baseApiURL         string
	paypalClientID     string
	paypalClientSecret string
}

type PaypalLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

type PaypalCreateOrderResult struct {
	ID     string       `json:"id"`
	Links  []PaypalLink `json:"links"`
	Status string       `json:"status"`
}

type CreateOrderResponse struct {
	OrderID    string
	ApproveURL string
}

func NewPaypalClient(paypalCfg *config.Paypal) PaypalClient {
	return &paypalClientImpl{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseApiURL:         paypalCfg.BaseApiURL,
		paypalClientID:     paypalCfg.ClientID,
		paypalClientSecret: paypalCfg.ClientSecret,
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

func (c *paypalClientImpl) CreateOrder(ctx context.Context, serviceBaseUrl string) (*CreateOrderResponse, error) {
	accessToken, err := c.getAccessToken()
	if err != nil {
		return nil, fmt.Errorf("get paypal access token: %w", err)
	}

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
		"application_context": map[string]string{
			"return_url": fmt.Sprintf("%s/api/paypal/success", serviceBaseUrl),
			"cancel_url": fmt.Sprintf("%s", serviceBaseUrl), // if user cancel during paypal payment, return to our homepage
		},
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

	var result PaypalCreateOrderResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode paypal response: %w", err)
	}

	json.NewDecoder(resp.Body).Decode(&result)

	approveURL := _extractApproveURL(result.Links)

	return &CreateOrderResponse{
		OrderID:    result.ID,
		ApproveURL: approveURL,
	}, nil
}

func (c *paypalClientImpl) CaptureOrder(ctx context.Context, orderID string) error {
	accessToken, err := c.getAccessToken()
	if err != nil {
		return fmt.Errorf("get paypal access token: %w", err)
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
		return fmt.Errorf("create capture request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("paypal capture request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf(
			"paypal capture failed: status=%d body=%s",
			resp.StatusCode,
			string(body),
		)
	}

	// Optional: decode response if want details
	// For now, success response means capture accepted
	return nil
}

func _extractApproveURL(links []PaypalLink) string {
	for _, link := range links {
		if link.Rel == "approve" {
			return link.Href
		}
	}
	return ""
}
