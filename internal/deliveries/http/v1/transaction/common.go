package transaction

import (
	"errors"
	nethttp "net/http"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
)

// getHTTPStatusCode will return http status code based on error
func getHTTPStatusCode(err error) int {
	if err == nil {
		return nethttp.StatusOK
	}

	if errors.Is(err, common.ErrDataTrxDuplicate) {
		return nethttp.StatusConflict
	}

	if errors.Is(err, common.ErrInvalidAmount) ||
		errors.Is(err, common.ErrInsufficientAvailableBalance) ||
		errors.Is(err, common.ErrInsufficientPendingBalance) {
		return nethttp.StatusBadRequest
	}

	if errors.Is(err, common.ErrInvalidOrderType) ||
		errors.Is(err, common.ErrInvalidTransactionType) {
		return nethttp.StatusUnprocessableEntity
	}

	return nethttp.StatusInternalServerError
}
