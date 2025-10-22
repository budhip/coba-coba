package middleware

import (
	"fmt"
	"net/http"

	httpUtil "bitbucket.org/Amartha/go-fp-transaction/internal/common/http"

	"github.com/labstack/echo/v4"
)

func (m *AppMiddleware) InternalAuth(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		secretKey := c.Request().Header.Get("X-Secret-Key")
		statusCode := http.StatusUnauthorized
		if secretKey == "" {
			return httpUtil.RestErrorResponse(c, statusCode, fmt.Errorf("%s", "required secret key"))
		}

		if secretKey != m.conf.SecretKey {
			return httpUtil.RestErrorResponse(c, statusCode, fmt.Errorf("%s", "invalid secret key"))
		}

		return next(c)
	}
}
