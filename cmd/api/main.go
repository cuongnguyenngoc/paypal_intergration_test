package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"paypal-integration-demo/internal/model"
	"time"
)

// Config - Replace with your Sandbox Credentials
const (
	ClientID     = "AdOhTlMi0m4_ssuz3-3bjF4oU_Nv2Ekh-QbCXr9THeK91splN2PKUPQM1xr9rsHOTS5HloMq_NNiPnHH" // Get this from your PayPal Developer Dashboard
	ClientSecret = "EFyiRDAxAWw88iV79KvfU-8ZGI_LrRKaWF-D6bepIbPQ0_6Dp1VpUAMmuta2k75ZDudCyQoAj1BrGSgo"
	BaseURL      = "https://api-m.sandbox.paypal.com" // Use live URL for production
	// PlanID       = "P-1D059697H5353731UNGHLL4Q"       // Create this in PayPal Dashboard first
)

// HTTP Client with timeout
var client = &http.Client{Timeout: 10 * time.Second}

func main() {
	// 1. Get Access Token
	accessToken, err := getAccessToken()
	if err != nil {
		log.Fatalf("Error getting access token: %v", err)
	}
	fmt.Println("✅ Access Token acquired")

	// 2. Create Setup Token (To start the vaulting process)
	// setupTokenID, approvalURL, err := createSetupToken(accessToken)
	// if err != nil {
	// 	log.Fatalf("Error creating setup token: %v", err)
	// }

	setupTokenID, approvalURL, err := createPayPalSetupToken(accessToken)
	if err != nil {
		log.Fatalf("Error creating PayPal setup token: %v", err)
	}

	fmt.Printf("\n🔗 Please approve the vaulting by visiting this URL:\n%s\n", approvalURL)
	fmt.Println("\n(In a real app, your frontend would handle this approval step and send the Setup Token ID back to your server.)")
	fmt.Println("Press ENTER once you have approved the link in your browser...")
	fmt.Scanln()

	// fmt.Printf("\n🔴 ACTION REQUIRED: Open this URL to approve the vaulting:\n%s\n", approvalLink)
	// fmt.Println("\n(In a real app, your frontend handles the approval and sends the Setup Token ID back to your server.)")
	// fmt.Println("Press ENTER once you have approved the link in your browser...")
	// fmt.Scanln()

	// 3. Exchange Setup Token for Payment Token (Permanent Vault Token)
	paymentToken, err := createPaymentToken(accessToken, setupTokenID)
	if err != nil {
		log.Fatalf("Error creating payment token: %v", err)
	}
	fmt.Printf("✅ Payment Token Vaulted: %s\n", paymentToken)

	newPlanID, err := setupNewPlan(accessToken)
	if err != nil {
		log.Fatalf("Error setting up plan: %v", err)
	}

	// err = activatePlan(accessToken, newPlanID)
	// if err != nil {
	// 	log.Fatalf("Error activating plan: %v", err)
	// }

	// 4. Create Subscription using the Vaulted Token
	subID, err := createSubscription(accessToken, paymentToken, newPlanID)
	if err != nil {
		log.Fatalf("Error creating subscription: %v", err)
	}

	fmt.Printf("\n🎉 SUCCESS! Subscription created with Vaulted Card.\nSubscription ID: %s\n", subID)
}

// --- Helper Functions ---

