package middleware

import (
	"bitbucket.org/Amartha/go-x/log/ctxdata"

	"github.com/labstack/echo/v4"
)

func (m *AppMiddleware) Context() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			ctx := c.Request().Context()

			// set idempotency to context
			cdata := ctxdata.Sets(
				ctxdata.SetContextFromHTTP(ctx, c.Request(), m.conf.GcloudProjectID),
			)

			c.SetRequest(c.Request().WithContext(cdata))
			return next(c)
		}
	}
}
