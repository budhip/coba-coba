package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

	goAcuanLib "bitbucket.org/Amartha/go-acuan-lib/model"
	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/Shopify/sarama"
	"github.com/shopspring/decimal"
)

type ReconService interface {
	DoDailyBalance(ctx context.Context) (url string, err error)
	ProcessReconTaskQueue(ctx context.Context, reconHistoryId uint64) error
	AppendAccountTransactions(accountNumber string, trx goAcuanLib.Transaction)
	UploadReconTemplate(ctx context.Context, req *models.UploadReconFileRequest) error
	GetListReconHistory(ctx context.Context, opts models.ReconToolHistoryFilterOptions) (reconHistories []models.ReconToolHistory, total int, err error)
	GetResultFileURL(ctx context.Context, id uint64) (url string, err error)
}

type reconService struct {
	srv       *Services
	reconDate *time.Time

	accountTransactions map[string][]goAcuanLib.Transaction
}

func NewReconBalanceService(srv *Services) ReconService {
	return &reconService{
		srv:                 srv,
		accountTransactions: make(map[string][]goAcuanLib.Transaction),
		reconDate:           common.YesterdayTime(),
	}
}

// DoDailyBalance will do reconciliation on balance and upload the result on gcp
func (s *reconService) DoDailyBalance(ctx context.Context) (url string, err error) {
	// check if report exist
	isExist, url := s.srv.Storage.IsReportExist(ctx, models.BalanceReconReportName, *s.reconDate)
	if isExist {
		err = errors.New("report is exist")
		return
	}

	// get last daily balance
	consumerOffsets := sarama.OffsetOldest
	lastDailyBalance, err := s.getLastDailyBalance(ctx)
	if err != nil && !errors.Is(common.ErrDataNotFound, err) {
		return
	}
	isLastDailyBalanceAvailable := false
	if lastDailyBalance != nil && len(*lastDailyBalance) != 0 {
		isLastDailyBalanceAvailable = true
	}
	expectedBalance := map[string]decimal.Decimal{}
	if isLastDailyBalanceAvailable {
		consumerOffsets = sarama.OffsetNewest
		for _, b := range *lastDailyBalance {
			expectedBalance[b.AccountNumber] = b.Balance
		}
	}

	// get latest balance
	latestBalances, err := s.srv.sqlRepo.GetAccountRepository().GetAllWithoutPagination(ctx)
	if err != nil {
		return
	}

	now := common.Now()
	// Populate daily balance for all account
	todayBalances := []models.AccountBalanceDaily{}
	actualBalance := map[string]decimal.Decimal{}
	for _, acc := range *latestBalances {
		actualBalance[acc.AccountNumber] = acc.ActualBalance
		todayBalances = append(todayBalances, models.AccountBalanceDaily{
			AccountNumber: acc.AccountNumber,
			Date:          &now,
			Balance:       acc.ActualBalance,
		})
	}

	// Insert daily balance
	if err = s.srv.sqlRepo.GetAccountBalanceDailyRepository().Create(ctx, &todayBalances); err != nil {
		return
	}

	// Stream kafka, it will update s.expectedBalance & s.expectedBalance
	s.srv.consumerRecon.Consume(now, consumerOffsets, s.calculateReconBalanceExpectedResult)

	// Recon: Compare expected balance (kafka) & actual balance (database)
	for accountNumber, v := range s.accountTransactions {
		for _, trx := range v {
			if strings.EqualFold(trx.SourceAccountId, accountNumber) {
				expectedBalance[accountNumber] = expectedBalance[accountNumber].Sub(trx.Amount)
			} else {
				expectedBalance[accountNumber] = expectedBalance[accountNumber].Add(trx.Amount)
			}
		}
	}

	reconDiff := []models.BalanceReconDifference{}
	for accountNumber, expectedBalance := range expectedBalance {
		if !actualBalance[accountNumber].Equal(expectedBalance) {
			reconDiff = append(reconDiff, models.BalanceReconDifference{
				AccountNumber:   accountNumber,
				ConsumerBalance: expectedBalance.String(),
				DatabaseBalance: actualBalance[accountNumber].String(),
			})
		}
	}

	field := []xlog.Field{
		xlog.Int("total-from-database", len(todayBalances)),
		xlog.Int("total-from-kafka", len(expectedBalance)),
		xlog.Int("total-difference", len(reconDiff)),
	}

	// Write storage
	if len(reconDiff) == 0 {
		field = append(field, xlog.String("status", "there is no difference on the recon process"))
		xlog.Info(ctx, "[RECON-BALANCE]", field...)
		return
	}
	field = append(field, xlog.String("status", "there is difference on the recon process"))
	xlog.Warn(ctx, "[RECON-BALANCE]", field...)

	chanData := make(chan []byte)
	go func() {
		defer close(chanData)
		chanData <- []byte(fmt.Sprintf("%s\n", strings.Join(models.BALANCE_RECON_HEADER, models.CSV_SEPARATOR)))
		for _, v := range reconDiff {
			chanData <- []byte(fmt.Sprintf("%s\n", strings.Join(v.ToReconFormat(), models.CSV_SEPARATOR)))
		}
	}()

	gcsPayload := models.CloudStoragePayload{
		Filename: fmt.Sprintf("%d%02d%02d.csv", s.reconDate.Year(), s.reconDate.Month(), s.reconDate.Day()),
		Path:     fmt.Sprintf("%s/%d/%d", models.BalanceReconReportName, s.reconDate.Year(), s.reconDate.Month()),
	}
	r := s.srv.cloudStorage.WriteStream(ctx, &gcsPayload, chanData)
	url, err = r.Wait()

	return
}

