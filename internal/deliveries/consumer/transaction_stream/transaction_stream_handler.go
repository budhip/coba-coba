package transaction_stream

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	kafkacommon "bitbucket.org/Amartha/go-fp-transaction/internal/common/kafka"
	gopaymentlib "bitbucket.org/Amartha/go-payment-lib/payment-api/models/event"
	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	"bitbucket.org/Amartha/go-x/log/audit"
	"bitbucket.org/Amartha/go-x/log/ctxdata"

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
)

var validStatuses = map[string]bool{
	"SUCCESSFUL": true,
	"REJECTED":   true,
}

type TransactionStreamHandler struct {
	kafkacommon.BaseHandler
	mfs services.MoneyFlowService
	cfg config.Config
}

func NewTransactionStreamHandler(
	clientId string,
	mfs services.MoneyFlowService,
	dlq dlqpublisher.Publisher,
	cfg config.Config,
	consumerMetrics *metrics.ConsumerMetrics,
) sarama.ConsumerGroupHandler {
	return &TransactionStreamHandler{
		BaseHandler: kafkacommon.BaseHandler{
			ClientID:        clientId,
			ConsumerMetrics: consumerMetrics,
			DLQ:             dlq,
			LogPrefix:       logMessage,
		},
		mfs: mfs,
		cfg: cfg,
	}
}

func (tsh TransactionStreamHandler) Setup(session sarama.ConsumerGroupSession) error {
	xlog.Info(
		context.Background(),
		logMessage+"[SETUP]",
		xlog.String("member_id", session.MemberID()),
		xlog.Int32("generation_id", session.GenerationID()),
	)
	return nil
}

func (tsh TransactionStreamHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	xlog.Info(
		context.Background(),
		logMessage+"[CLEANUP]",
		xlog.String("member_id", session.MemberID()),
	)
	return nil
}

func (tsh TransactionStreamHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	xlog.Info(
		session.Context(),
		logMessage+"[CONSUME-CLAIM-START]",
		xlog.String("topic", claim.Topic()),
		xlog.Int32("partition", claim.Partition()),
		xlog.Int64("initial_offset", claim.InitialOffset()),
	)

	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				xlog.Info(session.Context(), logMessage+"[CONSUME-CLAIM-CLOSED]")
				return nil
			}

			ctx := ctxdata.Sets(session.Context(),
				ctxdata.SetCorrelationId(uuid.New().String()),
				ctxdata.SetHost(tsh.ClientID),
			)

			start := time.Now()
			logField := tsh.CreateLogField(message)

			err, shouldSkip := tsh.handler(ctx, message)

			if err != nil {
				logField = append(logField,
					xlog.Duration("response-time", time.Since(start)),
					xlog.Err(err),
				)

				if shouldSkip {
					// Skip message without sending to DLQ (ineligible status or payment type)
					logField = append(logField, xlog.String("status", "skipped"))
					xlog.Info(ctx, logMessage+"[PROCESS-SKIPPED]", logField...)
					tsh.Ack(session, message)
				} else {
					// Send to DLQ (actual error)
					logField = append(logField, xlog.String("status", "failed"))
					xlog.Warn(ctx, logMessage+"[PROCESS-FAILED]", logField...)
					tsh.Nack(ctx, session, message, err)
				}
				continue
			}

			logField = append(logField,
				xlog.Duration("response-time", time.Since(start)),
				xlog.String("status", "success"),
			)
			xlog.Info(ctx, logMessage+"[PROCESS-SUCCESS]", logField...)

			audit.Info(ctx, audit.Message{ActivityData: string(message.Value)})

			tsh.Ack(session, message)

		case <-session.Context().Done():
			xlog.Info(session.Context(), logMessage+"[CONSUME-CLAIM-CONTEXT-DONE]")
			return nil
		}
	}
}

