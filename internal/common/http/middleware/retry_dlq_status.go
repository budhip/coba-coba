package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"

	httpUtil "bitbucket.org/Amartha/go-fp-transaction/internal/common/http"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/labstack/echo/v4"
	"golang.org/x/exp/slices"
)

func (m *AppMiddleware) CheckRetryDLQ() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			dlqProcessId := c.Request().Header.Get("X-DLQ-Process-Id")
			if dlqProcessId == "" {
				return next(c)
			}

			status, err := m.dlqProcessor.GetStatusRetry(c.Request().Context(), dlqProcessId)
			if err != nil {
				return httpUtil.RestErrorResponse(c, http.StatusInternalServerError, err)
			}

			// process request first
			err = next(c)
			if err != nil {
				return err
			}

			if c.Response().Status >= 200 && c.Response().Status < 300 {
				// if process success do nothing
				return nil
			}

			status.CurrentRetry += 1

			maxRetryReached := status.CurrentRetry > status.MaxRetry
			willRetryAgain := slices.Contains([]int{408, 504, 503, 500}, c.Response().Status)

			if maxRetryReached || !willRetryAgain {
				message := fmt.Sprintf("max retry reached or status code not retryable: %d", c.Response().Status)

				errMsg := getErrorMessageFromResponse(m.getResponseBodyBuffer(c).Bytes())
				if errMsg != "" {
					message = errMsg
				}

				message += "\n\n Process Id: " + dlqProcessId

				err = m.dlqProcessor.SendNotificationRetryFailure(c.Request().Context(), status.ProcessName, message)
				if err != nil {
					xlog.Warn(c.Request().Context(), "failed to send notification retry failure", xlog.Err(err))
				}

				return nil
			}

			err = m.dlqProcessor.UpsertStatusRetry(c.Request().Context(), dlqProcessId, status)
			if err != nil {
				xlog.Warn(c.Request().Context(), "failed to update status retry dlq", xlog.Err(err))
			}

			return nil
		}
	}
}

type errorResponse struct {
	Message string `json:"message"`
}

func getErrorMessageFromResponse(res []byte) string {
	var errRes errorResponse
	err := json.Unmarshal(res, &errRes)
	if err != nil {
		return ""
	}

	return errRes.Message
}
