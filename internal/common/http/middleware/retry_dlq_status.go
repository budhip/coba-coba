package middleware

import (
	"encoding/json"
	"fmt"

	xlog "bitbucket.org/Amartha/go-x/log"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/exp/slices"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
)

func (m *AppMiddleware) CheckRetryDLQ() fiber.Handler {
	return func(c *fiber.Ctx) error {
		dlqProcessId := c.Get("X-DLQ-Process-Id")
		if dlqProcessId == "" {
			return c.Next()
		}

		status, err := m.dlqProcessor.GetStatusRetry(c.Context(), dlqProcessId)
		if err != nil {
			return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
		}

		// process request first
		err = c.Next()
		if err != nil {
			return err
		}

		if c.Response().StatusCode() >= 200 && c.Response().StatusCode() < 300 {
			// if process success do nothing
			return nil
		}

		status.CurrentRetry += 1

		maxRetryReached := status.CurrentRetry > status.MaxRetry
		willRetryAgain := slices.Contains([]int{408, 504, 503, 500}, c.Response().StatusCode())

		if maxRetryReached || !willRetryAgain {
			message := fmt.Sprintf("max retry reached or status code not retryable: %d", c.Response().StatusCode())

			errMsg := getErrorMessageFromResponse(c.Response().Body())
			if errMsg != "" {
				message = errMsg
			}

			message += "\n\n Process Id: " + dlqProcessId

			err = m.dlqProcessor.SendNotificationRetryFailure(c.Context(), status.ProcessName, message)
			if err != nil {
				xlog.Warn(c.Context(), "failed to send notification retry failure", xlog.Err(err))
			}

			return nil
		}

		err = m.dlqProcessor.UpsertStatusRetry(c.Context(), dlqProcessId, status)
		if err != nil {
			xlog.Warn(c.Context(), "failed to update status retry dlq", xlog.Err(err))
		}

		return nil
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
