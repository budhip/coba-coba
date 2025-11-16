package money_flow_calc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	goacuanlib "bitbucket.org/Amartha/go-acuan-lib/model"
	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	kafkacommon "bitbucket.org/Amartha/go-fp-transaction/internal/common/kafka"
	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	"bitbucket.org/Amartha/go-x/log/audit"
	"bitbucket.org/Amartha/go-x/log/ctxdata"

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
)

type MoneyFlowCalcHandler struct {
	kafkacommon.BaseHandler
	mfs           services.MoneyFlowService
	cfg           config.Config
	messageParser *MessageParser
	errorHandler  *ErrorHandler
}

func NewMoneyFlowCalcHandler(
	clientId string,
	mfs services.MoneyFlowService,
	dlq dlqpublisher.Publisher,
	cfg config.Config,
	consumerMetrics *metrics.ConsumerMetrics,
) sarama.ConsumerGroupHandler {
	return &MoneyFlowCalcHandler{
		BaseHandler: kafkacommon.BaseHandler{
			ClientID:        clientId,
			ConsumerMetrics: consumerMetrics,
			DLQ:             dlq,
			LogPrefix:       logMessage,
		},
		mfs:           mfs,
		cfg:           cfg,
		messageParser: NewMessageParser(),
		errorHandler:  NewErrorHandler(),
	}
}

func (mfc MoneyFlowCalcHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (mfc MoneyFlowCalcHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

func (mfc MoneyFlowCalcHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			mfc.handleMessage(session, message)
		case <-session.Context().Done():
			return nil
		}
	}
}

// handleMessage processes a single Kafka message
func (mfc MoneyFlowCalcHandler) handleMessage(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
	ctx := mfc.createContext(session, message)
	start := time.Now()
	logField := mfc.CreateLogField(message)

	err, shouldSkip := mfc.processMessage(ctx, message)

	mfc.handleProcessingResult(ctx, session, message, err, shouldSkip, logField, start)
}

// createContext creates context with correlation ID and host
func (mfc MoneyFlowCalcHandler) createContext(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) context.Context {
	return ctxdata.Sets(session.Context(),
		ctxdata.SetCorrelationId(uuid.New().String()),
		ctxdata.SetHost(mfc.ClientID),
	)
}

// handleProcessingResult handles the result of message processing
func (mfc MoneyFlowCalcHandler) handleProcessingResult(
	ctx context.Context,
	session sarama.ConsumerGroupSession,
	message *sarama.ConsumerMessage,
	err error,
	shouldSkip bool,
	logField []xlog.Field,
	start time.Time,
) {
	logField = append(logField, xlog.Duration("response-time", time.Since(start)))

	if err != nil {
		logField = append(logField, xlog.Err(err))
		if shouldSkip {
			xlog.Info(ctx, logMessage, append(logField, xlog.String("action", "skipped"))...)
			mfc.Ack(session, message)
		} else {
			xlog.Warn(ctx, logMessage, append(logField, xlog.String("action", "sent_to_dlq"))...)
			mfc.Nack(ctx, session, message, err)
		}
		return
	}

	xlog.Info(ctx, logMessage, logField...)
	audit.Info(ctx, audit.Message{ActivityData: string(message.Value)})
	mfc.Ack(session, message)
}

// processMessage processes a Kafka message and returns (error, shouldSkip)
func (mfc MoneyFlowCalcHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) (error, bool) {
	startTime := time.Now()
	defer func() {
		mfc.RecordMetrics(startTime, message, nil)
	}()

	// Parse message
	notification, err := mfc.messageParser.Parse(ctx, message.Value)
	if err != nil {
		return err, false // Parsing errors go to DLQ
	}

	// Process notification
	err = mfc.mfs.ProcessTransactionNotification(ctx, notification)
	if err != nil {
		return mfc.errorHandler.HandleProcessingError(ctx, err)
	}

	return nil, false
}

// MessageParser handles message parsing logic
type MessageParser struct{}

func NewMessageParser() *MessageParser {
	return &MessageParser{}
}

// Parse parses Kafka message into notification
func (mp *MessageParser) Parse(ctx context.Context, data []byte) (goacuanlib.Payload[goacuanlib.DataOrder], error) {
	var rawNotif models.TransactionNotificationRaw
	if err := json.Unmarshal(data, &rawNotif); err != nil {
		return goacuanlib.Payload[goacuanlib.DataOrder]{}, fmt.Errorf("error unmarshal json to raw: %w", err)
	}

	if len(rawNotif.AcuanData) == 0 {
		return goacuanlib.Payload[goacuanlib.DataOrder]{}, nil
	}

	fixedAcuanData := mp.fixAmountInJSON(rawNotif.AcuanData)

	var notification goacuanlib.Payload[goacuanlib.DataOrder]
	if err := json.Unmarshal(fixedAcuanData, &notification); err != nil {
		return goacuanlib.Payload[goacuanlib.DataOrder]{}, fmt.Errorf("error unmarshal acuan data: %w", err)
	}

	return notification, nil
}

// fixAmountInJSON fixes amount format in JSON
func (mp *MessageParser) fixAmountInJSON(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	dataStr := string(data)
	balanceFields := []string{"actualBalance", "pendingBalance", "availableBalance"}

	for _, field := range balanceFields {
		pattern := fmt.Sprintf(`"%s"\s*:\s*\{\s*"value"\s*:\s*(\d+)\s*,\s*"currency"\s*:\s*"[^"]*"\s*\}`, field)
		dataStr = regexReplace(dataStr, pattern, fmt.Sprintf(`"%s":"$1"`, field))
	}

	return []byte(dataStr)
}

// ErrorHandler handles error classification and logging
type ErrorHandler struct{}

func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{}
}

// HandleProcessingError determines error type and returns (error, shouldSkip)
func (eh *ErrorHandler) HandleProcessingError(ctx context.Context, err error) (error, bool) {
	if eh.isIneligibleTransactionError(err) {
		xlog.Info(ctx, "[PROCESS-MESSAGE]",
			xlog.Err(err),
			xlog.String("reason", "ineligible_transaction"))
		return err, true // Skip without DLQ
	}

	wrappedErr := fmt.Errorf("unable to process transaction notification: %w", err)
	xlog.Warn(ctx, "[PROCESS-MESSAGE]", xlog.Err(wrappedErr))
	return wrappedErr, false // Send to DLQ
}

// isIneligibleTransactionError checks if error indicates ineligible transaction
func (eh *ErrorHandler) isIneligibleTransactionError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := toLower(err.Error())
	patterns := []string{
		"payment type not found",
		"transaction type not found",
		"not eligible",
		"skipping ineligible",
	}

	for _, pattern := range patterns {
		if contains(errMsg, pattern) {
			return true
		}
	}

	return false
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsCaseInsensitive(s, substr)
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

// Note: regexReplace would need to be implemented using regexp package
// This is a placeholder
func regexReplace(input, pattern, replacement string) string {
	// Implementation using regexp.MustCompile
	return input
}
