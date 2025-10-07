package services

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

func (ts *transaction) PublishTransaction(ctx context.Context, in models.DoPublishTransactionRequest) (out models.DoPublishTransactionResponse, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	if in.RefNumber == "" {
		in.RefNumber = ts.srv.idgenerator.Generate(models.TransactionIDManualPrefix)
	}

	req, err := in.ValidateToRequest()
	if err != nil {
		return
	}

	if err = ts.srv.acuanClient.PublishTransaction(ctx, req); err != nil {
		return
	}

	out = in.ToPublishResponse()

	return
}
