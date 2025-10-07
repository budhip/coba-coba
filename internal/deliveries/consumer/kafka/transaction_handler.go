package kafkaconsumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/retry"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/transaction_notification"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	goacuanlib "bitbucket.org/Amartha/go-acuan-lib/model"
	xlog "bitbucket.org/Amartha/go-x/log"
	"bitbucket.org/Amartha/go-x/log/audit"
	"bitbucket.org/Amartha/go-x/log/ctxdata"

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TransactionHandler struct {
	clientId                string
	ts                      services.TransactionService
	accountSvc              services.AccountService
	dlq                     dlqpublisher.Publisher
	journal                 JournalPublisher // TODO: Remove unused
	transactionNotification transaction_notification.TransactionNotificationPublisher
	ebRetry                 retry.Retryer
	featureFlag             config.FeatureFlag
	consumerMetrics         *metrics.ConsumerMetrics
}

type ackPayload struct {
	message    *sarama.ConsumerMessage
	acuanOrder goacuanlib.Payload[goacuanlib.DataOrder]
}

func NewTransactionHandler(
	clientId string,
	ts services.TransactionService,
	dlq dlqpublisher.Publisher,
	ebr retry.Retryer,
	journal JournalPublisher,
	accountSvc services.AccountService,
	featureFlag config.FeatureFlag,
	transactionNotification transaction_notification.TransactionNotificationPublisher,
	consumerMetrics *metrics.ConsumerMetrics) sarama.ConsumerGroupHandler {
	return &TransactionHandler{
		clientId:                clientId,
		ts:                      ts,
		accountSvc:              accountSvc,
		dlq:                     dlq,
		ebRetry:                 ebr,
		journal:                 journal,
		transactionNotification: transactionNotification,
		featureFlag:             featureFlag,
		consumerMetrics:         consumerMetrics,
	}
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (th TransactionHandler) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (th TransactionHandler) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim must start a consumer loop of ConsumerGroupClaim's Messages().
func (th TransactionHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				return nil
			}

			ctx := ctxdata.Sets(session.Context(),
				ctxdata.SetCorrelationId(uuid.New().String()),
				ctxdata.SetHost(th.clientId),
			)
			start := time.Now()
			logField := createLogField(message)

			ackPld := &ackPayload{
				message: message,
			}

			acuanOrder, err := th.parseMessageToAcuanOrder(ctx, message)
			if err != nil {
				logField = append(logField, xlog.Duration("response-time", time.Since(start)), xlog.Err(err))
				xlog.Warn(ctx, logMessage, logField...)
				th.Nack(ctx, session, ackPld, err)
				continue
			}

			ackPld.acuanOrder = *acuanOrder

			var operationErr error
			operation := func() error {
				operationErr = th.handler(ctx, message, acuanOrder)
				if operationErr != nil {
					logField = append(logField, xlog.Duration("response-time", time.Since(start)), xlog.Err(operationErr))
					xlog.Warn(ctx, logMessage, logField...)

					if errors.Is(operationErr, common.ErrOrderAlreadyExists) ||
						errors.Is(operationErr, common.ErrOrderContainExcludeInsertDB) {
						return th.ebRetry.StopRetryWithErr(operationErr)
					}

					return operationErr
				}
				return nil
			}
			dlqCallback := func() error {
				th.Nack(ctx, session, ackPld, operationErr)
				return operationErr
			}

			if err = th.ebRetry.Retry(ctx, operation, dlqCallback); err != nil {
				logField = append(logField, xlog.Duration("response-time", time.Since(start)), xlog.Err(err))
				xlog.Warn(ctx, logMessage, logField...)
				continue
			}

			logField = append(logField, xlog.Duration("response-time", time.Since(start)))
			xlog.Info(ctx, logMessage, logField...)
			audit.Info(ctx, audit.Message{ActivityData: string(message.Value)})
			th.Ack(ctx, session, ackPld)
		case <-session.Context().Done():
			return nil
		}
	}
}

func (th TransactionHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage, acuanOrder *goacuanlib.Payload[goacuanlib.DataOrder]) error {
	const logMessage = "[PROCESS-MESSAGE]"

	logField := append(
		createLogField(message),
		xlog.String("source-system", acuanOrder.Headers.SourceSystem),
		xlog.Any("request", acuanOrder),
	)

	order := acuanOrder.Body.Data.Order

	var req []models.TransactionReq
	for _, v := range acuanOrder.Body.Data.Order.Transactions {
		var transactionID string
		if v.Id != nil {
			transactionID = v.Id.String()
		}
		orderReq := models.TransactionReq{
			TransactionID:   transactionID,
			FromAccount:     v.SourceAccountId,
			ToAccount:       v.DestinationAccountId,
			FromNarrative:   "",
			ToNarrative:     "",
			TransactionDate: common.FormatDatetimeToStringInLocalTime(*v.TransactionTime.Time, common.DateFormatYYYYMMDD),
			Amount:          decimal.NewNullDecimal(v.Amount),
			Status:          fmt.Sprint(v.Status),
			Method:          string(v.Method),
			TypeTransaction: string(v.TransactionType),
			Description:     v.Description,
			RefNumber:       order.RefNumber,
			Metadata:        v.Meta,
			OrderTime:       *order.OrderTime.Time,
			OrderType:       string(order.OrderType),
			TransactionTime: *v.TransactionTime.Time,
			Currency:        v.Currency,
		}
		req = append(req, orderReq)
	}
	if err := th.ts.NewStoreBulkTransaction(ctx, req); err != nil {
		if errors.Is(err, common.ErrOrderAlreadyExists) || errors.Is(err, common.ErrOrderContainExcludeInsertDB) {
			return err
		}

		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
		return fmt.Errorf("error when StoreBulkTransaction: %w", err)
	}
	xlog.Info(ctx, logMessage, logField...)

	return nil
}