// AppendAccountTransactions implements ReconService.
func (s *reconService) AppendAccountTransactions(accountNumber string, trx goAcuanLib.Transaction) {
	s.accountTransactions[accountNumber] = append(s.accountTransactions[accountNumber], trx)
}

// calculateReconBalanceExpectedResult is a helper function to claims consumer message
func (s *reconService) calculateReconBalanceExpectedResult(transactions []goAcuanLib.Transaction) {
	for _, trx := range transactions {
		s.AppendAccountTransactions(trx.SourceAccountId, trx)
		s.AppendAccountTransactions(trx.DestinationAccountId, trx)
	}
}

// getLastDailyBalance will try to get last daily balance
func (s *reconService) getLastDailyBalance(ctx context.Context) (lastDailyBalance *[]models.AccountBalanceDaily, err error) {
	// Get last record
	lastAbd, err := s.srv.sqlRepo.GetAccountBalanceDailyRepository().GetLast(ctx)
	if err != nil {
		return
	}

	// Get last daily balance
	return s.srv.sqlRepo.GetAccountBalanceDailyRepository().ListByDate(ctx, *lastAbd.Date)
}

func (s *reconService) UploadReconTemplate(ctx context.Context, req *models.UploadReconFileRequest) error {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// Check header
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	streamReadResult := s.srv.fileRepo.StreamReadMultipartFile(ctx, req.ReconFile)
	firstLine := <-streamReadResult
	if firstLine.Err != nil {
		err = firstLine.Err
		return err
	}
	delimiter := ","
	semiColonIndex := strings.Index(firstLine.Data, ";")
	if semiColonIndex != -1 {
		delimiter = ";"
	}
	if !strings.HasPrefix(firstLine.Data, fmt.Sprintf("identifier%samount%spayment_date%sremark", delimiter, delimiter, delimiter)) {
		err = models.GetErrMap(models.ErrKeyReconFileInvalidTemplate, "invalid template")
		return err
	}

	// Upload gcs
	now := common.Now()
	gcsPayload := &models.CloudStoragePayload{
		Filename: fmt.Sprintf("%s.csv", now.Format(common.DateFormatYYYYMMDDHHMMSSWithoutDash)),
		Path:     fmt.Sprintf("%s/upload/%04d/%02d", models.ReconToolFolderName, now.Year(), now.Month()),
	}
	writer := s.srv.cloudStorage.NewWriter(ctx, gcsPayload)
	defer writer.Close()

	_, err = writer.Write([]byte(fmt.Sprint(firstLine.Data, "\n")))
	if err != nil {
		xlog.Errorf(ctx, "failed to write row to writer: %v", err)
		return err
	}

	for v := range streamReadResult {
		if v.Err != nil {
			cancel()
			err = v.Err
			xlog.Errorf(ctx, "got error while streaming data: %v", err)
			return err
		}

		_, err = writer.Write([]byte(fmt.Sprint(v.Data, "\n")))
		if err != nil {
			cancel()
			xlog.Errorf(ctx, "failed to write row to writer: %v", err)
			return err
		}
	}

	// Insert to db
	created, err := s.srv.sqlRepo.GetReconToolHistoryRepository().Create(ctx, &models.CreateReconToolHistoryIn{
		OrderType:        req.OrderType,
		TransactionType:  req.TransactionType,
		TransactionDate:  req.TransactionDate,
		UploadedFilePath: gcsPayload.GetFilePath(),
		Status:           models.ReconHistoryStatusPending,
	})
	if err != nil {
		xlog.Errorf(ctx, "insert db failed: %v", err)

		// Rollback uploaded file
		if subErr := s.deleteReconFile(ctx, gcsPayload); subErr != nil {
			xlog.Errorf(ctx, "failed delete recon file: %v", subErr)
		}

		err = common.ErrUnableToCreate
		return err
	}

	// Notify recon engine
	if err = s.srv.reconPub.Publish(ctx, models.ReconPublisher{
		ID:   fmt.Sprint(created.ID),
		Task: models.ReconTaskName,
	}); err != nil {
		xlog.Errorf(ctx, "failed to publish: %v", err)

		// Rollback uploaded file
		if subErr := s.deleteReconFile(ctx, gcsPayload); subErr != nil {
			xlog.Errorf(ctx, "failed delete recon file: %v", subErr)
		}

		// Rollback inserted db
		if subErr := s.srv.sqlRepo.GetReconToolHistoryRepository().DeleteByID(ctx, fmt.Sprint(created.ID)); subErr != nil {
			xlog.Errorf(ctx, "failed delete recon data: %v", subErr)
		}

		err = common.ErrUnableToRecon
		return err
	}

	return nil
}

