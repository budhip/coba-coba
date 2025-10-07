package middleware

import (
	"net/http"

	xlog "bitbucket.org/Amartha/go-x/log"
	"bitbucket.org/Amartha/go-x/log/ctxdata"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

func (m *AppMiddleware) Context(gcpProjectID string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var req http.Request
		ctx := c.Context()
		if err := fasthttpadaptor.ConvertRequest(ctx, &req, true); err != nil {
			xlog.Warn(ctx, "error converting fasthttp.Request to http.Request", xlog.Err(err))
		}
		c.SetUserContext(ctxdata.SetContextFromHTTP(ctx, &req, gcpProjectID))
		return c.Next()
	}
}
