package kafka

import (
	"context"
	"time"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/Shopify/sarama"
)

type BaseHandler struct {
	ClientID        string
	ConsumerMetrics *metrics.ConsumerMetrics
	DLQ             dlqpublisher.Publisher
	LogPrefix       string
}

func (b *BaseHandler) CreateLogField(msg *sarama.ConsumerMessage) []xlog.Field {
	return []xlog.Field{
		xlog.Time("timestamp", msg.Timestamp),
		xlog.String("topic", msg.Topic),
		xlog.String("key", string(msg.Key)),
		xlog.Int32("partition", msg.Partition),
		xlog.Int64("offset", msg.Offset),
		xlog.String("message-claimed", string(msg.Value)),
	}
}

func (b *BaseHandler) Ack(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
	session.MarkMessage(message, "")
	xlog.Debug(
		context.Background(),
		b.LogPrefix+"[ACK]",
		xlog.String("topic", message.Topic),
		xlog.Int32("partition", message.Partition),
		xlog.Int64("offset", message.Offset),
	)
}

func (b *BaseHandler) Nack(ctx context.Context, session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage, causeErr error) {
	logField := b.CreateLogField(message)
	logField = append(logField, xlog.Err(causeErr))

	err := b.DLQ.Publish(models.FailedMessage{
		Payload:    message.Value,
		Timestamp:  message.Timestamp,
		CauseError: causeErr,
	})

	if err != nil {
		logField = append(logField, xlog.String("dlq_status", "failed"))
		xlog.Error(ctx, b.LogPrefix+"[NACK-DLQ-FAILED]", logField...)
	} else {
		logField = append(logField, xlog.String("dlq_status", "success"))
		xlog.Info(ctx, b.LogPrefix+"[NACK-DLQ-SUCCESS]", logField...)
	}

	session.MarkMessage(message, "")
	xlog.Warn(ctx, b.LogPrefix+"[NACK]", logField...)
}

func (b *BaseHandler) RecordMetrics(startTime time.Time, message *sarama.ConsumerMessage, err error) {
	if b.ConsumerMetrics != nil {
		b.ConsumerMetrics.GenerateMetrics(startTime, message, err)
	}
}
