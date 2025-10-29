package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/hashicorp/go-multierror"
	"golang.org/x/exp/slices"
	"golang.org/x/sync/errgroup"
)

type TransactionService interface {
	PublishTransaction(ctx context.Context, in models.DoPublishTransactionRequest) (out models.DoPublishTransactionResponse, err error)
	NewStoreBulkTransaction(ctx context.Context, req []models.TransactionReq) (err error)
	GetAllTransaction(ctx context.Context, opts models.TransactionFilterOptions) (transactions []models.GetTransactionOut, total int, err error)
	GetByTransactionTypeAndRefNumber(ctx context.Context, req *models.TransactionGetByTypeAndRefNumberRequest) (*models.GetTransactionOut, error)
	GenerateTransactionReport(ctx context.Context) (urls []string, err error)
	DownloadTransactionFileCSV(ctx context.Context, req models.DownloadTransactionRequest) (err error)

	GetStatusCount(ctx context.Context, threshold uint, opts models.TransactionFilterOptions) (out models.StatusCountTransaction, err error)

	StoreBulkTransaction(ctx context.Context, req []models.TransactionReq) (err error)
	StoreTransaction(ctx context.Context, req models.TransactionReq, processType models.TransactionStoreProcessType, clientID string) (out models.GetTransactionOut, err error)

	CommitReservedTransaction(ctx context.Context, transactionID, clientID string) (*models.Transaction, error)
	CancelReservedTransaction(ctx context.Context, transactionID string) (*models.Transaction, error)
	CollectRepayment(ctx context.Context) (out *models.CollectRepayment, err error)
	GetReportRepayment(ctx context.Context) (out []models.ReportRepayment, err error)
}

type transaction service

var _ TransactionService = (*transaction)(nil)

func (ts *transaction) DownloadTransactionFileCSV(ctx context.Context, req models.DownloadTransactionRequest) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	trxRepo := ts.srv.sqlRepo.GetTransactionRepository()

	// Init master data
	orderTypes, err := ts.srv.masterDataRepo.GetListOrderType(ctx, models.FilterMasterData{})
	if err != nil {
		err = fmt.Errorf("unable to GetListOrderType: %w", err)
		return
	}
	mapOrderTypes, mapTransactionTypes := models.MakeOrderTypesMap(orderTypes)

	sc, err := trxRepo.GetStatusCount(ctx, models.DefaultThresholdStatusCountTransaction, req.Options)
	if err != nil {
		return
	}

	if sc.ExceedThreshold {
		return common.ErrRowLimitDownloadExceed
	}

	header := []string{
		"Transaction ID",
		"No Ref",
		"Order Type Code",
		"Order Type Name",
		"Transaction Type Code",
		"Transaction Type Name",
		"Transaction Date",
		"From Account Number",
		"From Account Name",
		"From Account Product Name",
		"To Account Number",
		"To Account Name",
		"To Account Product Name",
		"Amount",
		"Status",
		"Description",
		"Method",
		"Currency",
		"Metadata",
	}

	w := csv.NewWriter(req.Writer)

	err = w.Write(header)
	if err != nil {
		err = fmt.Errorf("failed to write header: %w", err)
		return err
	}

	for trx := range trxRepo.StreamAll(ctx, req.Options) {
		if trx.Err != nil {
			err = fmt.Errorf("failed to read stream: %w", trx.Err)
			return err
		}

		t := trx.Data.ToGetTransactionOut(mapOrderTypes, mapTransactionTypes)

		amount := "0"
		if trx.Data.Amount.Valid {
			amount = trx.Data.Amount.Decimal.String()
		}

		err = w.Write([]string{
			t.TransactionID,
			t.RefNumber,
			t.OrderType,
			t.OrderTypeName,
			t.TransactionType,
			t.TransactionTypeName,
			t.TransactionTime.In(common.GetLocation()).Format(common.DateFormatYYYYMMDDWithTime),
			t.FromAccount,
			t.FromAccountName,
			t.FromAccountProductTypeName,
			t.ToAccount,
			t.ToAccountName,
			t.ToAccountProductTypeName,
			amount,
			t.Status,
			t.Description,
			t.Method,
			t.Currency,
			t.Metadata,
		})
		if err != nil {
			err = fmt.Errorf("failed to write row: %w", err)
			return err
		}
	}

	w.Flush()

	err = w.Error()
	if err != nil {
		err = fmt.Errorf("failed to flush writer: %w", err)
		return err
	}

	return nil
}