// processMessage returns (error, shouldSkip)
// shouldSkip = true means the message should be acknowledged without sending to DLQ
func (tsh TransactionStreamHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) (error, bool) {
	var (
		logMsg = logMessage + "[PROCESS-MESSAGE]"
	)

	logField := tsh.CreateLogField(message)

	if len(message.Value) == 0 {
		xlog.Warn(ctx, logMsg, append(logField, xlog.String("error", "empty message"))...)
		// Empty message should go to DLQ
		return fmt.Errorf("empty message received"), false
	}

	var transactionEvent gopaymentlib.Event
	if err := json.Unmarshal(message.Value, &transactionEvent); err != nil {
		logField = append(logField, xlog.Err(err), xlog.String("raw_message", string(message.Value)))
		xlog.Warn(ctx, logMsg, logField...)
		// Unmarshal error should go to DLQ
		return fmt.Errorf("error unmarshal json: %w", err), false
	}

	// Validation errors - these should skip without DLQ
	if transactionEvent.ID == "" {
		xlog.Info(ctx, logMsg, append(logField, xlog.String("error", "missing transaction_id"))...)
		return fmt.Errorf("papa_transaction_id is required"), true
	}

	if transactionEvent.PaymentType == "" {
		xlog.Info(ctx, logMsg, append(logField, xlog.String("error", "missing payment_type"))...)
		return fmt.Errorf("payment_type is required"), true
	}

	status := transactionEvent.Status.ConvertSingleAPI().ToString()
	if !validStatuses[status] {
		// Invalid status - skip without DLQ
		xlog.Info(ctx, logMsg, append(logField,
			xlog.String("status", status),
			xlog.String("papa_transaction_id", transactionEvent.ID),
			xlog.String("payment_type", transactionEvent.PaymentType.ConvertSingleAPI().ToString()),
			xlog.String("info", "skipping invalid transaction status"),
		)...)
		return fmt.Errorf("invalid transaction status: %s", status), true
	}

	xlog.Info(ctx, logMsg, append(logField,
		xlog.String("papa_transaction_id", transactionEvent.ID),
		xlog.String("payment_type", transactionEvent.PaymentType.ConvertSingleAPI().ToString()),
		xlog.String("status", status),
		xlog.String("info", "processing successful/rejected transaction"),
	)...)

	err := tsh.mfs.ProcessTransactionStream(ctx, transactionEvent)
	if err != nil {
		// Check if error is due to ineligible payment type
		if isIneligibleTransactionError(err) {
			logField = append(logField, xlog.Err(err), xlog.String("reason", "ineligible_transaction"))
			xlog.Info(ctx, logMsg, logField...)
			// Skip without sending to DLQ
			return err, true
		}

		// Other errors should go to DLQ
		err = fmt.Errorf("unable to process transaction stream: %w", err)
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMsg, logField...)
		return err, false
	}

	xlog.Info(ctx, logMsg+" [SUCCESS]", append(logField,
		xlog.String("papa_transaction_id", transactionEvent.ID),
		xlog.String("payment_type", transactionEvent.PaymentType.ConvertSingleAPI().ToString()),
	)...)

	return nil, false
}

// handler returns (error, shouldSkip)
func (tsh TransactionStreamHandler) handler(ctx context.Context, message *sarama.ConsumerMessage) (error, bool) {
	startTime := time.Now()
	err, shouldSkip := tsh.processMessage(ctx, message)
	tsh.RecordMetrics(startTime, message, err)
	return err, shouldSkip
}

// isIneligibleTransactionError checks if error is due to ineligible transaction
func isIneligibleTransactionError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := err.Error()
	// Check for common ineligible transaction error patterns
	ineligiblePatterns := []string{
		"payment type not found",
		"transaction type not found",
		"not eligible",
		"skipping non-eligible",
		"summary id not found",
	}

	for _, pattern := range ineligiblePatterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// contains checks if string contains substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > len(substr) &&
				containsCaseInsensitive(s, substr))
}

func containsCaseInsensitive(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
