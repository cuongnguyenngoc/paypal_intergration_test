package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"paypal-integration-demo/config"
	"time"
)

type PaypalClient interface {
	Pay(ctx context.Context) (map[string]interface{}, error)
}

type paypalClientImpl struct {
	httpClient         *http.Client
	baseApiURL         string
	paypalClientID     string
	paypalClientSecret string
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

func (c *paypalClientImpl) Pay(ctx context.Context) (map[string]interface{}, error) {
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
			"return_url": "http://localhost:8080/success",
			"cancel_url": "http://localhost:8080/cancel",
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

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Println("result", result)

	return result, nil
}
