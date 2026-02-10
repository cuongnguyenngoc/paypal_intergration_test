package model

type Payer struct {
	PayerID string `json:"payer_id"`
	Email   string `json:"email_address"`
}

type PaypalLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

type PaypalResult struct {
	ID     string       `json:"id"`
	Links  []PaypalLink `json:"links"`
	Status string       `json:"status"`
	Payer  Payer        `json:"payer"`
}

type Amount struct {
	Currency string `json:"currency_code"`
	Value    string `json:"value"`
}

type Capture struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	CreateTime string `json:"create_time"`
	Final      bool   `json:"final_capture"`
	Amount     Amount `json:"amount"`
}

type Payments struct {
	Captures []Capture `json:"captures"`
}

type PurchaseUnit struct {
	ReferenceID string   `json:"reference_id"`
	Payments    Payments `json:"payments"`
}

type RelatedIDs struct {
	OrderID string `json:"order_id"`
}

type SupplementaryData struct {
	RelatedIDs RelatedIDs `json:"related_ids"`
}

type PaymentSource struct {
	PayPal Payer `json:"paypal"`
}

type PayPalMetadata struct {
	OrderID string `json:"order_id"`
}

type PaypalResource struct {
	ID                string            `json:"id"`
	Intent            string            `json:"intent"`
	Status            string            `json:"status"`
	Payer             Payer             `json:"payer"`
	PurchaseUnits     []PurchaseUnit    `json:"purchase_units"`
	SupplementaryData SupplementaryData `json:"supplementary_data"`

	// Vault-specific
	Metadata        PayPalMetadata `json:"metadata"`
	PaymentResource PaymentSource  `json:"payment_source"`
}

type PayPalWebhookEvent struct {
	ID         string         `json:"id"`
	EventType  string         `json:"event_type"`
	CreateTime string         `json:"create_time"`
	Resource   PaypalResource `json:"resource"`
}