// getAccessToken gets the Oauth2 Bearer token
func getAccessToken() (string, error) {
	req, _ := http.NewRequest("POST", BaseURL+"/v1/oauth2/token", bytes.NewBufferString("grant_type=client_credentials&response_type=token"))
	auth := base64.StdEncoding.EncodeToString([]byte(ClientID + ":" + ClientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.AccessToken, nil
}

// createSetupToken initializes the vaulting request (V3)
func createSetupToken(accessToken string) (string, string, error) {
	// Request body for a Credit Card Setup Token
	payload := map[string]interface{}{
		"payment_source": map[string]interface{}{
			"card": map[string]interface{}{
				"number":        "4111111111111111", // Visa Test Card
				"expiry":        "2030-12",
				"security_code": "123",
				"name":          "Go Developer",
				// Billing address is often required for ACDC
				"billing_address": map[string]interface{}{
					"address_line_1": "123 Main St",
					"admin_area_2":   "San Jose",
					"admin_area_1":   "CA",
					"postal_code":    "95131",
					"country_code":   "US",
				},
			},
		},
		"usage_type": "PLATFORM",
		"experience_context": map[string]interface{}{
			"brand_name": "My Go App",
			"locale":     "en-US",
			"return_url": "https://example.com/return", // CRITICAL: Required for 'approve' link
			"cancel_url": "https://example.com/cancel", // CRITICAL: Required for 'approve' link
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", BaseURL+"/v3/vault/setup-tokens", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	// v3 often requires a Request-ID for idempotency
	req.Header.Set("PayPal-Request-Id", fmt.Sprintf("req-%d", time.Now().UnixNano()))

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("paypal setup token error: %s", b)
	}

	// Parse response to get ID and Approval Link
	var result struct {
		ID    string `json:"id"`
		Links []struct {
			Rel  string `json:"rel"`
			Href string `json:"href"`
		} `json:"links"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}
	fmt.Println("result", result)

	var approvalURL string
	for _, link := range result.Links {
		if link.Rel == "approve" {
			approvalURL = link.Href
		}
	}

	return result.ID, approvalURL, nil
}

// createPaymentToken turns an approved Setup Token into a permanent Payment Token
func createPaymentToken(accessToken, setupTokenID string) (string, error) {
	payload := map[string]interface{}{
		"payment_source": map[string]interface{}{
			"token": map[string]interface{}{
				"id":   setupTokenID,
				"type": "SETUP_TOKEN",
			},
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", BaseURL+"/v3/vault/payment-tokens", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PayPal-Request-Id", fmt.Sprintf("pt-%d", time.Now().UnixNano()))

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API Error: %s", string(bodyBytes))
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.ID, nil
}

func setupNewPlan(accessToken string) (string, error) {
	// 1. Create a Product (Required for a Plan)
	productPayload := map[string]interface{}{
		"name":        "Go Vault Service",
		"description": "Subscription for Vaulted Card",
		"type":        "SERVICE",
		"category":    "SOFTWARE",
	}
	prodBody, _ := json.Marshal(productPayload)

	req, _ := http.NewRequest("POST", BaseURL+"/v1/catalogs/products", bytes.NewBuffer(prodBody))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PayPal-Request-Id", fmt.Sprintf("prod-%d", time.Now().UnixNano()))

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var prodResult struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prodResult); err != nil {
		return "", err
	}
	fmt.Printf("1. Product Created: %s\n", prodResult.ID)

	// 2. Create the Plan
	planPayload := map[string]interface{}{
		"product_id":  prodResult.ID,
		"name":        "Monthly Subscription",
		"description": "Changed $10 every month",
		"status":      "ACTIVE", // Try setting ACTIVE directly (works in some regions)
		"billing_cycles": []map[string]interface{}{
			{
				"frequency": map[string]interface{}{
					"interval_unit":  "MONTH",
					"interval_count": 1,
				},
				"tenure_type":  "REGULAR",
				"sequence":     1,
				"total_cycles": 0, // 0 = Infinite
				"pricing_scheme": map[string]interface{}{
					"fixed_price": map[string]interface{}{
						"value":         "10.00",
						"currency_code": "USD",
					},
				},
			},
		},
		"payment_preferences": map[string]interface{}{
			"auto_bill_outstanding": true,
			"setup_fee": map[string]interface{}{
				"value":         "0",
				"currency_code": "USD",
			},
			"setup_fee_failure_action":  "CONTINUE",
			"payment_failure_threshold": 3,
		},
	}
	planBody, _ := json.Marshal(planPayload)

	reqPlan, _ := http.NewRequest("POST", BaseURL+"/v1/billing/plans", bytes.NewBuffer(planBody))
	reqPlan.Header.Set("Authorization", "Bearer "+accessToken)
	reqPlan.Header.Set("Content-Type", "application/json")
	reqPlan.Header.Set("PayPal-Request-Id", fmt.Sprintf("plan-%d", time.Now().UnixNano()))

	respPlan, err := client.Do(reqPlan)
	if err != nil {
		return "", err
	}
	defer respPlan.Body.Close()

	// Read body to check for errors
	planBytes, _ := io.ReadAll(respPlan.Body)
	if respPlan.StatusCode >= 300 {
		return "", fmt.Errorf("plan creation failed: %s", string(planBytes))
	}

	var planResult struct {
		ID string `json:"id"`
	}
	json.Unmarshal(planBytes, &planResult)
	fmt.Printf("2. Plan Created: %s\n", planResult.ID)

	return planResult.ID, nil
}

func activatePlan(accessToken, planID string) error {
	// There is no body, just a POST to the /activate endpoint
	req, _ := http.NewRequest("POST", BaseURL+"/v1/billing/plans/"+planID+"/activate", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 && resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to activate plan: %s", string(b))
	}

	fmt.Println("Plan activated successfully!")
	return nil
}

// createSubscription creates a subscription using the Payment Token
func createSubscription(accessToken, paymentToken, planID string) (string, error) {
	payload := map[string]interface{}{
		"plan_id": planID,
		"subscriber": map[string]interface{}{
			"name": map[string]interface{}{
				"given_name": "John",
				"surname":    "Doe",
			},
			"email_address": "john.doe@example.com",
			"payment_source": map[string]interface{}{
				"card": map[string]interface{}{
					"number":        "4111111111111111", // Visa Test Card
					"expiry":        "2030-12",
					"security_code": "123",
					"name":          "Go Developer",
					// Billing address is often required for ACDC
					"billing_address": map[string]interface{}{
						"address_line_1": "123 Main St",
						"admin_area_2":   "San Jose",
						"admin_area_1":   "CA",
						"postal_code":    "95131",
						"country_code":   "US",
					},
				},
			},
		},

		"application_context": map[string]interface{}{
			"return_url": "https://example.com/return",
			"cancel_url": "https://example.com/cancel",
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", BaseURL+"/v1/billing/subscriptions", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API Error: %s", string(bodyBytes))
	}

	var result model.PaypalResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	fmt.Println("Subscription Creation Result:", result)
	return result.ID, nil
}

func createPayPalSetupToken(accessToken string) (string, string, error) {
	// 1. Change Source to "paypal"
	payload := map[string]interface{}{
		"payment_source": map[string]interface{}{
			"paypal": map[string]interface{}{
				"usage_type": "MERCHANT", // or "MERCHANT"
				"experience_context": map[string]interface{}{
					// "brand_name": "My Go App",
					// "locale":     "en-US",
					"return_url": "https://your-domain.com/callback", // User comes here after approving
					"cancel_url": "https://your-domain.com/cancel",
				},
			},
		},
	}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", BaseURL+"/v3/vault/setup-tokens", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PayPal-Request-Id", fmt.Sprintf("req-%d", time.Now().UnixNano()))

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	// Parse Response
	var result struct {
		ID    string `json:"id"`
		Links []struct {
			Rel  string `json:"rel"`
			Href string `json:"href"`
		} `json:"links"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	fmt.Println("Setup Token Result:", result)

	// 2. Extract the "approve" link
	var approvalURL string
	for _, link := range result.Links {
		if link.Rel == "approve" {
			approvalURL = link.Href
		}
	}

	if approvalURL == "" {
		return "", "", fmt.Errorf("no approval url found (check your scope/permissions)")
	}

	// You now have the URL! Redirect your user here.
	return result.ID, approvalURL, nil
}
