package money_flow_calc

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
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
	mfs services.MoneyFlowService
	cfg config.Config
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
		mfs: mfs,
		cfg: cfg,
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
			ctx := ctxdata.Sets(session.Context(),
				ctxdata.SetCorrelationId(uuid.New().String()),
				ctxdata.SetHost(mfc.ClientID),
			)

			start := time.Now()
			logField := mfc.CreateLogField(message)

			err, shouldSkip := mfc.handler(ctx, message)
			if err != nil {
				logField = append(logField, xlog.Duration("response-time", time.Since(start)), xlog.Err(err))

				if shouldSkip {
					// Skip message without sending to DLQ (ineligible transaction)
					xlog.Info(ctx, logMessage, append(logField, xlog.String("action", "skipped"))...)
					mfc.Ack(session, message)
				} else {
					// Send to DLQ (actual error)
					xlog.Warn(ctx, logMessage, append(logField, xlog.String("action", "sent_to_dlq"))...)
					mfc.Nack(ctx, session, message, err)
				}
				continue
			}

			logField = append(logField, xlog.Duration("response-time", time.Since(start)))
			xlog.Info(ctx, logMessage, logField...)
			audit.Info(ctx, audit.Message{ActivityData: string(message.Value)})

			mfc.Ack(session, message)
		case <-session.Context().Done():
			return nil
		}
	}
}

// processMessage returns (error, shouldSkip)
// shouldSkip = true means the message should be acknowledged without sending to DLQ
func (mfc MoneyFlowCalcHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) (error, bool) {
	var (
		logMsg = "[PROCESS-MESSAGE]"
	)

	logField := mfc.CreateLogField(message)

	var rawNotif models.TransactionNotificationRaw
	if err := json.Unmarshal(message.Value, &rawNotif); err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMsg, logField...)
		// Unmarshal error should go to DLQ
		return fmt.Errorf("error unmarshal json to raw: %w", err), false
	}

	fixedAcuanData := mfc.fixAmountInJSON(rawNotif.AcuanData)

	var notification goacuanlib.Payload[goacuanlib.DataOrder]

	if len(fixedAcuanData) > 0 {
		var acuanData goacuanlib.Payload[goacuanlib.DataOrder]
		if err := json.Unmarshal(fixedAcuanData, &acuanData); err != nil {
			logField = append(logField, xlog.Err(err))
			xlog.Warn(ctx, logMsg, logField...)
			// Unmarshal error should go to DLQ
			return fmt.Errorf("error unmarshal acuan data: %w", err), false
		}
		notification = acuanData
	}

	err := mfc.mfs.ProcessTransactionNotification(ctx, notification)
	if err != nil {
		// Check if error is due to ineligible transaction type
		if isIneligibleTransactionError(err) {
			logField = append(logField, xlog.Err(err), xlog.String("reason", "ineligible_transaction"))
			xlog.Info(ctx, logMsg, logField...)
			// Skip without sending to DLQ
			return err, true
		}

		// Other errors should go to DLQ
		err = fmt.Errorf("unable to process transaction notification: %w", err)
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMsg, logField...)
		return err, false
	}

	xlog.Info(ctx, logMsg, logField...)
	return nil, false
}

// handler returns (error, shouldSkip)
func (mfc MoneyFlowCalcHandler) handler(ctx context.Context, message *sarama.ConsumerMessage) (error, bool) {
	startTime := time.Now()
	err, shouldSkip := mfc.processMessage(ctx, message)
	mfc.RecordMetrics(startTime, message, err)
	return err, shouldSkip
}

func (mfc MoneyFlowCalcHandler) fixAmountInJSON(data []byte) []byte {
	if len(data) == 0 {
		return data
	}

	dataStr := string(data)

	balanceFields := []string{
		"actualBalance",
		"pendingBalance",
		"availableBalance",
	}

	for _, field := range balanceFields {
		pattern := fmt.Sprintf(`"%s"\s*:\s*\{\s*"value"\s*:\s*(\d+)\s*,\s*"currency"\s*:\s*"[^"]*"\s*\}`, field)
		re := regexp.MustCompile(pattern)
		dataStr = re.ReplaceAllString(dataStr, fmt.Sprintf(`"%s":"$1"`, field))
	}

	return []byte(dataStr)
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
		"skipping ineligible",
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
