package services

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	localstorage "bitbucket.org/Amartha/go-fp-transaction/internal/common/local_storage"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

	xlog "bitbucket.org/Amartha/go-x/log"
)

type storageRecon localstorage.LocalStorage[[]models.ReconRecord]

func (s *reconService) ProcessReconTaskQueue(ctx context.Context, reconHistoryId uint64) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	reconHistory, err := s.prepareReconHistory(ctx, reconHistoryId)
	if err != nil {
		return err
	}

	// Log recon start
	xlog.Info(ctx, "[RECON-INFO]", 
		xlog.String("operation", "Start process for recon data"),
		xlog.Uint64("recon_history_id", reconHistoryId))

	ls, err := s.createLocalStorage()
	if err != nil {
		return err
	}
	defer s.closeLocalStorage(ls)

	err = s.streamTransactionsToLocalStorage(ctx, ls, *reconHistory)
	if err != nil {
		return err
	}

	resultFilePath, err := s.reconcileRecordsAndGenerateReport(ctx, reconHistory, ls)
	if err != nil {
		errUpdateStatus := s.updateReconHistoryStatus(ctx, reconHistory, models.ReconHistoryStatusFailed, "")
		if errUpdateStatus != nil {
			xlog.Warn(ctx, "[SERVICE]", xlog.String("status", "error"), xlog.Err(errUpdateStatus))
		}

		return err
	}

	err = s.updateReconHistoryStatus(ctx, reconHistory, models.ReconHistoryStatusSuccess, resultFilePath)
	if err != nil {
		return err
	}

	// Log recon completion
	xlog.Info(ctx, "[RECON-INFO]", 
		xlog.String("operation", "Finish process for recon data"),
		xlog.Uint64("recon_history_id", reconHistoryId))

	return nil
}

func (s *reconService) createLocalStorage() (storageRecon, error) {
	ls, err := localstorage.NewBadgerStorage[[]models.ReconRecord]("reconTool")
	if err != nil {
		return nil, fmt.Errorf("failed to make local storage: %w", err)
	}
	return ls, nil
}

func (s *reconService) closeLocalStorage(ls storageRecon) {
	ls.Close()
	ls.Clean()
}

func (s *reconService) prepareReconHistory(ctx context.Context, reconHistoryId uint64) (*models.ReconToolHistory, error) {
	reconHistory, err := s.srv.sqlRepo.GetReconToolHistoryRepository().GetById(ctx, reconHistoryId)
	if err != nil {
		return nil, err
	}

	if reconHistory.TransactionDate == nil {
		return nil, errors.New("transaction date is empty")
	}

	reconHistory.Status = models.ReconHistoryStatusProcessing
	_, err = s.srv.sqlRepo.GetReconToolHistoryRepository().Update(ctx, reconHistoryId, reconHistory)
	if err != nil {
		return nil, err
	}

	return reconHistory, nil
}

func (s *reconService) streamTransactionsToLocalStorage(ctx context.Context, ls storageRecon, reconHistory models.ReconToolHistory) error {
	repoTransaction := s.srv.sqlRepo.GetTransactionRepository()

	chanTrx := repoTransaction.StreamAll(ctx, models.TransactionFilterOptions{
		TransactionDate: reconHistory.TransactionDate,
		TransactionType: reconHistory.TransactionType,
	})
	for trx := range chanTrx {
		if trx.Err != nil {
			return fmt.Errorf("failed to stream transaction: %w", trx.Err)
		}

		rr, err := models.ConvertTransactionTopUpToReconRecord(trx.Data)
		if err != nil {
			return fmt.Errorf("failed to convert transaction to recon record: %w", err)
		}

		records, err := ls.Get(rr.Identifier)
		if err != nil {
			return fmt.Errorf("failed to get value from localstorage: %w", err)
		}

		records = append(records, *rr)

		err = ls.Set(rr.Identifier, records)
		if err != nil {
			return fmt.Errorf("failed to set value to localstorage: %w", err)
		}
	}

	return nil
}

