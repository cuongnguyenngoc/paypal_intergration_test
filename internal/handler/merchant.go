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

	merchantID, err := h.merchantService.CreateMerchant(ctx, req.MerchantName)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]string{
		"id": merchantID,
	})
}

func (h *MerchantHandler) PayPalStatus(c echo.Context) error {
	ctx := c.Request().Context()

	merchantID := c.Param("id")

	merchant, err := h.merchantService.GetMerchant(ctx, merchantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	return c.JSON(http.StatusOK, map[string]bool{
		"connected": merchant.PayPalAccessToken != "",
	})
}

func (h *MerchantHandler) DisconnectPayPal(c echo.Context) error {
	ctx := c.Request().Context()

	merchantID := c.Param("id")

	err := h.merchantService.DisconnectPayPal(ctx, merchantID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, map[string]string{
		"status": "disconnected",
	})
}
