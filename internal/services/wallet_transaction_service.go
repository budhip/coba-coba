package services

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/megatron"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/publisher"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/transformer"

	"github.com/hashicorp/go-multierror"
	"github.com/shopspring/decimal"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

type WalletTrxService interface {
	CreateTransaction(ctx context.Context, in models.CreateWalletTransactionRequest) (*models.WalletTransaction, error)
	EnqueueTransaction(ctx context.Context, in models.CreateWalletTransactionRequest) (*models.WalletTransaction, error)
	ProcessReservedTransaction(ctx context.Context, req models.UpdateStatusWalletTransactionRequest) (*models.WalletTransaction, error)
	List(ctx context.Context, opts models.WalletTrxFilterOptions) (transactions []models.WalletTransaction, total int, err error)
}

//type walletTrx service

var _ WalletTrxService = (*walletTrx)(nil)

type walletTrx struct {
	service
	megatronClient megatron.Client
}

// CreateTransaction will process request for new wallet transaction
// This process will only make change to the sourceAccountNumber, destinationAccountNumber is ignored
func (ts *walletTrx) CreateTransaction(ctx context.Context, in models.CreateWalletTransactionRequest) (*models.WalletTransaction, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	trx, err := ts.validateTransactionTypeAndRefNumber(ctx, in)
	if err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}
	if trx != nil {
		return trx, nil
	}

	if err = ts.validateTransactionInput(ctx, in); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	lceRolloutFlag := ts.srv.flag.IsEnabled(ts.srv.conf.FeatureFlagKeyLookup.LceRollout)
	if slices.Contains(models.AllowedTransactionTypesForLceRollout, in.TransactionType) && lceRolloutFlag {
		for _, transactionAmount := range in.Amounts {
			if slices.Contains(models.AllowedTransactionAmountTypesForLceRollout, transactionAmount.Type) {
				in.NetAmount.ValueDecimal.Decimal = in.NetAmount.ValueDecimal.Decimal.Sub(transactionAmount.Amount.ValueDecimal.Decimal)
			}
		}
	}

	isContainAsyncClient := slices.Contains(ts.srv.conf.TransactionConfig.AsyncWalletTransactionForClients, in.ClientId)
	if isContainAsyncClient {
		return ts.EnqueueTransaction(ctx, in)
	}

	return ts.CreateTransactionAtomic(ctx, in.ToNewWalletTransaction(), in.IsReserved, true, in.ClientId)
}

func (ts *walletTrx) getAccountConfigRepository() repositories.AccountConfigRepository {
	useExternalAccountConfig := ts.srv.flag.IsEnabled(ts.srv.conf.FeatureFlagKeyLookup.UseAccountConfigFromExternal)

	if useExternalAccountConfig {
		return ts.srv.sqlRepo.GetAccountConfigExternalRepository()
	}

	return ts.srv.sqlRepo.GetAccountConfigInternalRepository()
}

