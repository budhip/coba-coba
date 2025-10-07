package hvtbalanceupdate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	xlog "bitbucket.org/Amartha/go-x/log"
	"bitbucket.org/Amartha/go-x/log/audit"
	"bitbucket.org/Amartha/go-x/log/ctxdata"

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
)

type HvtBalanceHandler struct {
	clientId        string
	bs              services.BalanceService
	cacheRepo       repositories.CacheRepository
	dlq             dlqpublisher.Publisher
	featureFlag     config.FeatureFlag
	consumerMetrics *metrics.ConsumerMetrics
}

type ackPayload struct {
	message    *sarama.ConsumerMessage
	hvtPayload models.UpdateBalanceHVTPayload
}

func NewHvtBalanceHandler(
	clientId string,
	bs services.BalanceService,
	cacheRepo repositories.CacheRepository,
	dlq dlqpublisher.Publisher,
	featureFlag config.FeatureFlag,
	consumerMetrics *metrics.ConsumerMetrics,
) sarama.ConsumerGroupHandler {
	return &HvtBalanceHandler{
		clientId:        clientId,
		bs:              bs,
		cacheRepo:       cacheRepo,
		dlq:             dlq,
		featureFlag:     featureFlag,
		consumerMetrics: consumerMetrics,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (bh HvtBalanceHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (bh HvtBalanceHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (bh HvtBalanceHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			ctx := ctxdata.Sets(session.Context(),
				ctxdata.SetCorrelationId(uuid.New().String()),
				ctxdata.SetHost(bh.clientId),
			)
			start := time.Now()
			logField := createLogField(message)

			ackPld := &ackPayload{
				message: message,
			}

			hvtPayload, err := bh.parseMessageToHvtPayload(ctx, message)
			if err != nil {
				logField = append(logField, xlog.Duration("response-time", time.Since(start)), xlog.Err(err))
				xlog.Warn(ctx, logMessage, logField...)
				bh.Nack(ctx, session, ackPld, err)
				continue
			}
			ackPld.hvtPayload = *hvtPayload

			idempotencyKey := bh.createIdempotencyKey(ackPld.hvtPayload.WalletTransactionId, ackPld.hvtPayload.RefNumber, ackPld.hvtPayload.AccountNumber)
			set, err := bh.cacheRepo.SetIfNotExists(ctx, idempotencyKey, fmt.Sprintf("Processing: %s", idempotencyKey), models.TTLIdempotency)
			if err != nil {
				logField = append(logField, xlog.Duration("response-time", time.Since(start)), xlog.Err(err))
				xlog.Warn(ctx, logMessage, logField...)
				bh.Nack(ctx, session, ackPld, err)
				continue
			}

			if !set {
				logField = append(logField, xlog.Duration("response-time", time.Since(start)), xlog.Err(common.ErrRequestBeingProcessed))
				xlog.Warn(ctx, logMessage, logField...)
				bh.Ack(session, ackPld)
				continue
			}

			err = bh.handler(ctx, message, &ackPld.hvtPayload)
			if err != nil {
				logField = append(logField, xlog.Duration("response-time", time.Since(start)), xlog.Err(err))
				xlog.Warn(ctx, logMessage, logField...)
				bh.Nack(ctx, session, ackPld, err)
				continue
			}

			logField = append(logField, xlog.Duration("response-time", time.Since(start)))
			xlog.Info(ctx, logMessage, logField...)
			audit.Info(ctx, audit.Message{ActivityData: string(message.Value)})
			bh.Ack(session, ackPld)
		case <-session.Context().Done():
			return nil
		}

	}
}
func (bh HvtBalanceHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage, hvtPayload *models.UpdateBalanceHVTPayload) error {
	const logMessage = "[PROCESS-MESSAGE]"
	logField := append(
		createLogField(message),
		xlog.Any("request", hvtPayload),
	)

	if err := bh.bs.AdjustAccountBalance(ctx, hvtPayload.AccountNumber, hvtPayload.UpdateAmount.ValueDecimal); err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
		return fmt.Errorf("error when Increment HVT Balance: %w", err)
	}
	xlog.Info(ctx, logMessage, logField...)

	return nil
}

func (bh HvtBalanceHandler) handler(ctx context.Context, message *sarama.ConsumerMessage, hvtPayload *models.UpdateBalanceHVTPayload) (err error) {
	startTime := time.Now()
	err = bh.processMessage(ctx, message, hvtPayload)

	if bh.consumerMetrics != nil {
		bh.consumerMetrics.GenerateMetrics(startTime, message, err)
	}

	return
}

func (bh HvtBalanceHandler) parseMessageToHvtPayload(ctx context.Context, msg *sarama.ConsumerMessage) (*models.UpdateBalanceHVTPayload, error) {
	var (
		payload    models.UpdateBalanceHVTPayload
		logMessage = "[PROCESS-MESSAGE]"
	)

	logField := createLogField(msg)

	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
		return nil, fmt.Errorf("error unmarshal json: %w", err)
	}

	return &payload, nil
}

func (bh HvtBalanceHandler) Ack(session sarama.ConsumerGroupSession, payload *ackPayload) {
	session.MarkMessage(payload.message, "")
}

// Nack is a custom function for handling failed messages during Kafka consumer processing.
// It sends failed message to dlq and acknowledge the message.
func (bh HvtBalanceHandler) Nack(ctx context.Context, session sarama.ConsumerGroupSession, payload *ackPayload, causeErr error) {
	logField := createLogField(payload.message)

	if errors.Is(causeErr, common.ErrRequestBeingProcessed) {
		// Don't delete the idempotency key & Don't send to DLQ
		session.MarkMessage(payload.message, "")
		return
	}

	failedMessage := models.FailedMessage{
		Payload:    payload.message.Value,
		Timestamp:  payload.message.Timestamp,
		CauseError: causeErr,
		Error:      causeErr.Error(),
	}

	idempotencyKey := bh.createIdempotencyKey(payload.hvtPayload.WalletTransactionId, payload.hvtPayload.RefNumber, payload.hvtPayload.AccountNumber)
	err := bh.cacheRepo.Del(ctx, idempotencyKey)
	if err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
	}

	// Feature flag to publish hvt balance dlq
	if !bh.featureFlag.EnablePublishHvtBalanceDLQ {
		session.MarkMessage(payload.message, "")
		return
	}

	if err = bh.dlq.Publish(failedMessage); err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
	}

	session.MarkMessage(payload.message, "")
}

func (bh HvtBalanceHandler) createIdempotencyKey(trxID, refNumber, accountNumber string) string {
	return fmt.Sprintf("acuan:hvt:%s:%s:%s", trxID, refNumber, accountNumber)
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
