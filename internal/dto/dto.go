package dto

type Item struct {
	Sku      string `json:"sku"`
	Quantity int32  `json:"quantity"`
}

type PayRequest struct {
	Items []*Item `json:"items"`
}

type PayResponse struct {
	OrderID          string `json:"order_id"`
	OrderApprovalURL string `json:"order_approval_url"`
}

type SubscribeRequest struct {
	ProductID string `json:"product_id"`
}

type SubscribeResponse struct {
	ApprovalURL string `json:"approval_url"`
}

type CreateMerchantRequest struct {
	MerchantName string `json:"name"`
}

type PaypalConnectRequest struct {
	MerchantID string `json:"merchant_id"`
}