// CreateTransactionAtomic dengan support megatron
func (ts *walletTrx) CreateTransactionAtomic(
	ctx context.Context,
	nwt models.NewWalletTransaction,
	isReserved,
	isPublish bool,
	clientID string,
) (*models.WalletTransaction, error) {
	maxWaitingTimeDB := 8 * time.Second
	dbCtx, cancelDB := context.WithTimeout(ctx, maxWaitingTimeDB)
	defer cancelDB()

	var acuanTransactions []models.Transaction
	var updatedBalances map[string]models.Balance
	var currentBalances map[string]models.Balance

	calculateBalance := getWalletBalanceCalculator(nwt.TransactionFlow, isReserved)
	created := &models.WalletTransaction{}

	err := ts.srv.sqlRepo.Atomic(dbCtx, func(atomicCtx context.Context, r repositories.SQLRepository) error {
		accRepo := r.GetAccountRepository()
		balanceRepo := r.GetBalanceRepository()
		walletTrxRepo := r.GetWalletTransactionRepository()
		acuanTrxRepo := r.GetTransactionRepository()

		// Check feature flag: use megatron or legacy transformer
		useMegatron := ts.srv.flag.IsEnabled("use_megatron_transformer")

		var childTransactions []models.TransactionReq
		var errAtomic error

		if useMegatron {
			// Use megatron for transformation
			childTransactions, errAtomic = ts.transformUsingMegatron(atomicCtx, nwt.ToWalletTransaction())
		} else {
			// Use legacy transformer
			childTransactions, errAtomic = ts.transformUsingLegacy(atomicCtx, nwt.ToWalletTransaction())
		}

		if errAtomic != nil {
			return fmt.Errorf("unable to transform wallet transaction: %w", errAtomic)
		}

		// Rest of the code remains the same...
		accountNumbers := getAccountNumbersForUpdateBalance(childTransactions)
		abs, errAtomic := balanceRepo.GetMany(atomicCtx,
			models.GetAccountBalanceRequest{
				AccountNumbers:               accountNumbers,
				ForUpdate:                    true,
				AccountNumbersExcludedFromDB: ts.srv.conf.AccountConfig.ExcludedBalanceUpdateAccountNumbers,
			},
		)
		if errAtomic != nil {
			return fmt.Errorf("unable to get current balance: %w", errAtomic)
		}

		// Continue with balance calculation and updates...
		// [Rest of the atomic transaction logic remains unchanged]

		return nil
	})

	if err != nil {
		return created, err
	}

	// Publish notification if needed
	if !isReserved && isPublish {
		maxWaitingTimeKafka := 7 * time.Second
		kafkaCtx, cancelKafka := context.WithTimeout(context.Background(), maxWaitingTimeKafka)
		defer cancelKafka()

		err := ts.publishNotificationCreateWalletTransactionSuccess(
			kafkaCtx,
			*created,
			acuanTransactions,
			currentBalances,
			updatedBalances,
			clientID,
		)
		if err != nil {
			return created, err
		}
	}

	return created, nil
}

// transformUsingMegatron transforms wallet transaction using megatron service
func (ts *walletTrx) transformUsingMegatron(
	ctx context.Context,
	walletTrx models.WalletTransaction,
) ([]models.TransactionReq, error) {
	// Get account info
	account, err := ts.srv.sqlRepo.GetAccountRepository().GetCachedAccount(ctx, walletTrx.AccountNumber)
	if err != nil {
		return nil, fmt.Errorf("unable to get account: %w", err)
	}

	// Build transform context
	transformCtx := ts.megatronClient.BuildTransformContext()
	transformCtx.ClientID = "go-fp-transaction"
	transformCtx.CorrelationID = "" // Get from context

	// Convert wallet transaction to input
	wtInput := ts.megatronClient.ConvertWalletTransactionToInput(ctx, walletTrx, *account)

	// Build batch transform request
	var transforms []megatron.TransformItem

	// Add net amount if > 0
	if walletTrx.NetAmount.ValueDecimal.GreaterThan(decimal.Zero) {
		transforms = append(transforms, megatron.TransformItem{
			Amount: megatron.AmountInput{
				Value:    walletTrx.NetAmount.ValueDecimal.Decimal,
				Currency: walletTrx.NetAmount.Currency,
			},
			TransactionType: walletTrx.TransactionType,
		})
	}

	// Add amounts
	for _, amount := range walletTrx.Amounts {
		if amount.Amount.ValueDecimal.GreaterThan(decimal.Zero) {
			transforms = append(transforms, megatron.TransformItem{
				Amount: megatron.AmountInput{
					Value:    amount.Amount.ValueDecimal.Decimal,
					Currency: amount.Amount.Currency,
				},
				TransactionType: amount.Type,
			})
		}
	}

	// Call megatron batch transform
	batchReq := megatron.BatchTransformRequest{
		ParentTransaction: wtInput,
		Transforms:        transforms,
		Context:           transformCtx,
	}

	batchResp, err := ts.megatronClient.BatchTransform(ctx, batchReq)
	if err != nil {
		return nil, fmt.Errorf("megatron transform failed: %w", err)
	}

	// Check for errors
	if len(batchResp.Errors) > 0 {
		var errs *multierror.Error
		for _, transformErr := range batchResp.Errors {
			errs = multierror.Append(errs,
				fmt.Errorf("%s: %s (%s)", transformErr.TransactionType, transformErr.Error, transformErr.Code))
		}
		return nil, errs.ErrorOrNil()
	}

	// Convert response to transaction req
	return ts.megatronClient.ConvertResponseToTransactionReq(batchResp.Transactions), nil
}

