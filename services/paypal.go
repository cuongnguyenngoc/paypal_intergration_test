package services

import "paypal-integration-demo/client"

type PaypalService interface {
	GetAccessToken() (string, error)
}

type paypalServiceImpl struct {
	paypalClient client.PaypalClient
}

func NewPaypalService(paypalClient client.PaypalClient) PaypalService {
	return &paypalServiceImpl{
		paypalClient: paypalClient,
	}
}

func (s *paypalServiceImpl) GetAccessToken() (string, error) {
	return s.paypalClient.GetAccessToken()
}
