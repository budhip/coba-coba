package dlqretrier

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

type DLQRetrierHandler struct {
	clientId        string
	dp              services.DLQProcessorService
	consumerCfg     config.ConsumerConfig
	consumerMetrics *metrics.ConsumerMetrics
}

func NewRetrierHandler(clientId string, dp services.DLQProcessorService, consumerCfg config.ConsumerConfig, consumerMetrics *metrics.ConsumerMetrics) sarama.ConsumerGroupHandler {
	return &DLQRetrierHandler{clientId, dp, consumerCfg, consumerMetrics}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (dt DLQRetrierHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (dt DLQRetrierHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (dt DLQRetrierHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			ctx := ctxdata.Sets(session.Context(),
				ctxdata.SetCorrelationId(uuid.New().String()),
				ctxdata.SetHost(dt.clientId),
			)
			start := time.Now()
			logField := createLogField(message)

			err := dt.handler(ctx, message)
			if err != nil {
				logField = append(logField, xlog.Duration("response-time", time.Since(start)), xlog.Err(err))
				xlog.Warn(ctx, logMessage, logField...)
				continue
			}
			logField = append(logField, xlog.Duration("response-time", time.Since(start)))
			xlog.Info(ctx, logMessage, logField...)
			audit.Info(ctx, audit.Message{ActivityData: string(message.Value)})
			session.MarkMessage(message, "")
		case <-session.Context().Done():
			return nil
		}
	}
}

func (dt DLQRetrierHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	var (
		payload    models.FailedMessage
		logMessage = "[PROCESS-MESSAGE]"
	)

	logField := createLogField(message)

	if err := json.Unmarshal(message.Value, &payload); err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
		return fmt.Errorf("error unmarshal json: %w", err)
	}

	var err error
	if message.Topic == dt.consumerCfg.TopicAccountMutationDLQ {
		err = dt.dp.RetryAccountMutation(ctx, payload)
	} else if message.Topic == dt.consumerCfg.TopicDLQ {
		err = dt.dp.RetryCreateOrderTransaction(ctx, payload)
	} else {
		err = fmt.Errorf("unknown topic: %s", message.Topic)
	}

	if err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
		return fmt.Errorf("err process dlq message: %w", err)
	}

	xlog.Info(ctx, logMessage, logField...)
	return nil
}

func (dt DLQRetrierHandler) handler(ctx context.Context, message *sarama.ConsumerMessage) (err error) {
	startTime := time.Now() // time when a process consumes a message started
	err = dt.processMessage(ctx, message)

	if dt.consumerMetrics != nil {
		dt.consumerMetrics.GenerateMetrics(startTime, message, err)
	}

	return
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
