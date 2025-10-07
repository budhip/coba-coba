package report

import (
	"context"
	"strings"
	"time"

	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/flag"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
)

type reportHandler struct {
	transactionSrv services.TransactionService
	reconSrv       services.ReconService
}

func Routes(ts services.TransactionService, rs services.ReconService) map[string]func(ctx context.Context, date time.Time, flag flag.Job) error {
	handler := reportHandler{
		transactionSrv: ts,
		reconSrv:       rs,
	}
	return map[string]func(ctx context.Context, date time.Time, flag flag.Job) error{
		"GenerateTransactionReport": handler.GenerateTransactionReport,
		"DoBalanceReconDaily":       handler.DoBalanceReconDaily,
		// add more job here
	}
}

func (rh *reportHandler) GenerateTransactionReport(ctx context.Context, date time.Time, flag flag.Job) error {
	urls, err := rh.transactionSrv.GenerateTransactionReport(ctx)
	if err != nil {
		return err
	}
	xlog.Info(ctx, "GenerateTransactionReport", xlog.String("urls", strings.Join(urls, ",")))

	return nil
}

func (rh *reportHandler) DoBalanceReconDaily(ctx context.Context, date time.Time, flag flag.Job) error {
	url, err := rh.reconSrv.DoDailyBalance(ctx)
	if err != nil {
		return err
	}

	xlog.Info(ctx, "DoBalanceReconDaily", xlog.String("url", url))

	return nil
}
