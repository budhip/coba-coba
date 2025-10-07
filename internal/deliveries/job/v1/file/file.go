package v1file

import (
	"context"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/flag"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	xlog "bitbucket.org/Amartha/go-x/log"
)

type fileHandler struct {
	fileSrv services.FileService
}

func Routes(fs services.FileService) map[string]func(ctx context.Context, date time.Time, flag flag.Job) error {
	handler := fileHandler{fileSrv: fs}
	return map[string]func(ctx context.Context, date time.Time, flag flag.Job) error{
		"DoUploadTransactionWallet": handler.DoUploadWalletTransaction,
	}
}

func (fh *fileHandler) DoUploadWalletTransaction(ctx context.Context, date time.Time, flag flag.Job) (err error) {
	err = fh.fileSrv.UploadWalletTransactionFromGCS(ctx, flag.FileName, flag.BucketName, flag.JobName, flag.FlagPublishAcuan)
	if err != nil {
		return err
	}

	xlog.Info(ctx, "DoUploadWalletTransaction", xlog.String("file name", flag.FileName))

	return nil
}
