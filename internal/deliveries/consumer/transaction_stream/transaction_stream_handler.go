package transaction_stream

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	dlqpublisher "bitbucket.org/Amartha/go-fp-transaction/internal/common/dlq_publisher"
	kafkacommon "bitbucket.org/Amartha/go-fp-transaction/internal/common/kafka"
	gopaymentlib "bitbucket.org/Amartha/go-payment-lib/payment-api/models/event"
	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/metrics"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	"bitbucket.org/Amartha/go-x/log/audit"
	"bitbucket.org/Amartha/go-x/log/ctxdata"

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
)

var validStatuses = map[string]bool{
	"SUCCESSFUL": true,
	"REJECTED":   true,
}

type TransactionStreamHandler struct {
	kafkacommon.BaseHandler
	mfs services.MoneyFlowService
	cfg config.Config
}

func NewTransactionStreamHandler(
	clientId string,
	mfs services.MoneyFlowService,
	dlq dlqpublisher.Publisher,
	cfg config.Config,
	consumerMetrics *metrics.ConsumerMetrics,
) sarama.ConsumerGroupHandler {
	return &TransactionStreamHandler{
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

func (tsh TransactionStreamHandler) Setup(session sarama.ConsumerGroupSession) error {
	xlog.Info(
		context.Background(),
		logMessage+"[SETUP]",
		xlog.String("member_id", session.MemberID()),
		xlog.Int32("generation_id", session.GenerationID()),
	)
	return nil
}

func (tsh TransactionStreamHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	xlog.Info(
		context.Background(),
		logMessage+"[CLEANUP]",
		xlog.String("member_id", session.MemberID()),
	)
	return nil
}

func (tsh TransactionStreamHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	xlog.Info(
		session.Context(),
		logMessage+"[CONSUME-CLAIM-START]",
		xlog.String("topic", claim.Topic()),
		xlog.Int32("partition", claim.Partition()),
		xlog.Int64("initial_offset", claim.InitialOffset()),
	)

	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				xlog.Info(session.Context(), logMessage+"[CONSUME-CLAIM-CLOSED]")
				return nil
			}

			ctx := ctxdata.Sets(session.Context(),
				ctxdata.SetCorrelationId(uuid.New().String()),
				ctxdata.SetHost(tsh.ClientID),
			)

			start := time.Now()
			logField := tsh.CreateLogField(message)

			err := tsh.handler(ctx, message)

			if err != nil {
				logField = append(logField,
					xlog.Duration("response-time", time.Since(start)),
					xlog.Err(err),
					xlog.String("status", "failed"),
				)
				xlog.Warn(ctx, logMessage+"[PROCESS-FAILED]", logField...)

				tsh.Nack(ctx, session, message, err)
				continue
			}

			logField = append(logField,
				xlog.Duration("response-time", time.Since(start)),
				xlog.String("status", "success"),
			)
			xlog.Info(ctx, logMessage+"[PROCESS-SUCCESS]", logField...)

			audit.Info(ctx, audit.Message{ActivityData: string(message.Value)})

			tsh.Ack(session, message)

		case <-session.Context().Done():
			xlog.Info(session.Context(), logMessage+"[CONSUME-CLAIM-CONTEXT-DONE]")
			return nil
		}
	}
}

func (tsh TransactionStreamHandler) processMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	var (
		logMsg = logMessage + "[PROCESS-MESSAGE]"
	)

	logField := tsh.CreateLogField(message)

	if len(message.Value) == 0 {
		xlog.Warn(ctx, logMsg, append(logField, xlog.String("error", "empty message"))...)
		return fmt.Errorf("empty message received")
	}

	var transactionEvent gopaymentlib.Event
	if err := json.Unmarshal(message.Value, &transactionEvent); err != nil {
		logField = append(logField, xlog.Err(err), xlog.String("raw_message", string(message.Value)))
		xlog.Warn(ctx, logMsg, logField...)
		return fmt.Errorf("error unmarshal json: %w", err)
	}

	if transactionEvent.ID == "" {
		xlog.Warn(ctx, logMsg, append(logField, xlog.String("error", "missing transaction_id"))...)
		return fmt.Errorf("papa_transaction_id is required")
	}

	if transactionEvent.PaymentType == "" {
		xlog.Warn(ctx, logMsg, append(logField, xlog.String("error", "missing payment_type"))...)
		return fmt.Errorf("payment_type is required")
	}

	status := transactionEvent.Status.ConvertSingleAPI().ToString()
	if !validStatuses[status] {
		xlog.Info(ctx, logMsg, append(logField,
			xlog.String("status", status),
			xlog.String("papa_transaction_id", transactionEvent.ID),
			xlog.String("payment_type", transactionEvent.PaymentType.ConvertSingleAPI().ToString()),
			xlog.String("info", "skipping invalid transaction status"),
		)...)
		return nil
	}

	xlog.Info(ctx, logMsg, append(logField,
		xlog.String("papa_transaction_id", transactionEvent.ID),
		xlog.String("payment_type", transactionEvent.PaymentType.ConvertSingleAPI().ToString()),
		xlog.String("status", status),
		xlog.String("info", "processing successful/rejected transaction"),
	)...)

	err := tsh.mfs.ProcessTransactionStream(ctx, transactionEvent)
	if err != nil {
		err = fmt.Errorf("unable to process transaction stream: %w", err)
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMsg, logField...)
		return err
	}

	xlog.Info(ctx, logMsg+" [SUCCESS]", append(logField,
		xlog.String("papa_transaction_id", transactionEvent.ID),
		xlog.String("payment_type", transactionEvent.PaymentType.ConvertSingleAPI().ToString()),
	)...)

	return nil
}

func (tsh TransactionStreamHandler) handler(ctx context.Context, message *sarama.ConsumerMessage) (err error) {
	startTime := time.Now()
	err = tsh.processMessage(ctx, message)
	tsh.RecordMetrics(startTime, message, err)
	return err
}