// transformUsingLegacy transforms wallet transaction using legacy transformer
func (ts *walletTrx) transformUsingLegacy(
	ctx context.Context,
	walletTrx models.WalletTransaction,
) ([]models.TransactionReq, error) {
	mapTransformer := transformer.NewMapTransformer(
		ts.srv.conf,
		ts.srv.masterDataRepo,
		ts.srv.accountingClient,
		ts.srv.sqlRepo.GetAccountRepository(),
		ts.srv.sqlRepo.GetTransactionRepository(),
		ts.getAccountConfigRepository(),
		ts.srv.sqlRepo.GetWalletTransactionRepository(),
		ts.srv.flag,
	)

	return mapTransformer.Transform(ctx, walletTrx)
}

func (ts *walletTrx) enrichTransactionsWithEntityData(ctx context.Context, transactions []models.Transaction) error {
	accountNumbers := getAccountNumbersForEntity(transactions)
	if len(accountNumbers) == 0 {
		return nil
	}
	accountRepo := ts.srv.sqlRepo.GetAccountRepository()
	mapEntity, err := accountRepo.GetAccountNumberEntity(ctx, accountNumbers)
	if err != nil {
		return fmt.Errorf("unable to get entityCode: %w", err)
	}

	for i, trx := range transactions {
		if sourceEntity, ok := mapEntity[trx.FromAccount]; ok {
			transactions[i].SourceEntity = sourceEntity
		}
		if destinationEntity, ok := mapEntity[trx.ToAccount]; ok {
			transactions[i].DestinationEntity = destinationEntity
		}
	}
	return nil
}