func (ts *transaction) GetAllTransaction(ctx context.Context, opts models.TransactionFilterOptions) (result []models.GetTransactionOut, total int, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	trxRepo := ts.srv.sqlRepo.GetTransactionRepository()

	// Init master data
	orderTypes, err := ts.srv.masterDataRepo.GetListOrderType(ctx, models.FilterMasterData{})
	if err != nil {
		err = fmt.Errorf("unable to GetListOrderType: %w", err)
		return
	}
	mapOrderTypes, mapTransactionTypes := models.MakeOrderTypesMap(orderTypes)

	// List transactions
	opts.OnlyAMF = ts.srv.flag.IsEnabled(ts.srv.conf.FeatureFlagKeyLookup.ShowOnlyAMFTransactionList)
	transactions, err := trxRepo.GetList(ctx, opts)
	if err != nil {
		return
	}

	// Modify entity
	for _, v := range transactions {
		result = append(result, v.ToGetTransactionOut(mapOrderTypes, mapTransactionTypes))
	}

	// Count
	total, err = trxRepo.CountAll(ctx, opts)
	if err != nil {
		return
	}

	return result, total, nil
}

func (ts *transaction) ensureAccountExists(ctx context.Context, accountNumbers ...string) error {
	accRepo := ts.srv.sqlRepo.GetAccountRepository()
	es, err := accRepo.CheckAccountNumbers(ctx, accountNumbers)
	if err != nil {
		return err
	}

	var errs *multierror.Error

	autoCreateAccount := ts.srv.flag.IsEnabled(ts.srv.conf.FeatureFlagKeyLookup.AutoCreateAccountIfNotExists)

	for accountNumber, exists := range es {
		if !exists {
			isNeedToReject := ts.srv.conf.FeatureFlag.EnableConsumerValidationReject && !autoCreateAccount
			if isNeedToReject {
				errs = multierror.Append(errs, fmt.Errorf("account number %s not exists", accountNumber))
				continue
			}

			if autoCreateAccount {
				err = accRepo.Create(ctx, models.CreateAccount{AccountNumber: accountNumber})
				if err != nil {
					errs = multierror.Append(errs, err)
				}
			} else {
				errs = multierror.Append(errs, fmt.Errorf("account number %s not exists", accountNumber))
			}
		}
	}

	return errs.ErrorOrNil()
}

func (ts *transaction) GenerateTransactionReport(ctx context.Context) (urls []string, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	group, ctx := errgroup.WithContext(ctx)

	reportDate := *common.YesterdayTime()
	chanTrx := ts.srv.sqlRepo.GetTransactionRepository().StreamAll(ctx, models.TransactionFilterOptions{
		TransactionDate: &reportDate,
	})

	fileCounter := 1

	url, reportFile, err := ts.createReportFile(ctx, reportDate, fileCounter)
	if err != nil {
		xlog.Errorf(ctx, "unable to create report file: %v", err)
		return
	}
	urls = append(urls, url)

	group.Go(func() error {
		var line uint64
		for v := range chanTrx {
			if v.Err != nil {
				cancel()
				xlog.Errorf(ctx, "got error while streaming data: %v", err)
				return v.Err
			}

			line++
			if line%models.MaxRowTransactionFile == 0 {
				errClose := reportFile.Close()
				if errClose != nil {
					xlog.Errorf(ctx, "got error while close file: %v", err)
					return errClose
				}
				fileCounter++

				url, reportFile, err = ts.createReportFile(ctx, reportDate, fileCounter)
				if err != nil {
					cancel()
					xlog.Errorf(ctx, "failed to create next report file: %v", err)
					return err
				}
				urls = append(urls, url)
			}

			row := []byte(fmt.Sprintf("%s\n", strings.Join(v.Data.ToReconFormat(), models.CSV_SEPARATOR)))
			_, err := reportFile.Write(row)
			if err != nil {
				cancel()
				xlog.Errorf(ctx, "failed to write row to writer: %v", err)
				return err
			}
		}

		// close latest file
		errClose := reportFile.Close()
		if errClose != nil {
			xlog.Errorf(ctx, "got error while close file: %v", err)
			return errClose
		}
		return nil
	})

	err = group.Wait()
	if err != nil {
		errDel := ts.deleteReportFiles(ctx, reportDate, fileCounter)
		if errDel != nil {
			xlog.Errorf(ctx, "unable to delete file: %v", err)
		}
	}

	return
}

