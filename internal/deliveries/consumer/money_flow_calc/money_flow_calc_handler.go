package money_flow_calc

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

type MoneyFlowCalcHandler struct {
	clientId        string
	mfs             services.MoneyFlowService
	cfg             config.Config
	dlq             dlqpublisher.Publisher
	consumerMetrics *metrics.ConsumerMetrics
}

func NewMoneyFlowCalcHandler(
	clientId string,
	mfs services.MoneyFlowService,
	dlq dlqpublisher.Publisher,
	cfg config.Config,
	consumerMetrics *metrics.ConsumerMetrics,
) sarama.ConsumerGroupHandler {
	return &MoneyFlowCalcHandler{clientId, mfs, cfg, dlq, consumerMetrics}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (mfc MoneyFlowCalcHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (mfc MoneyFlowCalcHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (mfc MoneyFlowCalcHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			ctx := ctxdata.Sets(session.Context(),
				ctxdata.SetCorrelationId(uuid.New().String()),
				ctxdata.SetHost(mfc.clientId),
			)

			start := time.Now()
			logField := createLogField(message)

			err := mfc.handler(ctx, message)
			if err != nil {
				logField = append(logField, xlog.Duration("response-time", time.Since(start)), xlog.Err(err))
				xlog.Warn(ctx, logMessage, logField...)

				mfc.Nack(ctx, session, message, err)
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

func (mfc MoneyFlowCalcHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) (err error) {
	var (
		notification models.TransactionNotificationPayload
		logMsg       = "[PROCESS-MESSAGE]"
	)

	logField := createLogField(message)

	if err = json.Unmarshal(message.Value, &notification); err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMsg, logField...)
		return fmt.Errorf("error unmarshal json: %w", err)
	}

	// Process the notification
	err = mfc.mfs.ProcessTransactionNotification(ctx, notification)
	if err != nil {
		err = fmt.Errorf("unable to process transaction notification: %w", err)
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMsg, logField...)
		return err
	}

	xlog.Info(ctx, logMsg, logField...)
	return nil
}

func (mfc MoneyFlowCalcHandler) handler(ctx context.Context, message *sarama.ConsumerMessage) (err error) {
	startTime := time.Now()
	err = mfc.processMessage(ctx, message)

	if mfc.consumerMetrics != nil {
		mfc.consumerMetrics.GenerateMetrics(startTime, message, err)
	}

	return
}

func (mfc MoneyFlowCalcHandler) Ack(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
	session.MarkMessage(message, "")
}

// Nack is a custom function for handling failed messages during Kafka consumer processing.
// It publishes the failed message to a DLQ and mark the message as consumed.
func (mfc MoneyFlowCalcHandler) Nack(ctx context.Context, session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage, causeErr error) {
	logField := createLogField(message)

	err := mfc.dlq.Publish(models.FailedMessage{
		Payload:    message.Value,
		Timestamp:  message.Timestamp,
		CauseError: causeErr,
	})
	if err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
	}

	session.MarkMessage(message, "")
}

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
