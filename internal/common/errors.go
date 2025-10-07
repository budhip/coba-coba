package common

import (
	"database/sql"
	"errors"
	"fmt"
)

var (
	ErrNoRowsAffected                                 = errors.New("no rows affected")
	ErrValidation                                     = errors.New("validation failed")
	ErrPositionInvalid                                = errors.New("position invalid")
	ErrDataNotFound                                   = errors.New("data not found")
	ErrInternalServerError                            = errors.New("internal server error")
	ErrInvalidFormatDate                              = errors.New("invalid format date")
	ErrDataTrxNotFound                                = errors.New("data transaction not found")
	ErrDataTrxDuplicate                               = errors.New("duplicate transaction found by ref number")
	ErrIDEmpty                                        = errors.New("ID is empty")
	ErrDataExist                                      = errors.New("data exist")
	ErrUnableToCreate                                 = errors.New("unable to create data")
	ErrUnableToUpdate                                 = errors.New("unable to update data")
	ErrUnableToRecon                                  = errors.New("unable to recon")
	ErrFilePathEmpty                                  = errors.New("file path is empty")
	ErrOrderAlreadyExists                             = errors.New("order already exists")
	ErrOrderContainExcludeInsertDB                    = errors.New("order transaction skipped db insert by validation")
	ErrInvalidAmount                                  = errors.New("amount must be greater than zero")
	ErrInsufficientAvailableBalance                   = errors.New("insufficient balance")
	ErrInsufficientPendingBalance                     = errors.New("insufficient balance")
	ErrTransactionNotReserved                         = errors.New("transaction status not reserved")
	ErrInvalidFingerprint                             = errors.New("idempotency key cannot be reused for different requests payload")
	ErrRequestBeingProcessed                          = errors.New("request with same idempotency key is being processed")
	ErrMissingIdempotencyKey                          = errors.New("missing idempotency key. this operation requires idempotency key")
	ErrInvalidPreset                                  = errors.New("invalid preset wallet feature")
	ErrMaxBalanceExceeded                             = errors.New("max balance exceeded")
	ErrInvalidOrderType                               = errors.New("invalid order type")
	ErrInvalidTransactionType                         = errors.New("invalid transaction type")
	ErrInvalidStatus                                  = errors.New("invalid status transaction")
	ErrUnableGetTransformer                           = errors.New("unable to get transformer")
	ErrUnsupportedReservedTransactionFlow             = errors.New("unsupported reserved transaction flow")
	ErrFailedToCreateNotificationPayload              = errors.New("failed to create notification payload")
	ErrInvalidAccountNumber                           = errors.New("invalid account number")
	ErrNegativeBalanceReached                         = errors.New("negative balance reached")
	ErrMissingDescription                             = errors.New("missing description")
	ErrMissingEntityFromAccount                       = errors.New("missing entity from data account")
	ErrMissingEntityFromMetadata                      = errors.New("missing entity from metadata")
	ErrMissingDisbursementDateFromMetadata            = errors.New("missing disbursementDate from metadata")
	ErrMissingVirtualAccountPointFromMetadata         = errors.New("missing virtualAccountPoint from metadata")
	ErrMissingCustomerNumberFromMetadata              = errors.New("missing customerNumber from metadata")
	ErrMissingAgreementNumberFromMetadata             = errors.New("missing agreementNumber from metadata")
	ErrMissingRepaymentDateFromMetadata               = errors.New("missing repaymentDate from metadata")
	ErrMissingPartnerPPOBFromMetadata                 = errors.New("missing partner ppob from metadata")
	ErrMissingLoanAccountNumberFromMetadata           = errors.New("missing loan account number from metadata")
	ErrMissingOldLoanAccountNumberFromMetadata        = errors.New("missing old loan account number from metadata")
	ErrMissingNewLoanAccountNumberFromMetadata        = errors.New("missing new loan account number from metadata")
	ErrNewEntityLoanAccountNumberFromMetadataNotFound = errors.New("new entity loan account number from metadata not found")
	ErrOldEntityLoanAccountNumberFromMetadataNotFound = errors.New("old entity loan account number from metadata not found")
	ErrMissingLoanTypeFromMetadata                    = errors.New("missing loanType from metadata")
	ErrMissingDebitFromMetadata                       = errors.New("missing debit from metadata")
	ErrMissingCreditFromMetadata                      = errors.New("missing credit from metadata")
	ErrInvalidLoanIdsTypeMetadata                     = errors.New("invalid type loanIds metadata")
	ErrMissingLoanIdsFromMetadata                     = errors.New("missing loanIds from metadata")
	ErrConfigAccountNumberNotFound                    = errors.New("config account number not found")
	ErrAccountNumberNotFoundInAccounting              = errors.New("account number not found in accounting")
	ErrInvestedAccountNumberNotFound                  = errors.New("invested account number not found")
	ErrReceivableAccountNumberNotFound                = errors.New("receivable account number not found")
	ErrLoanAdvanceAccountNumberNotFound               = errors.New("loan advance account number not found")
	ErrMissingDestinationAccountNumber                = errors.New("missing destination account number")
	ErrRowLimitDownloadExceed                         = errors.New("row limit download exceed")
	ErrNoRows                                         = sql.ErrNoRows
	ErrUnsupportedDescription                         = errors.New("unsupported description")
	ErrMissingProductTypeFromMetadata                 = errors.New("missing product type from metadata")
	ErrCSVRowIsEmpty                                  = errors.New("csv row is empty")
	ErrAccountNotExists                               = errors.New("account not exists")
	ErrMissingWalletTransactionIdFromMetadata         = errors.New("missing walletTransactionId from metadata")
	ErrrefNumberNotFound                              = errors.New("RefNumber not found")
	ErrUnsupportedTransactionFlow                     = errors.New("transaction flow is not refund")
	ErrInvalidRefundData                              = errors.New("invalid refund transaction data")
)

type WrapError struct {
	Causer interface{}
	Err    error
}

func (e WrapError) Error() string {
	return fmt.Sprintf("%v, root cause: %v", e.Causer, e.Err)
}