func (ts *transaction) createReportFile(ctx context.Context, reportDate time.Time, fileCounter int) (string, io.WriteCloser, error) {
	payload := &models.CloudStoragePayload{
		Filename: fmt.Sprintf("%d%02d%02d__%d.csv", reportDate.Year(), reportDate.Month(), reportDate.Day(), fileCounter),
		Path:     fmt.Sprintf("%s/%d/%d", models.TransactionReportName, reportDate.Year(), reportDate.Month()),
	}

	r := ts.srv.cloudStorage.NewWriter(ctx, payload)

	_, err := r.Write([]byte(fmt.Sprintf("%s\n", strings.Join(models.TRANSACTION_REPORT_HEADER, models.CSV_SEPARATOR))))
	if err != nil {
		return "", nil, err
	}

	url := ts.srv.cloudStorage.GetURL(payload)
	return url, r, nil
}

func (ts *transaction) deleteReportFiles(ctx context.Context, reportDate time.Time, fileCounter int) error {
	group, ctx := errgroup.WithContext(ctx)

	for i := 1; i <= fileCounter; i++ {
		payload := &models.CloudStoragePayload{
			Filename: fmt.Sprintf("%d%02d%02d__%d.csv", reportDate.Year(), reportDate.Month(), reportDate.Day(), fileCounter),
			Path:     fmt.Sprintf("%s/%d/%d", models.TransactionReportName, reportDate.Year(), reportDate.Month()),
		}

		group.Go(func() error {
			return ts.srv.cloudStorage.DeleteFile(ctx, payload)
		})
	}

	return group.Wait()
}

func (ts *transaction) StoreTransaction(ctx context.Context, req models.TransactionReq, processType models.TransactionStoreProcessType, clientID string) (out models.GetTransactionOut, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	en, err := req.ToRequest()
	if err != nil {
		return
	}

	err = ts.validateInputStoreTransaction(ctx, processType, req)
	if err != nil {
		return
	}

	exists, err := ts.srv.sqlRepo.GetTransactionRepository().CheckRefNumbers(ctx, req.RefNumber)
	if err != nil {
		return
	}

	if exists[en.RefNumber] {
		return out, common.ErrDataTrxDuplicate
	}

	err = ts.ensureAccountExists(ctx, req.FromAccount, req.ToAccount)
	if err != nil {
		return
	}

	calculateBalance := getBalanceCalculator(processType)

	err = ts.srv.sqlRepo.Atomic(ctx, func(atomicCtx context.Context, r repositories.SQLRepository) error {
		accRepo := r.GetAccountRepository()
		trxRepo := r.GetTransactionRepository()

		balances, errAtomic := accRepo.GetAccountBalances(atomicCtx, models.GetAccountBalanceRequest{
			AccountNumbers: []string{req.FromAccount, req.ToAccount},
			ForUpdate:      true,
		})
		if errAtomic != nil {
			return errAtomic
		}

		for accountNumber, balance := range balances {
			// skip balance sufficiency check for specific account number
			if slices.Contains(ts.srv.conf.TransactionValidationConfig.SkipBalanceCheckAccountNumber, accountNumber) {
				balances[accountNumber] = models.NewBalance(
					balance.Actual(),
					balance.Pending(),
					models.WithIgnoreBalanceSufficiency(),
					models.WithBalanceLimitEnabled(ts.srv.flag.IsEnabled(ts.srv.conf.FeatureFlagKeyLookup.BalanceLimitToggle)),
				)
			}
		}

		trxSet := models.NewTransactionSet(req.FromAccount, req.ToAccount, req.Amount.Decimal)

		balances, errAtomic = calculateBalance(trxSet, balances)
		if errAtomic != nil {
			return errAtomic
		}

		for an, balance := range balances {
			_, errAtomic = accRepo.UpdateAccountBalance(atomicCtx, an, balance)
			if errAtomic != nil {
				return errAtomic
			}
		}

		errAtomic = trxRepo.Store(atomicCtx, &en)
		if errAtomic != nil {
			return errAtomic
		}

		// publish non reserved transaction
		if processType == models.TransactionStoreProcessNormal {
			errAtomic = ts.publishNotificationSuccess(atomicCtx, en, clientID)
			if errAtomic != nil {
				return errAtomic
			}
		}

		return nil
	})

	out = en.ToGetTransactionOut(map[string]string{}, map[string]string{})

	return
}

func (ts *transaction) publishNotificationSuccess(ctx context.Context, trx models.Transaction, clientID string) error {
	payloadNotification, err := trx.ToAcuanNotificationMessage(
		models.StatusTransactionNotificationSuccess,
		"success create transaction",
		clientID)
	if err != nil {
		return err
	}

	return ts.srv.transactionNotification.Publish(ctx, *payloadNotification)
}

