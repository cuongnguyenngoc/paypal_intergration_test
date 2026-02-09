package dto

type Item struct {
	Sku      string `json:"sku"`
	Quantity int32  `json:"quantity"`
}

type PayRequest struct {
	Items []*Item `json:"items"`
	Vault bool    `json:"vault"`
}

type PayResponse struct {
	OrderID          string `json:"order_id"`
	OrderApprovalURL string `json:"order_approval_url"`
}
