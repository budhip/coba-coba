package account_mutation

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

	goacuanlib "bitbucket.org/Amartha/go-acuan-lib/model"
	xlog "bitbucket.org/Amartha/go-x/log"
	"bitbucket.org/Amartha/go-x/log/audit"
	"bitbucket.org/Amartha/go-x/log/ctxdata"

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
)

type AccountMutationHandler struct {
	clientId        string
	as              services.AccountService
	cfg             config.Config
	dlq             dlqpublisher.Publisher
	consumerMetrics *metrics.ConsumerMetrics
}

func NewAccountMutationHandler(
	clientId string,
	as services.AccountService,
	dlq dlqpublisher.Publisher,
	cfg config.Config,
	consumerMetrics *metrics.ConsumerMetrics,
) sarama.ConsumerGroupHandler {
	return &AccountMutationHandler{clientId, as, cfg, dlq, consumerMetrics}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (am AccountMutationHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (am AccountMutationHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (am AccountMutationHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
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

func (am AccountMutationHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) (err error) {
	var (
		payload    goacuanlib.Payload[goacuanlib.DataAccount]
		logMessage = "[PROCESS-MESSAGE]"
	)

	logField := createLogField(message)

	if err = json.Unmarshal(message.Value, &payload); err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
		return fmt.Errorf("error unmarshal json: %w", err)
	}

	accountStream := payload.Body.Data.Account

	if accountStream.Type == "migration_database" {
		err = am.as.RemoveDuplicateAccountMigration(ctx, accountStream.AccountNumber)
		if err != nil {
			err = fmt.Errorf("unable to remove duplicate account: %w", err)
			logField = append(logField, xlog.Err(err))
			xlog.Warn(ctx, logMessage, logField...)
			return err
		}
	}

	var legacyId *models.AccountLegacyId
	if accountStream.LegacyId != nil {
		acuanAccountLegacy := models.AccountLegacyId(*accountStream.LegacyId)
		legacyId = &acuanAccountLegacy
	}

	var metadata models.AccountMetadata
	if accountStream.Metadata != nil {
		if metadataMap, ok := accountStream.Metadata.(map[string]any); ok {
			metadata = metadataMap
		} else {
			logField = append(logField, xlog.Err(fmt.Errorf("metadata is not a map[string]any")))
			xlog.Warn(ctx, logMessage, logField...)
		}
	}

	action := "upsert"
	if am.cfg.FeatureFlag.EnablePreventSameAccountMutationActing {
		action = "insert"
		var accountExist models.GetAccountOut

		// Check exist
		accountExist, err = am.as.GetOneByAccountNumber(ctx, accountStream.AccountNumber)
		if err == nil {
			err = fmt.Errorf("account is exist: %d", accountExist.ID)
			logField = append(logField, xlog.Err(err))
			xlog.Warn(ctx, logMessage, logField...)
			return err
		}

		_, err = am.as.Create(ctx, models.CreateAccount{
			AccountNumber:   accountStream.AccountNumber,
			Name:            accountStream.Name,
			ProductTypeName: accountStream.ProductTypeName,
			OwnerID:         accountStream.OwnerId,
			CategoryCode:    accountStream.CategoryCode,
			SubCategoryCode: accountStream.SubCategoryCode,
			EntityCode:      accountStream.EntityCode,
			Currency:        accountStream.Currency,
			AltId:           accountStream.AltId,
			LegacyId:        legacyId,
			Status:          accountStream.Status,
			Metadata:        metadata,
		})
	} else {
		err = am.as.Upsert(ctx, models.AccountUpsert{
			AccountNumber:   accountStream.AccountNumber,
			Name:            accountStream.Name,
			ProductTypeName: accountStream.ProductTypeName,
			OwnerID:         accountStream.OwnerId,
			CategoryCode:    accountStream.CategoryCode,
			SubCategoryCode: accountStream.SubCategoryCode,
			EntityCode:      accountStream.EntityCode,
			Currency:        accountStream.Currency,
			AltID:           accountStream.AltId,
			LegacyId:        legacyId,
			Status:          accountStream.Status,
			Metadata:        metadata,
		})
	}
	if err != nil {
		err = fmt.Errorf("unable to %s account: %w", action, err)
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
		return err
	}

	xlog.Info(ctx, logMessage, logField...)
	return nil
}

func (am AccountMutationHandler) handler(ctx context.Context, message *sarama.ConsumerMessage) (err error) {
	startTime := time.Now() // time when a process consumes a message started
	err = am.processMessage(ctx, message)

	if am.consumerMetrics != nil {
		am.consumerMetrics.GenerateMetrics(startTime, message, err)
	}

	return
}

func (am AccountMutationHandler) Ack(session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage) {
	session.MarkMessage(message, "")
}

// Nack is a custom function for handling failed messages during Kafka consumer processing.
// It publishes the failed message to a DLQ and mark the message as consumed.
func (am AccountMutationHandler) Nack(ctx context.Context, session sarama.ConsumerGroupSession, message *sarama.ConsumerMessage, causeErr error) {
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