func (ts *transaction) StoreBulkTransaction(ctx context.Context, req []models.TransactionReq) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	ops := "TransactionService.StoreBulkTransaction"

	trxRepo := ts.srv.sqlRepo.GetTransactionRepository()

	chunkRequest := common.ChunkBy(req, ts.srv.conf.TransactionConfig.BatchSize)
	for _, chunk := range chunkRequest {
		var refNumbers []string
		var rawReqs []*models.Transaction

		for _, request := range chunk {
			en, err := request.ToRequest()
			if err != nil {
				return err
			}

			refNumbers = append(refNumbers, en.RefNumber)
			rawReqs = append(rawReqs, &en)
		}

		exists, err := trxRepo.CheckRefNumbers(ctx, refNumbers...)
		if err != nil {
			return err
		}

		var transactionDBReq []*models.Transaction
		var accountNumbers []string

		for _, request := range rawReqs {
			if exists[request.RefNumber] {
				message := fmt.Sprintf("%s - duplicate refNumber: %s. skipping", ops, request.RefNumber)
				xlog.Info(ctx, message)
				continue
			}

			transactionDBReq = append(transactionDBReq, request)
			accountNumbers = append(accountNumbers, request.FromAccount, request.ToAccount)
		}

		if len(transactionDBReq) == 0 {
			return nil
		}

		err = trxRepo.StoreBulkTransaction(ctx, transactionDBReq)
		if err != nil {
			return err
		}

		err = ts.ensureAccountExists(ctx, accountNumbers...)
		if err != nil {
			xlog.Errorf(ctx, "%s.%s - %v", ops, "ensureAccountExists", err)
			return err
		}
	}

	return err
}

func (ts *transaction) GetByTransactionTypeAndRefNumber(ctx context.Context, req *models.TransactionGetByTypeAndRefNumberRequest) (*models.GetTransactionOut, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	transaction, err := ts.srv.sqlRepo.GetTransactionRepository().
		GetByTransactionTypeAndRefNumber(ctx, req)
	if err != nil {
		err = checkDatabaseError(err)
		return nil, err
	}

	return transaction, nil
}

