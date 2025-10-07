package middleware

import (
	"sync"
	"time"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/gofiber/fiber/v2"
)

func (m *AppMiddleware) Logger() fiber.Handler {
	var (
		errPadding = 15
		once       sync.Once
		errHandler fiber.ErrorHandler
	)

	return func(c *fiber.Ctx) error {
		once.Do(func() {
			errHandler = c.App().Config().ErrorHandler
			stack := c.App().Stack()
			for m := range stack {
				for r := range stack[m] {
					if len(stack[m][r].Path) > errPadding {
						errPadding = len(stack[m][r].Path)
					}
				}
			}
		})

		start := time.Now()

		err := c.Next()
		if err != nil {
			if err := errHandler(c, err); err != nil {
				_ = c.SendStatus(fiber.StatusInternalServerError)
			}
		}

		latency := time.Since(start)

		uctx := c.UserContext()
		statusCode := c.Response().StatusCode()

		fields := []xlog.Field{
			xlog.String("latency", latency.String()),
			xlog.Object("request", Req(c)),
			xlog.Object("response", Resp(c.Response())),
		}
		if statusCode < 200 || (statusCode >= 300 && statusCode < 500) || err != nil {
			if err != nil {
				fields = append(fields, xlog.Err(err))
			}
			xlog.Warn(uctx, "[HTTP.REQUEST]", fields...)
		} else if statusCode >= 500 {
			xlog.Error(uctx, "[HTTP.REQUEST]", fields...)
		} else {
			xlog.Info(uctx, "[HTTP.REQUEST]", fields...)
		}

		return nil
	}
}
