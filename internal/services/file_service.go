package services

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/csv"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/ddd_notification"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type FileService interface {
	Upload(ctx context.Context, file *multipart.FileHeader) error
	UploadWalletTransaction(ctx context.Context, file *multipart.FileHeader, reportTo, clientID string) error
	UploadWalletTransactionFromGCS(ctx context.Context, filePath, bucketName, clientID string, isPublish bool) (err error)
}

type file service

var _ FileService = (*file)(nil)

// Upload implements the FileService interface for uploading files.
func (s *file) Upload(ctx context.Context, file *multipart.FileHeader) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	reportNumber, nowDate, err := s.getReportNumberFromCache(ctx, models.TransactionIDManualPrefix)
	if err != nil {
		return err
	}

	// Iterate over each line and process it
	lineNum := 0 // 0 is header
	for res := range s.srv.fileRepo.StreamReadMultipartFile(ctx, file) {
		if res.Err != nil {
			err = res.Err
			return err
		}

		refNumber := fmt.Sprintf("%03d-%s-%s-%d", reportNumber, models.TransactionIDManualPrefix, nowDate, lineNum)
		lineNum++
		err = s.publishOrderToACuan(ctx, res.Data, refNumber)
		if err != nil {
			s.logProcessError(ctx, refNumber, err)
			continue
		}
	}

	return nil
}

// Template: https://docs.google.com/spreadsheets/d/1m5XSQK6evLlTwJNL4l86xsmjotkN_oIAS647OZgHEvk
func (s *file) UploadWalletTransaction(ctx context.Context, file *multipart.FileHeader, reportTo, clientID string) error {
	var (
		err    error
		header = []string{
			"Transaction Date",
			"Reference Number",
			"Transaction Flow",
			"Transaction Type",
			"Account Number",
			"Amount",
			"Destination Account Number",
			"Description",
			"Metadata",
			"Child Transaction Type",
			"Child Amount",
			"Error Message",
		}
		hasFailedData = false
	)

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	reportNumber, nowDate, err := s.getReportNumberFromCache(ctx, models.WalletTransactionIDManualPrefix)
	if err != nil {
		return err
	}

	var fileBuffer bytes.Buffer
	w := csv.NewWriter(&fileBuffer)

	err = w.Write(header)
	if err != nil {
		err = fmt.Errorf("failed to write header: %w", err)
		return err
	}

	// Iterate over each line and process it
	lineNum := 0 // 0 is header
	for csvRow := range s.srv.fileRepo.StreamReadMultipartFile(ctx, file) {
		if csvRow.Err != nil {
			err = csvRow.Err
			return err
		}

		// Parse data into wallet transaction
		refNumber := fmt.Sprintf("%s-%s-%03d-%d", models.WalletTransactionIDManualPrefix, nowDate, reportNumber, lineNum)
		lineNum++
		var record []string
		// Skip EOF or header
		if csvRow.Data == "" || s.isWalletTransactionCSVHeader(csvRow.Data) {
			continue
		}

		reader := csv.NewReader(strings.NewReader(csvRow.Data))
		reader.LazyQuotes = true // fix: error bare " in non-quoted-field; meta using double-quote

		record, err = reader.Read()
		if err != nil {
			err = fmt.Errorf("failed to read csv row: %w", err)
			return err
		}

		var input models.NewWalletTransaction
		input, err = s.parseWalletTransactionData(refNumber, record)
		if err != nil {
			dataTemporary := append(record, err.Error())
			w.Write(dataTemporary)
			hasFailedData = true
			continue
		}

		// Validate transaction types
		trxTypes := []string{input.TransactionType}
		for _, v := range input.Amounts {
			trxTypes = append(trxTypes, v.Type)
		}

		if err = s.srv.MasterData.EnsureTransactionTypeExist(ctx, trxTypes); err != nil {
			dataTemporary := append(record, err.Error())
			w.Write(dataTemporary)
			hasFailedData = true
			continue
		}

		if input.AccountNumber != "" {
			// safe to ignore error, since we use input account number if not exists in legacyId
			input.AccountNumber, _ = s.srv.Account.GetACuanAccountNumber(ctx, input.AccountNumber)
		}

		if input.DestinationAccountNumber != "" {
			input.DestinationAccountNumber, _ = s.srv.Account.GetACuanAccountNumber(ctx, input.DestinationAccountNumber)
		}

		// Insert
		_, err = s.srv.WalletTrx.CreateTransactionAtomic(ctx, input, false, true, clientID)
		if err != nil {
			dataTemporary := append(record, err.Error())
			w.Write(dataTemporary)
			hasFailedData = true
			continue
		}
	}

	if hasFailedData {
		w.Flush()
		if err = w.Error(); err != nil {
			err = fmt.Errorf("failed to flush writer: %w", err)
			return err
		}

		s.sendErrorToEmail(ctx, fileBuffer, reportTo)
	}

	return nil
}

func (s *file) sendErrorToEmail(ctx context.Context, fileBuffer bytes.Buffer, reportTo string) error {
	return s.srv.dddNotification.SendEmail(ctx, ddd_notification.RequestEmail{
		From:     "noreply@amartha.com",
		FromName: "Amartha",
		Subject:  "ERROR UPLOAD MANUAL TRANSACTION",
		To:       reportTo,
		Template: "2024-mis-internal",
		CC: []ddd_notification.Cc{
			{
				Email: "finance.platform@amartha.com",
			},
		},
		Attachments: []ddd_notification.Attachment{
			{
				Type:    "text/csv",
				Name:    "error_result.csv",
				Content: base64.StdEncoding.EncodeToString(fileBuffer.Bytes()),
			},
		},
		Subs: []any{
			map[string]any{
				"foo": "bar",
			},
		},
	})
}