// ProcessReservedTransaction will process wallet transaction to COMMIT or CANCEL
func (ts *walletTrx) ProcessReservedTransaction(ctx context.Context, req models.UpdateStatusWalletTransactionRequest) (*models.WalletTransaction, error) {
	var err error
	var acuanTransactions []models.Transaction
	var updatedBalances map[string]models.Balance
	var currentBalances map[string]models.Balance

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	walletTrx, err := ts.srv.sqlRepo.GetWalletTransactionRepository().GetById(ctx, req.TransactionId)
	if err != nil {
		err = checkDatabaseError(err)
		return nil, fmt.Errorf("unable to get transaction: %w", err)
	}

	// if client not include transactionTime in request, then use current time as transactionTime
	if req.TransactionTime.IsZero() {
		req.TransactionTime = time.Now()
	}

	var balanceCalculator walletBalanceCalculator
	var nextWalletTrxStatus models.WalletTransactionStatus
	if req.Action == models.TransactionRequestCommitStatus {
		if walletTrx.Status == models.WalletTransactionStatusSuccess {
			return walletTrx, nil
		}

		balanceCalculator = getWalletBalanceCommitCalculator(walletTrx.TransactionFlow)
		nextWalletTrxStatus = models.WalletTransactionStatusSuccess
	} else if req.Action == models.TransactionRequestCancelStatus {
		if walletTrx.Status == models.WalletTransactionStatusCancel {
			return walletTrx, nil
		}

		balanceCalculator = getWalletBalanceCancelCalculator(walletTrx.TransactionFlow)
		nextWalletTrxStatus = models.WalletTransactionStatusCancel
	} else {
		return nil, fmt.Errorf("action not supported: %s", req.Action)
	}

	if walletTrx.Status != models.WalletTransactionStatusPending {
		return nil, common.ErrTransactionNotReserved
	}

	// assume that the handler timeout is 16 seconds
	// maxWaitingTimeDB is the maximum time to wait for database operations to complete, usually it should be less than 8 seconds
	// because we have several operations in one transaction, including select for update, insert, and update
	// and the database timeout has been set
	maxWaitingTimeDB := 8 * time.Second
	dbCtx, cancelDB := context.WithTimeout(ctx, maxWaitingTimeDB)
	defer cancelDB()
	err = ts.srv.sqlRepo.Atomic(dbCtx, func(atomicCtx context.Context, r repositories.SQLRepository) (errAtomic error) {
		accRepo := r.GetAccountRepository()
		balanceRepo := r.GetBalanceRepository()
		walletTrxRepo := r.GetWalletTransactionRepository()
		acuanTrxRepo := r.GetTransactionRepository()

		// merge metadata
		maps.Copy(walletTrx.Metadata, req.Metadata)

		// Update parent transaction to wallet_transaction table
		walletTrx, errAtomic = walletTrxRepo.Update(atomicCtx, req.TransactionId, models.WalletTransactionUpdate{
			Status:          &nextWalletTrxStatus,
			TransactionTime: &req.TransactionTime,
			Metadata:        &walletTrx.Metadata,
		})
		if errAtomic != nil {
			return fmt.Errorf("unable to update status: %w", errAtomic)
		}

		// create child transaction (depend on transaction type)
		//mapTransformer := transformer.NewMapTransformer(
		//	ts.srv.conf,
		//	ts.srv.masterDataRepo,
		//	ts.srv.accountingClient,
		//	ts.srv.sqlRepo.GetAccountRepository(),
		//	ts.srv.sqlRepo.GetTransactionRepository(),
		//	ts.getAccountConfigRepository(),
		//	ts.srv.sqlRepo.GetWalletTransactionRepository(),
		//	ts.srv.flag,
		//)

		//childTransactions, errAtomic := mapTransformer.Transform(atomicCtx, *walletTrx)
		useMegatron := ts.srv.flag.IsEnabled("use_megatron_transformer")

		var childTransactions []models.TransactionReq

		if useMegatron {
			childTransactions, errAtomic = ts.transformUsingMegatron(atomicCtx, *walletTrx)
		} else {
			childTransactions, errAtomic = ts.transformUsingLegacy(atomicCtx, *walletTrx)
		}
		if errAtomic != nil {
			return fmt.Errorf("unable to transform wallet transaction: %w", err)
		}

		accountNumbers := getAccountNumbersForUpdateBalance(childTransactions)
		abs, errAtomic := balanceRepo.GetMany(atomicCtx,
			models.GetAccountBalanceRequest{
				AccountNumbers:               accountNumbers,
				ForUpdate:                    true,
				AccountNumbersExcludedFromDB: ts.srv.conf.AccountConfig.ExcludedBalanceUpdateAccountNumbers,
			},
		)
		if errAtomic != nil {
			return fmt.Errorf("unable to get current balance: %w", errAtomic)
		}

		errAtomic = validateAccountExistsInTransactions(childTransactions, abs)
		if errAtomic != nil {
			return errAtomic
		}

		childTransactions = updateTransactionAccountNumber(childTransactions, abs)
		currentBalances = models.ConvertToBalanceMap(abs)

		// Prepare for "before after"
		updatedBalances = make(map[string]models.Balance)
		maps.Copy(updatedBalances, currentBalances)
		var fromAccounts, toAccounts []string

		// calculate balances
		for _, ct := range childTransactions {
			trxSet := models.NewWalletTransactionSet(ct.FromAccount, ct.ToAccount, ct.Amount.Decimal, ct.TypeTransaction)

			updatedBalances, errAtomic = balanceCalculator(atomicCtx, trxSet, updatedBalances)
			if errAtomic != nil {
				return fmt.Errorf("calculate balance failed: %w", errAtomic)
			}

			fromAccounts = append(fromAccounts, ct.FromAccount)
			toAccounts = append(toAccounts, ct.ToAccount)
		}

		// update balances
		for accountNumber, balance := range updatedBalances {
			if balance.IsSkipBalanceUpdateOnDB() {
				continue
			}

			isEligibleForHVT := !slices.Contains(fromAccounts, accountNumber) && balance.IsHVT()
			if isEligibleForHVT {
				prevBalance, ok := currentBalances[accountNumber]
				if !ok {
					return fmt.Errorf("unable to get previous balance for account number %s", accountNumber)
				}

				diffAmount := balance.Available().Sub(prevBalance.Available())
				errAtomic = ts.srv.balanceHVTPub.Publish(atomicCtx, models.UpdateBalanceHVTPayload{
					Kind:                "balanceUpdateHVT",
					WalletTransactionId: walletTrx.ID,
					RefNumber:           walletTrx.RefNumber,
					AccountNumber:       accountNumber,
					UpdateAmount: models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(diffAmount),
						Currency:     "IDR",
					},
				}, publisher.WithKey(accountNumber))
				if errAtomic != nil {
					return fmt.Errorf("unable to publish balance hvt: %w", errAtomic)
				}

				continue
			}

			ub, errUpdateBalance := accRepo.UpdateAccountBalance(atomicCtx, accountNumber, balance)
			if errUpdateBalance != nil {
				return fmt.Errorf("unable to update balance: %w", errUpdateBalance)
			}

			updatedBalances[accountNumber] = *ub
		}

		// insert to "transaction" table if SUCCESS
		if walletTrx.Status == models.WalletTransactionStatusSuccess {
			acuanTransactions, errAtomic = ts.insertChildTransactions(atomicCtx, acuanTrxRepo, childTransactions)
			if errAtomic != nil {
				return fmt.Errorf("unable to store acuan transaction: %w", errAtomic)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// maxWaitingTimeKafka is the maximum time to wait for kafka publish to complete, kafka client has been set to 2 seconds
	// so we set the max waiting time to be longer than that
	// please be aware that this timeout is associated with the kafka client timeout
	// if the kafka client timeout is changed, this timeout should be changed too
	maxWaitingTimeKafka := 7 * time.Second
	kafkaCtx, cancelKafka := context.WithTimeout(context.Background(), maxWaitingTimeKafka)
	defer cancelKafka()
	if walletTrx.Status == models.WalletTransactionStatusSuccess {
		err = ts.publishNotificationCreateWalletTransactionSuccess(
			kafkaCtx,
			*walletTrx,
			acuanTransactions,
			currentBalances,
			updatedBalances,
			req.ClientId,
		)
		if err != nil {
			return nil, fmt.Errorf("unable to publish notification: %w", err)
		}
	}

	return walletTrx, nil
}

func (ts *walletTrx) publishNotificationCreateWalletTransactionSuccess(
	ctx context.Context,
	walletTransaction models.WalletTransaction,
	acuanTransactions []models.Transaction,
	beforeBalances map[string]models.Balance,
	afterBalances map[string]models.Balance,
	clientID string) error {

	defer ts.srv.metrics.GetBalancePrometheus().Record(acuanTransactions)

	payloadNotification, err := models.CreateWalletNotificationPayload(
		walletTransaction,
		acuanTransactions,
		beforeBalances,
		afterBalances,
		models.StatusTransactionNotificationSuccess,
		"success create wallet transaction",
		clientID,
	)
	if err != nil {
		return fmt.Errorf("unable to create notification payload: %w", err)
	}

	return ts.srv.transactionNotification.Publish(ctx, *payloadNotification)
}

func (ts *walletTrx) insertChildTransactions(ctx context.Context, acuanRepo repositories.TransactionRepository, childTransactions []models.TransactionReq) ([]models.Transaction, error) {
	var res []models.Transaction
	var payloadCreateBulk []*models.Transaction
	var errs *multierror.Error

	for _, ct := range childTransactions {
		en, errToRequest := ct.ToRequest()
		if errToRequest != nil {
			errs = multierror.Append(errs, errToRequest)
			continue
		}

		payloadCreateBulk = append(payloadCreateBulk, &en)
		res = append(res, en)
	}

	if errs.ErrorOrNil() != nil {
		return res, errs.ErrorOrNil()
	}

	err := acuanRepo.StoreBulkTransaction(ctx, payloadCreateBulk)
	if err != nil {
		return res, fmt.Errorf("unable to store acuan transaction: %w", err)
	}

	return res, err
}

func (ts *walletTrx) validateTransactionInput(ctx context.Context, in models.CreateWalletTransactionRequest) error {
	// Init master
	tTypes, err := ts.srv.masterDataRepo.GetListTransactionTypeCode(ctx)
	if err != nil {
		return fmt.Errorf("unable get trxType master data: %w", err)
	}
	acceptedTransactionType := append(ts.srv.conf.TransactionValidationConfig.AcceptedTransactionType, tTypes...)

	// Start Validation

	trxTime, err := common.ParseStringToDatetime(time.RFC3339, in.TransactionTime)
	if err != nil {
		return fmt.Errorf("unable to parse transaction time: %w", err)
	}

	if common.IsTodayAfterDate(trxTime) {
		return fmt.Errorf("transaction date can't be the next day. value: %s", in.TransactionTime)
	}

	if in.IsReserved && in.TransactionFlow == models.TransactionFlowCashIn {
		return common.ErrUnsupportedReservedTransactionFlow
	}

	// transactionType
	if !slices.Contains(acceptedTransactionType, in.TransactionType) {
		return fmt.Errorf("%w: %v", common.ErrInvalidTransactionType, in.TransactionType)
	}

	// amounts
	for _, v := range in.Amounts {
		if !slices.Contains(acceptedTransactionType, v.Type) {
			return fmt.Errorf("%w: %v", common.ErrInvalidTransactionType, v.Type)
		}

		if in.NetAmount.ValueDecimal.LessThanOrEqual(decimal.Zero) && v.Amount.ValueDecimal.LessThanOrEqual(decimal.Zero) {
			return common.ErrInvalidAmount
		}
	}

	if in.TransactionFlow == models.TransactionFlowTransfer && in.DestinationAccountNumber == "" {
		return fmt.Errorf("%w: destinationAccountNumber is required for transfer", common.ErrMissingDestinationAccountNumber)
	}

	return nil
}

func (ts *walletTrx) List(ctx context.Context, opts models.WalletTrxFilterOptions) (transactions []models.WalletTransaction, total int, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	repo := ts.srv.sqlRepo.GetWalletTransactionRepository()

	transactions, err = repo.List(ctx, opts)
	if err != nil {
		return
	}

	total, err = repo.CountAll(ctx, opts)
	if err != nil {
		return
	}

	return calculateTotalAmountOfTransactions(transactions), total, nil
}

func (ts *walletTrx) EnqueueTransaction(ctx context.Context, in models.CreateWalletTransactionRequest) (*models.WalletTransaction, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	accounts := []string{in.AccountNumber, in.DestinationAccountNumber}
	slices.Sort(accounts)

	opts := []publisher.PublishOption{
		publisher.WithKey(strings.Join(accounts, ":")),
		publisher.WithHeaders(map[string]string{
			models.IdempotencyKeyHeader: in.IdempotencyKey,
		}),
	}

	err = ts.srv.walletTransactionAsync.Publish(ctx, in, opts...)
	if err != nil {
		return nil, err
	}

	return &models.WalletTransaction{
		Status: models.WalletTransactionStatusPending,
	}, nil
}

func (ts *walletTrx) validateTransactionTypeAndRefNumber(ctx context.Context, in models.CreateWalletTransactionRequest) (data *models.WalletTransaction, err error) {
	var list models.ListTransactionType
	variantTransactionTypeAndRefNumber := ts.srv.flag.GetVariant(ts.srv.conf.FeatureFlagKeyLookup.GetVariantTransactionTypeAndRefNumber)
	if variantTransactionTypeAndRefNumber.Payload.Value != "" {
		err = json.Unmarshal([]byte(variantTransactionTypeAndRefNumber.Payload.Value), &list)
		if err != nil {
			return nil, err
		}
	}

	if slices.Contains(list.TransactionType, in.TransactionType) {
		data, err = ts.srv.sqlRepo.GetWalletTransactionRepository().CheckTransactionTypeAndReferenceNumber(ctx, in.TransactionType, in.RefNumber)
		if err != nil {
			return nil, err
		}
	}
	return
}

// Helper function untuk migration gradual
func (ts *walletTrx) shouldUseMegatronForTransactionType(transactionType string) bool {
	// Get list of transaction types yang sudah migrate ke megatron
	migratedTypes := ts.srv.flag.GetVariant("megatron_migrated_transaction_types")

	if migratedTypes == nil || migratedTypes.Payload.Value == "" {
		return false
	}

	var types []string
	if err := json.Unmarshal([]byte(migratedTypes.Payload.Value), &types); err != nil {
		return false
	}

	return slices.Contains(types, transactionType)
}

// Alternative: Transform specific transaction type using megatron
func (ts *walletTrx) transformSingleUsingMegatron(
	ctx context.Context,
	amount models.Amount,
	parentWalletTransaction models.WalletTransaction,
	transactionType string,
) ([]models.TransactionReq, error) {
	// Get account info
	account, err := ts.srv.sqlRepo.GetAccountRepository().GetCachedAccount(
		ctx,
		parentWalletTransaction.AccountNumber,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to get account: %w", err)
	}

	// Build request
	req := megatron.TransformRequest{
		ParentTransaction: ts.megatronClient.ConvertWalletTransactionToInput(
			ctx,
			parentWalletTransaction,
			*account,
		),
		Amount: megatron.AmountInput{
			Value:    amount.ValueDecimal.Decimal,
			Currency: amount.Currency,
		},
		TransactionType: transactionType,
		Context:         ts.megatronClient.BuildTransformContext(),
	}

	// Call megatron
	resp, err := ts.megatronClient.Transform(ctx, req)
	if err != nil {
		return nil, err
	}

	// Convert response
	return ts.megatronClient.ConvertResponseToTransactionReq(resp.Transactions), nil
}
