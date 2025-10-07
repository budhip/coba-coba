package queuerecon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	xlog "bitbucket.org/Amartha/go-x/log"
	"bitbucket.org/Amartha/go-x/log/audit"
	"bitbucket.org/Amartha/go-x/log/ctxdata"

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
)

type TaskQueueReconHandler struct {
	clientId string
	rs       services.ReconService
}

func NewTaskQueueReconHandler(clientId string, rs services.ReconService) sarama.ConsumerGroupHandler {
	return &TaskQueueReconHandler{
		clientId: clientId,
		rs:       rs,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (am TaskQueueReconHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (am TaskQueueReconHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (am TaskQueueReconHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message := <-claim.Messages():
			{
				ctx := ctxdata.Sets(session.Context(),
					ctxdata.SetCorrelationId(uuid.New().String()),
					ctxdata.SetHost(am.clientId),
				)
				start := time.Now()
				logField := []xlog.Field{
					xlog.Time("timestamp", message.Timestamp),
					xlog.String("topic", message.Topic),
					xlog.String("key", string(message.Key)),
					xlog.Int32("partition", message.Partition),
					xlog.Int64("offset", message.Offset),
					xlog.String("message-claimed", string(message.Value)),
				}
				err := am.processMessage(ctx, message)
				if err != nil {
					logField = append(logField, xlog.Duration("response-time", time.Since(start)), xlog.Err(err))
					xlog.Warn(ctx, logMessage, logField...)
					continue
				}
				logField = append(logField, xlog.Duration("response-time", time.Since(start)))
				xlog.Info(ctx, logMessage, logField...)
				audit.Info(ctx, audit.Message{ActivityData: string(message.Value)})
				session.MarkMessage(message, "")
			}
		case <-session.Context().Done():
			return nil
		}
	}
}

func (am TaskQueueReconHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	var (
		payload    models.ReconPublisher
		logMessage = "[PROCESS-MESSAGE]"
	)

	logField := []xlog.Field{
		xlog.Time("timestamp", message.Timestamp),
		xlog.String("topic", message.Topic),
		xlog.String("key", string(message.Key)),
		xlog.Int32("partition", message.Partition),
		xlog.Int64("offset", message.Offset),
		xlog.String("message-claimed", string(message.Value)),
	}

	if err := json.Unmarshal(message.Value, &payload); err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
		return fmt.Errorf("error unmarshal json: %w", err)
	}

	if payload.Task != models.ReconTaskName {
		logField = append(logField,
			xlog.String("task", payload.Task),
			xlog.Err(errors.New("unsupported task")))
		xlog.Warn(ctx, logMessage, logField...)
		return nil
	}

	id, err := strconv.ParseUint(payload.ID, 10, 64)
	if err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
		return fmt.Errorf("unable to parse id payload to uint64: %w", err)
	}

	err = am.rs.ProcessReconTaskQueue(ctx, id)
	if err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
		return fmt.Errorf("error handle transaction failure: %w", err)
	}

	xlog.Info(ctx, logMessage, logField...)
	return nil
}
