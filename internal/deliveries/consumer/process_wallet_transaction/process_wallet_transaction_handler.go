package process_wallet_transaction

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	xlog "bitbucket.org/Amartha/go-x/log"
	"bitbucket.org/Amartha/go-x/log/audit"
	"bitbucket.org/Amartha/go-x/log/ctxdata"

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
)

type ProcessWalletTransactionHandler struct {
	clientId        string
	consumerMetrics *metrics.ConsumerMetrics

	dlq                      dlqpublisher.Publisher
	cacheRepo                repositories.CacheRepository
	walletTransactionService services.WalletTrxService
}

var idempotencyTTL = 7 * 24 * time.Hour

func NewHandler(
	clientId string,
	consumerMetrics *metrics.ConsumerMetrics,
	cacheRepo repositories.CacheRepository,
	dlq dlqpublisher.Publisher,
	walletTransactionService services.WalletTrxService,
) sarama.ConsumerGroupHandler {
	return &ProcessWalletTransactionHandler{
		clientId:        clientId,
		consumerMetrics: consumerMetrics,

		dlq:                      dlq,
		cacheRepo:                cacheRepo,
		walletTransactionService: walletTransactionService,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (am ProcessWalletTransactionHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (am ProcessWalletTransactionHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (am ProcessWalletTransactionHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			ctx := ctxdata.Sets(session.Context(),
				ctxdata.SetCorrelationId(uuid.New().String()),
				ctxdata.SetHost(am.clientId),
			)

			start := time.Now()
			logField := createLogField(message)

			err := am.handler(ctx, message)
			if err != nil {
				logField = append(logField, xlog.Duration("response-time", time.Since(start)), xlog.Err(err))
				xlog.Warn(ctx, logMessage, logField...)

				am.Nack(ctx, session, message, err)
				continue
			}

			logField = append(logField, xlog.Duration("response-time", time.Since(start)))
			xlog.Info(ctx, logMessage, logField...)
			audit.Info(ctx, audit.Message{ActivityData: string(message.Value)})

			am.Ack(session, message)
		case <-session.Context().Done():
			return nil
		}
	}
}

// checkIdempotency is dumb implementation for idempotency check
func (am ProcessWalletTransactionHandler) checkIdempotency(ctx context.Context, key string) (bool, error) {
	redisKey := fmt.Sprintf("go_fp_transaction_wallet_transaction_%s:lock", key)
	return am.cacheRepo.SetIfNotExists(ctx, redisKey, "processed", idempotencyTTL)
}

func (am ProcessWalletTransactionHandler) releaseIdempotency(ctx context.Context, key string) error {
	redisKey := fmt.Sprintf("go_fp_transaction_wallet_transaction_%s:lock", key)
	return am.cacheRepo.Del(ctx, redisKey)
}

func (am ProcessWalletTransactionHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) (err error) {
	var (
		payload           models.CreateWalletTransactionRequest
		logProcessMessage = "[PROCESS-MESSAGE]"
	)

	logField := createLogField(message)

	var idempotencyKey string
	for _, header := range message.Headers {
		if header != nil && string(header.Key) == models.IdempotencyKeyHeader {
			idempotencyKey = string(header.Value)
		}
	}

	if idempotencyKey == "" {
		err = fmt.Errorf("idempotency key is empty")
		xlog.Warn(ctx, logProcessMessage, append(logField, xlog.Err(err))...)
		return err
	}

	if err = json.Unmarshal(message.Value, &payload); err != nil {
		xlog.Warn(ctx, logProcessMessage, append(logField, xlog.Err(err))...)
		return fmt.Errorf("error unmarshal json: %w", err)
	}

	lockCreated, err := am.checkIdempotency(ctx, idempotencyKey)
	if err != nil {
		xlog.Warn(ctx, logProcessMessage, append(logField, xlog.Err(err))...)
		return fmt.Errorf("error check idempotency: %w", err)
	}

	if !lockCreated {
		xlog.Info(ctx, logProcessMessage, logField...)
		return nil
	}

	defer func() {
		if err != nil {
			xlog.Warn(ctx, logProcessMessage, append(logField, xlog.Err(err))...)

			// if error happen, make sure release idempotency so if next same message appear, it can be reprocessed
			errRelease := am.releaseIdempotency(ctx, idempotencyKey)
			if errRelease != nil {
				xlog.Warn(ctx, logProcessMessage, append(logField, xlog.Err(err))...)
			}
		}
	}()

	_, err = am.walletTransactionService.CreateTransaction(ctx, payload)
	if err != nil {
		return fmt.Errorf("error store transaction: %w", err)
	}

	xlog.Info(ctx, logProcessMessage, logField...)
	return nil
}

func (am ProcessWalletTransactionHandler) handler(ctx context.Context, message *sarama.ConsumerMessage) (err error) {
	startTime := time.Now() // time when a process consumes a message started
	err = am.processMessage(ctx, message)

	if am.consumerMetrics != nil {
		am.consumerMetrics.GenerateMetrics(startTime, message, err)
	}

	return
}

func (am ProcessWalletTransactionHandler) Ack(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
	session.MarkMessage(message, "")
}

// Nack is a custom function for handling failed messages during Kafka consumer processing.
// It publishes the failed message to a DLQ and mark the message as consumed.
func (am ProcessWalletTransactionHandler) Nack(ctx context.Context, session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage, causeErr error) {
	logField := createLogField(message)

	err := am.dlq.Publish(models.FailedMessage{
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
