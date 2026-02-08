package handler

import (
	"net/http"
	"paypal-integration-demo/internal/service"

	"github.com/labstack/echo/v4"
)

type UserHandler struct {
	userService service.UserService
}

func NewUserHandler(userService service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

func (h *UserHandler) GetUsersInventory(c echo.Context) error {
	ctx := c.Request().Context()

	inventories, err := h.userService.GetInventories(ctx)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, inventories)
}
