package handler

import (
	"net/http"
	"paypal-integration-demo/internal/dto"
	"paypal-integration-demo/internal/service"

	"github.com/labstack/echo/v4"
)

type MerchantHandler struct {
	merchantService service.MerchantService
}

func NewMerchantHandler(merchantService service.MerchantService) *MerchantHandler {
	return &MerchantHandler{
		merchantService: merchantService,
	}
}

func (h *MerchantHandler) CreateMerchant(c echo.Context) error {
	ctx := c.Request().Context()

	var req dto.CreateMerchantRequest
	if err := c.Bind(&req); err != nil {
		return err
	}

	err := h.merchantService.CreateMerchant(ctx, req.MerchantName)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "create merchant successfully",
	})
}
