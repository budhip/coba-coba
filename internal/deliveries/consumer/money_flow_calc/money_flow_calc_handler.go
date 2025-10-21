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
		logMsg = "[PROCESS-MESSAGE]"
	)

	logField := mfc.CreateLogField(message)

	var rawNotif models.TransactionNotificationRaw
	if err = json.Unmarshal(message.Value, &rawNotif); err != nil {
		logField = append(logField, xlog.Err(err))
		xlog.Warn(ctx, logMsg, logField...)
		return fmt.Errorf("error unmarshal json to raw: %w", err)
	}

	fixedAcuanData := mfc.fixAmountInJSON(rawNotif.AcuanData)

	var notification goacuanlib.Payload[goacuanlib.DataOrder]

	if len(fixedAcuanData) > 0 {
		var acuanData goacuanlib.Payload[goacuanlib.DataOrder]
		if err = json.Unmarshal(fixedAcuanData, &acuanData); err != nil {
			logField = append(logField, xlog.Err(err))
			xlog.Warn(ctx, logMsg, logField...)
			return fmt.Errorf("error unmarshal acuan data: %w", err)
		}
		notification = acuanData
	}

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
	mfc.RecordMetrics(startTime, message, err)
	return
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