// CommitReservedTransaction is the next function executed after reserve a transaction.
// Commit will change transaction to SUCCESS and update balance accordingly.
func (ts *transaction) CommitReservedTransaction(ctx context.Context, transactionID, clientID string) (trx *models.Transaction, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	trx, err = ts.srv.sqlRepo.GetTransactionRepository().GetByTransactionID(ctx, transactionID)
	if err != nil {
		err = checkDatabaseError(err)
		return nil, fmt.Errorf("unable to get trx: %w", err)
	}

	// Check already commit
	if trx.IsSuccess() {
		return trx, nil
	}

	// Check if not reserved
	if !trx.IsPending() {
		return nil, common.ErrTransactionNotReserved
	}

	// Process
	err = ts.srv.sqlRepo.Atomic(ctx, func(actx context.Context, r repositories.SQLRepository) error {
		// Update transaction
		newTrx, errAtomic := r.GetTransactionRepository().UpdateStatus(actx, trx.ID, models.TransactionStatusSuccessNum)
		if errAtomic != nil {
			return fmt.Errorf("unable to update trx: %w", errAtomic)
		}
		trx = newTrx
		if !trx.Amount.Valid {
			return fmt.Errorf("invalid trx %s amount: %v", trx.TransactionID, trx.Amount)
		}

		// Check account
		accBalances, errAtomic := r.GetAccountRepository().GetAccountBalances(actx, models.GetAccountBalanceRequest{
			AccountNumbers: []string{trx.FromAccount, trx.ToAccount},
			ForUpdate:      true,
		})
		if errAtomic != nil {
			return fmt.Errorf("unable to get balances: %w", errAtomic)
		}

		// Calculate
		trxSet := models.TransactionSet{
			FromAccount: trx.FromAccount,
			ToAccount:   trx.ToAccount,
			Amount:      trx.Amount.Decimal,
		}
		accBalances, errAtomic = trxSet.CalculateCommit(accBalances)
		if errAtomic != nil {
			return fmt.Errorf("unable to calculate balance: %w", errAtomic)
		}

		// Update
		for account, balance := range accBalances {
			_, errAtomic = r.GetAccountRepository().UpdateAccountBalance(actx, account, balance)
			if errAtomic != nil {
				return fmt.Errorf("unable to update balance: %w", errAtomic)
			}
		}

		errAtomic = ts.publishNotificationSuccess(actx, *trx, clientID)
		if errAtomic != nil {
			return errAtomic
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return trx, nil
}

// CancelReservedTransaction is the next function executed after reserve a transaction.
// Cancel will change transaction to CANCEL and rollback fromAccount balance.
func (ts *transaction) CancelReservedTransaction(ctx context.Context, transactionID string) (trx *models.Transaction, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	trx, err = ts.srv.sqlRepo.GetTransactionRepository().GetByTransactionID(ctx, transactionID)
	if err != nil {
		err = checkDatabaseError(err)
		return nil, fmt.Errorf("unable to get trx: %w", err)
	}

	// Check already cancel
	if trx.IsCancel() {
		return trx, nil
	}

	// Check if not reserved
	if !trx.IsPending() {
		return nil, common.ErrTransactionNotReserved
	}

	// Process
	err = ts.srv.sqlRepo.Atomic(ctx, func(actx context.Context, r repositories.SQLRepository) error {
		// Update transaction
		newTrx, errAtomic := r.GetTransactionRepository().UpdateStatus(actx, trx.ID, models.TransactionStatusCancelNum)
		if errAtomic != nil {
			return fmt.Errorf("unable to update trx: %w", errAtomic)
		}
		trx = newTrx
		if !trx.Amount.Valid {
			return fmt.Errorf("invalid trx %s amount: %v", trx.TransactionID, trx.Amount)
		}

		// Check account
		accBalances, errAtomic := r.GetAccountRepository().GetAccountBalances(actx, models.GetAccountBalanceRequest{
			AccountNumbers: []string{trx.FromAccount},
			ForUpdate:      true,
		})
		if errAtomic != nil {
			return fmt.Errorf("unable to get balances: %w", errAtomic)
		}

		// Calculate
		trxSet := models.TransactionSet{
			FromAccount: trx.FromAccount,
			ToAccount:   trx.ToAccount,
			Amount:      trx.Amount.Decimal,
		}
		accBalances, errAtomic = trxSet.CalculateCancel(accBalances)
		if errAtomic != nil {
			return fmt.Errorf("unable to calculate balance: %w", errAtomic)
		}

		_, errAtomic = r.GetAccountRepository().UpdateAccountBalance(actx, trx.FromAccount, accBalances[trx.FromAccount])
		if errAtomic != nil {
			return fmt.Errorf("unable to get update balance: %w", errAtomic)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return trx, nil
}

func (ts *transaction) GetStatusCount(ctx context.Context, threshold uint, opts models.TransactionFilterOptions) (out models.StatusCountTransaction, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	trxRepo := ts.srv.sqlRepo.GetTransactionRepository()

	out, err = trxRepo.GetStatusCount(ctx, threshold, opts)
	if err != nil {
		return
	}

	return
}

func (ts *transaction) GetReportRepayment(ctx context.Context) (out []models.ReportRepayment, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// calculate date range: yesterday and 6 days before
	endDate := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour)
	startDate := endDate.AddDate(0, 0, -6)

	trxRepo := ts.srv.sqlRepo.GetTransactionRepository()
	ttl := 24 * time.Hour

	// Loop day-by-day, get (or cache) each day's result, then merge
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dayStart := d
		dayEnd := d.AddDate(0, 0, 1) // half-open: [dayStart, dayEnd)

		var dayRes []models.ReportRepayment
		opts := models.GetOrSetCacheOptions[[]models.ReportRepayment]{
			Key: getCacheKeyReportRepayment(dayStart, dayEnd),
			TTL: ttl,
			Fn: func() ([]models.ReportRepayment, error) {
				return trxRepo.GetReportRepayment(ctx, dayStart, dayEnd)
			},
		}

		dayRes, err = repositories.GetOrSetCache[[]models.ReportRepayment](ctx, ts.srv.cacheRepo, opts)
		if err != nil {
			return nil, err
		}

		out = append(out, dayRes...)
	}

	return out, nil
}

func (ts *transaction) CollectRepayment(ctx context.Context) (out *models.CollectRepayment, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	yesterday := common.Now().AddDate(0, 0, -1)

	xlog.Infof(ctx, "start collect repayment with date : %s", yesterday)

	trxRepo := ts.srv.sqlRepo.GetTransactionRepository()
	out, err = trxRepo.ColectRepayment(ctx, yesterday)
	if err != nil {
		return out, err
	}

	xlog.Infof(ctx, "finish collect repayment with date : %s", yesterday)

	return out, nil
}