func (s *reconService) reconcileRecordsAndGenerateReport(ctx context.Context, reconHistory *models.ReconToolHistory, ls storageRecon) (string, error) {
	repoGCS := s.srv.cloudStorage

	gcsUploadedFilePayload := models.NewCloudStoragePayload(reconHistory.UploadedFilePath)
	fileReader, err := repoGCS.NewReader(ctx, &gcsUploadedFilePayload)
	if err != nil {
		return "", fmt.Errorf("failed to read csv file: %w", err)
	}
	defer fileReader.Close()

	now := time.Now()
	gcsResultFilePayload := &models.CloudStoragePayload{
		Filename: fmt.Sprintf("%s.csv", now.Format(common.DateFormatYYYYMMDDHHMMSSWithoutDash)),
		Path:     fmt.Sprintf("%s/result/%04d/%02d", models.ReconToolFolderName, now.Year(), now.Month()),
	}
	resultFile := repoGCS.NewWriter(ctx, gcsResultFilePayload)
	defer func() {
		// make sure, if file cannot be written, it will return an error
		err = resultFile.Close()
	}()

	reportFile := csv.NewWriter(resultFile)
	defer reportFile.Flush()

	err = reportFile.Write(models.CSVHeaderReconRecord)
	if err != nil {
		return "", fmt.Errorf("failed to write header to file: %w", err)
	}

	csvReader := csv.NewReader(fileReader)
	for {
		row, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("failed to read csv row: %w", err)
		}

		err = s.processCSVRow(row, reconHistory, ls, reportFile)
		if err != nil {
			return "", fmt.Errorf("failed to process csv row: %w", err)
		}
	}

	// check if there is any record in localstorage that not exists in csv
	// if exists, write it to report file with status "Exists in DB, Not Exists in CSV"
	err = ls.ForEach(func(key string, value []models.ReconRecord) error {
		for _, record := range value {
			record.Status = models.StatusReconRecordExistsDBNotExistsCSV
			err = reportFile.Write(record.ToCSVRow(*reconHistory))
			if err != nil {
				return fmt.Errorf("failed to write payload to file: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to loop localstorage: %w", err)
	}

	return gcsResultFilePayload.GetFilePath(), nil
}

func (s *reconService) processCSVRow(row []string, reconHistory *models.ReconToolHistory, ls storageRecon, reportFile *csv.Writer) error {
	isHeader := strings.Contains(strings.Join(row, ","), "identifier,amount,payment_date")
	if isHeader {
		return nil
	}

	csvRecord, err := models.ConvertStringCSVToReconRecord(row)
	if err != nil {
		identifier := ""
		if len(row) > 0 {
			identifier = row[0]
		}
		rr := models.ReconRecord{Identifier: identifier}
		err = reportFile.Write(rr.ToCSVRowWithErr(*reconHistory, err))
		if err != nil {
			return fmt.Errorf("failed to write payload to file: %w", err)
		}
		return nil
	}

	if csvRecord.PaymentDate != reconHistory.TransactionDate.Format(common.DateFormatDDMMMYYYY) {
		return nil
	}

	acuanRecords, err := ls.Get(csvRecord.Identifier)
	if err != nil {
		err = reportFile.Write(csvRecord.ToCSVRowWithErr(*reconHistory, err))
		if err != nil {
			return fmt.Errorf("failed to write payload to file: %w", err)
		}
		return nil
	}

	if len(acuanRecords) == 0 {
		csvRecord.Status = models.StatusReconRecordNotExistsDBExistsCSV
		err = reportFile.Write(csvRecord.ToCSVRow(*reconHistory))
		if err != nil {
			return fmt.Errorf("failed to write payload to file: %w", err)
		}
		return nil
	}

	var match bool
	for i := 0; i < len(acuanRecords); i++ {
		if acuanRecords[i].Amount.Equal(csvRecord.Amount) {
			match = true
			csvRecord.Match = true
			csvRecord.RefNumber = acuanRecords[i].RefNumber
			csvRecord.Status = models.StatusReconRecordMatch
			acuanRecords = append(acuanRecords[:i], acuanRecords[i+1:]...)
			break
		}
	}
	if !match {
		csvRecord.Status = models.StatusReconRecordNotExistsDBExistsCSV
	}

	if len(acuanRecords) > 0 {
		err = ls.Set(csvRecord.Identifier, acuanRecords)
		if err != nil {
			err = reportFile.Write(csvRecord.ToCSVRowWithErr(*reconHistory, err))
			if err != nil {
				return fmt.Errorf("failed to write payload to file: %w", err)
			}
			return nil
		}
	} else {
		err = ls.Delete(csvRecord.Identifier)
		if err != nil {
			err = reportFile.Write(csvRecord.ToCSVRowWithErr(*reconHistory, err))
			if err != nil {
				return fmt.Errorf("failed to write payload to file: %w", err)
			}
			return nil
		}
	}

	err = reportFile.Write(csvRecord.ToCSVRow(*reconHistory))
	if err != nil {
		return fmt.Errorf("failed to write payload to file: %w", err)
	}

	return nil
}

func (s *reconService) updateReconHistoryStatus(ctx context.Context, rh *models.ReconToolHistory, status string, resultFilePath string) error {
	rh.Status = status
	rh.ResultFilePath = resultFilePath
	_, err := s.srv.sqlRepo.GetReconToolHistoryRepository().Update(ctx, uint64(rh.ID), rh)
	if err != nil {
		return err
	}

	return nil
}
