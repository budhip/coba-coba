package transaction

import (
	"errors"

	"github.com/gofiber/fiber/v2"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
)

// getHTTPStatusCode will return http status code based on error
func getHTTPStatusCode(err error) int {
	if err == nil {
		return fiber.StatusOK
	}

	if errors.Is(err, common.ErrDataTrxDuplicate) {
		return fiber.StatusConflict
	}

	if errors.Is(err, common.ErrInvalidAmount) ||
		errors.Is(err, common.ErrInsufficientAvailableBalance) ||
		errors.Is(err, common.ErrInsufficientPendingBalance) {
		return fiber.StatusBadRequest
	}

	if errors.Is(err, common.ErrInvalidOrderType) ||
		errors.Is(err, common.ErrInvalidTransactionType) {
		return fiber.StatusUnprocessableEntity
	}

	return fiber.StatusInternalServerError
}
