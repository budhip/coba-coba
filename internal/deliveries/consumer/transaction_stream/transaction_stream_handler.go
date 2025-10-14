package transaction_stream

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	xlog "bitbucket.org/Amartha/go-x/log"
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
	clientId        string
	mfs             services.MoneyFlowService
	cfg             config.Config
	dlq             dlqpublisher.Publisher
	consumerMetrics *metrics.ConsumerMetrics
}

// NewTransactionStreamHandler creates a new handler instance
func NewTransactionStreamHandler(
	clientId string,
	mfs services.MoneyFlowService,
	dlq dlqpublisher.Publisher,
	cfg config.Config,
	consumerMetrics *metrics.ConsumerMetrics,
) sarama.ConsumerGroupHandler {
	return &TransactionStreamHandler{
		clientId:        clientId,
		mfs:             mfs,
		cfg:             cfg,
		dlq:             dlq,
		consumerMetrics: consumerMetrics,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (tsh TransactionStreamHandler) Setup(session sarama.ConsumerGroupSession) error {
	xlog.Info(
		context.Background(),
		logMessage+"[SETUP]",
		xlog.String("member_id", session.MemberID()),
		xlog.Int32("generation_id", session.GenerationID()),
	)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (tsh TransactionStreamHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	xlog.Info(
		context.Background(),
		logMessage+"[CLEANUP]",
		xlog.String("member_id", session.MemberID()),
	)
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
// This is the main processing loop for incoming Kafka messages
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

			// Create context with correlation ID and host info
			ctx := ctxdata.Sets(session.Context(),
				ctxdata.SetCorrelationId(uuid.New().String()),
				ctxdata.SetHost(tsh.clientId),
			)

			start := time.Now()
			logField := createLogField(message)

			// Process the message
			err := tsh.handler(ctx, message)

			if err != nil {
				// Log error with details
				logField = append(logField,
					xlog.Duration("response-time", time.Since(start)),
					xlog.Err(err),
					xlog.String("status", "failed"),
				)
				xlog.Warn(ctx, logMessage+"[PROCESS-FAILED]", logField...)

				// Send to DLQ and mark as consumed
				tsh.Nack(ctx, session, message, err)
				continue
			}

			// Log success
			logField = append(logField,
				xlog.Duration("response-time", time.Since(start)),
				xlog.String("status", "success"),
			)
			xlog.Info(ctx, logMessage+"[PROCESS-SUCCESS]", logField...)

			// Audit log for tracking
			audit.Info(ctx, audit.Message{ActivityData: string(message.Value)})

			// Mark message as consumed
			tsh.Ack(session, message)

		case <-session.Context().Done():
			xlog.Info(session.Context(), logMessage+"[CONSUME-CLAIM-CONTEXT-DONE]")
			return nil
		}
	}
}

// processMessage handles the actual message processing logic
func (tsh TransactionStreamHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	var (
		logMsg = logMessage + "[PROCESS-MESSAGE]"
	)

	logField := createLogField(message)

	// Step 1: Validate message is not empty
	if len(message.Value) == 0 {
		xlog.Warn(ctx, logMsg, append(logField, xlog.String("error", "empty message"))...)
		return fmt.Errorf("empty message received")
	}

	// Step 2: Unmarshal transaction stream event
	var transactionEvent models.TransactionStreamEvent
	if err := json.Unmarshal(message.Value, &transactionEvent); err != nil {
		logField = append(logField, xlog.Err(err), xlog.String("raw_message", string(message.Value)))
		xlog.Warn(ctx, logMsg, logField...)
		return fmt.Errorf("error unmarshal json: %w", err)
	}

	// Step 3: Validate required fields
	if transactionEvent.TransactionID == "" {
		xlog.Warn(ctx, logMsg, append(logField, xlog.String("error", "missing transaction_id"))...)
		return fmt.Errorf("papa_transaction_id is required")
	}

	if transactionEvent.PaymentType == "" {
		xlog.Warn(ctx, logMsg, append(logField, xlog.String("error", "missing payment_type"))...)
		return fmt.Errorf("payment_type is required")
	}

	// Step 4: Filter - Process only SUCCESSFUL or REJECTED status
	if !validStatuses[transactionEvent.Status] {
		xlog.Info(ctx, logMsg, append(logField,
			xlog.String("status", transactionEvent.Status),
			xlog.String("papa_transaction_id", transactionEvent.TransactionID),
			xlog.String("payment_type", transactionEvent.PaymentType),
			xlog.String("info", "skipping invalid transaction status"),
		)...)
		return nil
	}

	// Step 5: Additional logging for SUCCESSFUL or REJECTED transactions
	xlog.Info(ctx, logMsg, append(logField,
		xlog.String("papa_transaction_id", transactionEvent.TransactionID),
		xlog.String("payment_type", transactionEvent.PaymentType),
		xlog.String("status", transactionEvent.Status),
		xlog.String("amount", transactionEvent.Amount),
		xlog.String("info", "processing successful/rejected transaction"),
	)...)

	// Step 6: Process the transaction stream event via service layer
	err := tsh.mfs.ProcessTransactionStream(ctx, transactionEvent)
	if err != nil {
		err = fmt.Errorf("unable to process transaction stream: %w", err)
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMsg, logField...)
		return err
	}

	xlog.Info(ctx, logMsg+" [SUCCESS]", append(logField,
		xlog.String("papa_transaction_id", transactionEvent.TransactionID),
		xlog.String("payment_type", transactionEvent.PaymentType),
	)...)

	return nil
}

