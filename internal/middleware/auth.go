package middleware

import "github.com/labstack/echo/v4"

const userID = "demo-user-001"

// sample auth middleware for demo purpose
// later we can expand this to jwt auth or session auth
func AuthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user_id", userID)
			return next(c)
		}
	}
}