func (s *reconService) deleteReconFile(ctx context.Context, gcsPayload *models.CloudStoragePayload) error {
	return s.srv.cloudStorage.DeleteFile(ctx, gcsPayload)
}

func (s *reconService) GetListReconHistory(ctx context.Context, opts models.ReconToolHistoryFilterOptions) (reconHistories []models.ReconToolHistory, total int, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	repo := s.srv.sqlRepo.GetReconToolHistoryRepository()

	reconHistories, err = repo.GetList(ctx, opts)
	if err != nil {
		return reconHistories, total, err
	}

	total, err = repo.CountAll(ctx, opts)
	if err != nil {
		return
	}

	return reconHistories, total, nil
}

func (s *reconService) GetResultFileURL(ctx context.Context, id uint64) (url string, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	repo := s.srv.sqlRepo.GetReconToolHistoryRepository()

	reconHistory, err := repo.GetById(ctx, id)
	if err != nil {
		return "", err
	}

	if reconHistory.ResultFilePath == "" {
		return "", common.ErrFilePathEmpty
	}

	expireDuration := 15 * time.Minute // default 15 minutes
	if s.srv.conf.ReconEngine.ResultURLExpiryTime != 0 {
		expireDuration = time.Duration(s.srv.conf.ReconEngine.ResultURLExpiryTime) * time.Minute
	}

	return s.srv.cloudStorage.GetSignedURL(reconHistory.ResultFilePath, expireDuration)
}
