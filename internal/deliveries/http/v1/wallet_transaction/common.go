package wallettrx

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
)

func getHttpErrorStatusCode(err error) int {
	if strings.Contains(err.Error(), "validation") ||
		errors.Is(err, common.ErrInvalidAmount) ||
		errors.Is(err, common.ErrMissingDescription) ||
		errors.Is(err, common.ErrMissingDestinationAccountNumber) ||
		errors.Is(err, common.ErrMissingCustomerNumberFromMetadata) ||
		errors.Is(err, common.ErrMissingDisbursementDateFromMetadata) ||
		errors.Is(err, common.ErrMissingVirtualAccountPointFromMetadata) ||
		errors.Is(err, common.ErrMissingAgreementNumberFromMetadata) ||
		errors.Is(err, common.ErrMissingRepaymentDateFromMetadata) ||
		errors.Is(err, common.ErrMissingEntityFromMetadata) ||
		errors.Is(err, common.ErrMissingPartnerPPOBFromMetadata) ||
		errors.Is(err, common.ErrMissingLoanAccountNumberFromMetadata) ||
		errors.Is(err, common.ErrMissingOldLoanAccountNumberFromMetadata) ||
		errors.Is(err, common.ErrMissingNewLoanAccountNumberFromMetadata) ||
		errors.Is(err, common.ErrMissingProductTypeFromMetadata) ||
		errors.Is(err, common.ErrMissingLoanTypeFromMetadata) ||
		errors.Is(err, common.ErrMissingLoanIdsFromMetadata) ||
		errors.Is(err, common.ErrInvalidLoanIdsTypeMetadata) ||
		errors.Is(err, common.ErrMissingDebitFromMetadata) ||
		errors.Is(err, common.ErrMissingCreditFromMetadata) ||
		errors.Is(err, common.ErrUnsupportedDescription) ||
		errors.Is(err, common.ErrAccountNotExists) ||
		errors.Is(err, common.ErrMissingWalletTransactionIdFromMetadata) {
		return fiber.StatusBadRequest
	}

	if errors.Is(err, common.ErrUnsupportedReservedTransactionFlow) ||
		errors.Is(err, common.ErrNegativeBalanceReached) ||
		errors.Is(err, common.ErrInsufficientAvailableBalance) ||
		errors.Is(err, common.ErrInsufficientPendingBalance) ||
		errors.Is(err, common.ErrConfigAccountNumberNotFound) ||
		errors.Is(err, common.ErrUnableGetTransformer) ||
		errors.Is(err, common.ErrAccountNumberNotFoundInAccounting) ||
		errors.Is(err, common.ErrInvestedAccountNumberNotFound) ||
		errors.Is(err, common.ErrReceivableAccountNumberNotFound) ||
		errors.Is(err, common.ErrrefNumberNotFound) ||
		errors.Is(err, common.ErrUnsupportedTransactionFlow) ||
		errors.Is(err, common.ErrInvalidRefundData) {
		return fiber.StatusUnprocessableEntity
	}

	return fiber.StatusInternalServerError
}