func (am TransactionHandler) handler(ctx context.Context, message *sarama.ConsumerMessage, acuanOrder *goacuanlib.Payload[goacuanlib.DataOrder]) (err error) {
	startTime := time.Now() // time when a process consumes a message started
	err = am.processMessage(ctx, message, acuanOrder)

	if am.consumerMetrics != nil {
		am.consumerMetrics.GenerateMetrics(startTime, message, err)
	}

	return
}

func (th TransactionHandler) parseMessageToAcuanOrder(ctx context.Context, msg *sarama.ConsumerMessage) (*goacuanlib.Payload[goacuanlib.DataOrder], error) {
	var (
		payload    goacuanlib.Payload[goacuanlib.DataOrder]
		logMessage = "[PROCESS-MESSAGE]"
	)

	logField := createLogField(msg)

	if err := json.Unmarshal(msg.Value, &payload); err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
		return nil, fmt.Errorf("error unmarshal json: %w", err)
	}

	if th.featureFlag.EnableConsumerValidationReject {
		for _, v := range payload.Body.Data.Order.Transactions {
			if v.Amount.IsNegative() {
				return nil, fmt.Errorf("amount %s cannot negative: %s", v.Id.String(), v.Amount.String())
			}
		}
	}

	if th.featureFlag.EnableCheckAccountTransaction {
		for i, v := range payload.Body.Data.Order.Transactions {
			fromAccount, _ := th.accountSvc.GetACuanAccountNumber(ctx, v.SourceAccountId)
			payload.Body.Data.Order.Transactions[i].SourceAccountId = fromAccount

			toAccount, _ := th.accountSvc.GetACuanAccountNumber(ctx, v.DestinationAccountId)
			payload.Body.Data.Order.Transactions[i].DestinationAccountId = toAccount
		}
	}

	return &payload, nil
}

func (th TransactionHandler) Ack(ctx context.Context, session sarama.ConsumerGroupSession, payload *ackPayload) {
	if th.featureFlag.EnablePublishTransactionNotification {
		err := th.publishTransactionNotification(ctx, payload.acuanOrder, nil)
		if err != nil {
			logField := append(createLogField(payload.message), xlog.Err(err))
			xlog.Warn(ctx, logMessage, logField...)
		}
	}

	session.MarkMessage(payload.message, "")
}

// Nack is a custom function for handling failed messages during Kafka consumer processing.
// It publishes the failed message to a DLQ and mark the message as consumed.
func (th TransactionHandler) Nack(ctx context.Context, session sarama.ConsumerGroupSession, payload *ackPayload, causeErr error) {
	logField := createLogField(payload.message)

	if th.featureFlag.EnablePublishTransactionNotification {
		err := th.publishTransactionNotification(ctx, payload.acuanOrder, causeErr)
		if err != nil {
			logField = append(logField, xlog.Err(err))
			xlog.Warn(ctx, logMessage, logField...)
		}
	}

	if errors.Is(causeErr, common.ErrOrderAlreadyExists) ||
		errors.Is(causeErr, common.ErrOrderContainExcludeInsertDB) {
		// don't send to DLQ if order already exists
		session.MarkMessage(payload.message, "")
		return
	}

	err := th.dlq.Publish(models.FailedMessage{
		Payload:    payload.message.Value,
		Timestamp:  payload.message.Timestamp,
		CauseError: causeErr,
	})
	if err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMessage, logField...)
	}

	session.MarkMessage(payload.message, "")
}

func (th TransactionHandler) publishTransactionNotification(ctx context.Context, acuanOrder goacuanlib.Payload[goacuanlib.DataOrder], operationErr error) error {
	status := models.StatusTransactionNotificationSuccess
	message := "success consume transaction"

	if operationErr != nil {
		if errors.Is(operationErr, common.ErrOrderAlreadyExists) {
			status = models.StatusTransactionNotificationSkipped
			message = "skipped consume transaction, due to order already exists on DB"
		} else if errors.Is(operationErr, common.ErrOrderContainExcludeInsertDB) {
			status = models.StatusTransactionNotificationSkipped
			message = "skipped consume transaction, due to order contain exclude insert DB"
		} else {
			status = models.StatusTransactionNotificationFailed
			message = "failed consume transaction " + operationErr.Error()
		}
	}

	return th.transactionNotification.Publish(ctx, models.TransactionNotificationPayload{
		Identifier: acuanOrder.Body.Data.Order.RefNumber,
		Status:     status,
		AcuanData:  acuanOrder,
		Message:    message,
		ClientID:   th.clientId,
	})
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
