package constants

// Money Flow Status
const (
	MoneyFlowStatusPending    = "PENDING"
	MoneyFlowStatusInProgress = "IN_PROGRESS"
	MoneyFlowStatusSuccess    = "SUCCESS"
	MoneyFlowStatusFailed     = "FAILED"
	MoneyFlowStatusRejected   = "REJECTED"
)

// Log Prefixes
const (
	LogPrefixKafkaConsumer      = "[KAFKA-CONSUMER] [MONEY-FLOW-CALC] "
	LogPrefixProcessMessage     = "[PROCESS-MESSAGE]"
	LogPrefixMoneyFlowCalc      = "[MONEY-FLOW-CALC]"
	LogPrefixMoneyFlowUpdate    = "[MONEY-FLOW-UPDATE]"
	LogPrefixMoneyFlowProcessor = "[MONEY-FLOW-PROCESSOR]"
)

// Error Messages
const (
	ErrMsgUnmarshalJSON            = "error unmarshal json to raw: %w"
	ErrMsgUnmarshalAcuanData       = "error unmarshal acuan data: %w"
	ErrMsgProcessNotification      = "unable to process transaction notification: %w"
	ErrMsgAtLeastOneField          = "at least one field must be provided for update"
	ErrMsgInvalidStatusTransition  = "status transition from %s to %s is not allowed"
	ErrMsgPapaIDRequired           = "papaTransactionId is required when status is IN_PROGRESS"
	ErrMsgActualDateNotAllowed     = "actualDate should not be provided when status is IN_PROGRESS"
	ErrMsgInvalidStatus            = "invalid moneyFlowStatus: must be PENDING, SUCCESS, IN_PROGRESS, REJECTED or FAILED"
	ErrMsgPendingTransactionBefore = "cannot transition to IN_PROGRESS: there are PENDING transactions with the same payment type (%s) and transaction type (%s) from earlier dates that must be processed first"
)

// Ineligible Transaction Patterns
var IneligibleTransactionPatterns = []string{
	"payment type not found",
	"transaction type not found",
	"not eligible",
	"skipping ineligible",
}

// Date Formats
const (
	DateFormatYYYYMMDD     = "2006-01-02"
	DefaultDatePlaceholder = "-"
)

// Money Flow
const (
	MoneyFlowReferencePrefix = "MF-"
	MoneyFlowKind            = "moneyFlowCalc"
)

// CSV Headers
var CSVHeaders = []string{
	"transactionDate",
	"transactionId",
	"reffNumb",
	"typeTransaction",
	"fromAccount",
	"toAccount",
	"amount",
	"description",
	"metadata",
}
