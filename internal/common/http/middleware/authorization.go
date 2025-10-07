package middleware

import (
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"github.com/gofiber/fiber/v2"
)

func (m *AppMiddleware) InternalAuth() fiber.Handler {
	return func(c *fiber.Ctx) error {
		secretKey := c.Get("X-Secret-Key")
		statusCode := fiber.StatusUnauthorized
		if secretKey == "" {
			return http.RestErrorResponse(c, statusCode, fmt.Errorf("%s", "required secret key"))
		}

		if secretKey != m.conf.SecretKey {
			return http.RestErrorResponse(c, statusCode, fmt.Errorf("%s", "invalid secret key"))
		}

		return c.Next()
	}
}