func (s *file) logProcessError(ctx context.Context, refNumber string, processErr error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(processErr))

	username, ok := ctx.Value(models.CtxKeyNgmisHeader).(string)
	if !ok {
		username = "-"
	}

	xlog.Error(ctx, "[FILE-ERROR]",
		xlog.String("operation", "Process Manual Upload Transaction"),
		xlog.String("user_ngmis", username),
		xlog.String("ref_number", refNumber),
		xlog.String("error_message", processErr.Error()))
}

func (s *file) WriteCsvFile(ctx context.Context, filePath, bucketName string, errorRows <-chan models.ErrWalletTransaction) {
	chanData := make(chan []byte)

	filename := filepath.Base(filePath)
	pathFile := filepath.Dir(filePath)
	recordFailed := 0

	go func() {
		defer close(chanData)
		for v := range errorRows {
			select {
			case <-ctx.Done():
				return
			default:
				recordFailed++
				chanData <- []byte(fmt.Sprintf("%s\n", strings.Join([]string{v.LineNumb, v.AccountNumber, v.RefNumber, v.TransactionType, v.Error}, models.CSV_SEPARATOR)))
			}
		}
	}()

	gcsPayload := models.CloudStoragePayload{
		Filename: fmt.Sprintf("error_%s", filename),
		Path:     fmt.Sprintf("%s/error", pathFile),
	}
	r := s.srv.cloudStorage.WriteStreamCustomBucket(ctx, bucketName, &gcsPayload, chanData)
	_, errWait := r.Wait()
	if errWait != nil {
		fmt.Printf("got error write stream %v", errWait)
	}

	fmt.Printf("\n === Process have %v failed record %v === \n", recordFailed, "")
	return
}

func (s *file) getReportNumberFromCache(ctx context.Context, cachePrefix string) (reportNumber int, nowDate string, err error) {
	nowDate = common.Now().Format(common.DateFormatDDMMYYYYWithoutDash)

	// generate refNumber from redis
	cacheKey := fmt.Sprintf("%s-%s", cachePrefix, nowDate)
	cacheVal, err := s.srv.cacheRepo.Get(ctx, cacheKey)
	if err != nil {
		if !errors.Is(err, common.ErrDataNotFound) {
			return
		}
		reportNumber = 1
	} else {
		reportNumber, err = strconv.Atoi(cacheVal)
		if err != nil {
			return
		}
		reportNumber++
	}

	// save to redis
	durationUntilEOD := time.Until(common.NowEndOfDay())
	err = s.srv.cacheRepo.Set(ctx, cacheKey, reportNumber, durationUntilEOD)
	if err != nil {
		return
	}

	return
}

func (s *file) UploadWalletTransactionFromGCS(ctx context.Context, filePath, bucketName, clientID string, isPublish bool) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	errChannels := make(chan models.ErrWalletTransaction, 0)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.WriteCsvFile(ctx, filePath, bucketName, errChannels)
	}()

	reader, errReader := s.srv.cloudStorage.NewReaderBucketCustom(ctx, bucketName, filePath)
	if errReader != nil {
		return errReader
	}

	reportNumber, nowDate, err := s.getReportNumberFromCache(ctx, models.WalletTransactionIDManualPrefix)
	if err != nil {
		return err
	}

	// Iterate over each line and process it
	lineNum := 0 // 0 is header
	for csvRow := range s.srv.fileRepo.StreamReadCSVFile(ctx, reader) {
		if csvRow.Err != nil {
			err = csvRow.Err
			return err
		}

		if strings.HasPrefix(csvRow.Data[0], "Transaction Date") {
			continue
		}

		// Parse data into wallet transaction
		refNumber := fmt.Sprintf("%s-%s-%03d-%d", models.WalletTransactionIDManualPrefix, nowDate, reportNumber, lineNum)
		lineNum++
		//Count Row
		fmt.Printf("RUNNING PROCESS ROW-%d\n", lineNum)

		var input models.NewWalletTransaction
		input, err = s.parseWalletTransactionData(refNumber, csvRow.Data)
		if err != nil {
			if errors.Is(common.ErrNoRows, err) || errors.Is(common.ErrCSVRowIsEmpty, err) {
				continue
			}

			errRef := input.RefNumber
			if errRef == "" {
				errRef = refNumber
			}

			errChannels <- s.parseErrorWalletTransaction(csvRow.Data, lineNum, err.Error())
			continue
		}

		// Validate transaction types
		trxTypes := []string{input.TransactionType}
		for _, v := range input.Amounts {
			trxTypes = append(trxTypes, v.Type)
		}

		if err = s.srv.MasterData.EnsureTransactionTypeExist(ctx, trxTypes); err != nil {
			errChannels <- s.parseErrorWalletTransaction(csvRow.Data, lineNum, err.Error())
			continue
		}

		if input.AccountNumber != "" {
			// safe to ignore error, since we use input account number if not exists in legacyId
			input.AccountNumber, _ = s.srv.Account.GetACuanAccountNumber(ctx, input.AccountNumber)
		}

		if input.DestinationAccountNumber != "" {
			input.DestinationAccountNumber, _ = s.srv.Account.GetACuanAccountNumber(ctx, input.DestinationAccountNumber)
		}

		// Insert
		_, err = s.srv.WalletTrx.CreateTransactionAtomic(ctx, input, false, isPublish, clientID)
		if err != nil {
			errChannels <- s.parseErrorWalletTransaction(csvRow.Data, lineNum, err.Error())
			continue
		}
	}
	fmt.Printf("FINISH PROCESS %d DATA\n", lineNum)

	close(errChannels)
	wg.Wait()
	return nil
}