// handler wraps processMessage with metrics tracking
func (tsh TransactionStreamHandler) handler(ctx context.Context, message *sarama.ConsumerMessage) (err error) {
	startTime := time.Now()

	// Process the message
	err = tsh.processMessage(ctx, message)

	// Generate and record metrics if available
	if tsh.consumerMetrics != nil {
		tsh.consumerMetrics.GenerateMetrics(startTime, message, err)
	}

	return err
}

// Ack marks a message as successfully processed
func (tsh TransactionStreamHandler) Ack(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
	session.MarkMessage(message, "")

	xlog.Debug(
		context.Background(),
		logMessage+"[ACK]",
		xlog.String("topic", message.Topic),
		xlog.Int32("partition", message.Partition),
		xlog.Int64("offset", message.Offset),
	)
}

// Nack handles failed messages by sending them to DLQ and marking as consumed
// This prevents the message from being reprocessed indefinitely
func (tsh TransactionStreamHandler) Nack(ctx context.Context, session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage, causeErr error) {
	logField := createLogField(message)
	logField = append(logField, xlog.Err(causeErr))

	// Attempt to publish to DLQ
	err := tsh.dlq.Publish(models.FailedMessage{
		Payload:    message.Value,
		Timestamp:  message.Timestamp,
		CauseError: causeErr,
	})

	if err != nil {
		// Log DLQ publish failure
		logField = append(logField,
			xlog.Err(fmt.Errorf("dlq publish failed: %w", err)),
			xlog.String("dlq_status", "failed"),
		)
		xlog.Error(ctx, logMessage+"[NACK-DLQ-FAILED]", logField...)
	} else {
		logField = append(logField, xlog.String("dlq_status", "success"))
		xlog.Info(ctx, logMessage+"[NACK-DLQ-SUCCESS]", logField...)
	}

	// Mark message as consumed to prevent reprocessing
	// This is intentional - we don't want to retry indefinitely
	session.MarkMessage(message, "")

	xlog.Warn(ctx, logMessage+"[NACK]", logField...)
}

// createLogField creates standardized log fields from a Kafka message
func createLogField(msg *sarama.ConsumerMessage) []xlog.Field {
	return []xlog.Field{
		xlog.Time("timestamp", msg.Timestamp),
		xlog.String("topic", msg.Topic),
		xlog.String("key", string(msg.Key)),
		xlog.Int32("partition", msg.Partition),
		xlog.Int64("offset", msg.Offset),
		xlog.String("message-claimed", string(msg.Value)),
	}
}
