package services

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/acuanclient"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	goAcuanLibModel "bitbucket.org/Amartha/go-acuan-lib/model"

	"github.com/hashicorp/go-multierror"
	// TODO: Change to "slices" in go 1.21
)

// newFileTransactionFromString parses a CSV string into a `FileTransaction` structure and validates its fields.
func (s *file) newFileTransactionFromString(ctx context.Context, result string) (*models.FileTransaction, error) {
	// CSV reader
	reader := csv.NewReader(strings.NewReader(result))

	// Define delimiter; default is comma (,)
	semiColonIndex := strings.Index(result, ";")
	if semiColonIndex != -1 {
		reader.Comma = ';'
	}

	// Read
	record, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read csv row: %w", err)
	}

	if len(record) <= 8 {
		return nil, fmt.Errorf("column length mismatch with model type. value: %s", result)
	}

	// Validation
	var parseErr *multierror.Error
	trxDate, err := common.ParseStringToDatetime(common.DateFormatDDMMMYYYY, record[0])
	if err != nil {
		parseErr = multierror.Append(parseErr, fmt.Errorf("unable to parse amount: %v", err))
	}

	orderType := strings.ToUpper(record[1])
	if err = s.srv.MasterData.EnsureOrderTypeExist(ctx, []string{orderType}); err != nil {
		parseErr = multierror.Append(parseErr, fmt.Errorf("invalid order type: %s", orderType))
	}

	trxType := record[8]
	if err = s.srv.MasterData.EnsureTransactionTypeExist(ctx, []string{trxType}); err != nil {
		parseErr = multierror.Append(parseErr, fmt.Errorf("invalid transaction type: %s", trxType))
	}

	amount, err := common.NewDecimalFromString(record[2])
	if err != nil {
		parseErr = multierror.Append(parseErr, fmt.Errorf("unable to parse amount: %v", err))
	}

	if parseErr != nil {
		return nil, parseErr.ErrorOrNil()
	}

	return &models.FileTransaction{
		TransactionDate:      &trxDate,
		OrderType:            orderType,
		Amount:               amount,
		Currency:             record[3],
		SourceAccountId:      record[4],
		DestinationAccountId: record[5],
		Description:          record[6],
		Method:               record[7],
		TransactionType:      trxType,
	}, nil
}

// publishOrderToACuan parses transaction data, constructs a payload, and publishes it to ACuan.
func (s *file) publishOrderToACuan(ctx context.Context, txt, refNumber string) error {
	if txt == "" || s.isTransactionCSVHeader(txt) {
		return nil
	}

	// Parse
	trx, err := s.newFileTransactionFromString(ctx, txt)
	if err != nil {
		return fmt.Errorf("unable to parse transaction: %v", err)
	}

	accountRepo := s.srv.sqlRepo.GetAccountRepository()
	es, err := accountRepo.CheckAccountNumbers(ctx, []string{trx.SourceAccountId, trx.DestinationAccountId})
	if err != nil {
		return fmt.Errorf("unable to check account numbers: %w", err)
	}

	var errAccount *multierror.Error
	for accountNumber, exists := range es {
		if !exists {
			errAccount = multierror.Append(errAccount, fmt.Errorf("account number %s not found", accountNumber))
		}
	}

	if errAccount.ErrorOrNil() != nil {
		return errAccount.ErrorOrNil()
	}

	// Publish
	payload := acuanclient.PublishTransactionRequest{
		FromAccount:     trx.SourceAccountId,
		ToAccount:       trx.DestinationAccountId,
		Amount:          *trx.Amount,
		Method:          goAcuanLibModel.TransactionMethod(trx.Method),
		TransactionType: goAcuanLibModel.TransactionType(trx.TransactionType),
		TransactionTime: *trx.TransactionDate,
		OrderType:       trx.OrderType,
		RefNumber:       refNumber,
		Description:     trx.Description,
		Currency:        trx.Currency,
	}
	if err = s.srv.acuanClient.PublishTransaction(ctx, payload); err != nil {
		return fmt.Errorf("unable to publish ACuan: %w", err)
	}

	return nil
}

// isCSVHeader checks if the provided string `txt` starts with "transactionDate".
func (s *file) isTransactionCSVHeader(txt string) bool {
	return strings.HasPrefix(txt, "transactionDate")
}

// validateInputStoreTransaction validates the input of `StoreTransaction` service.
func (ts *transaction) validateInputStoreTransaction(ctx context.Context, processType models.TransactionStoreProcessType, req models.TransactionReq) error {
	var errs *multierror.Error

	isProcessPending := processType == models.TransactionStoreProcessReserved
	if isProcessPending && req.Status != string(models.TransactionStatusPending) {
		errs = multierror.Append(errs, common.ErrInvalidStatus)
	}

	if !req.Amount.Valid {
		errs = multierror.Append(errs, common.ErrInvalidAmount)
	}

	if req.Amount.Decimal.LessThanOrEqual(decimal.Zero) {
		errs = multierror.Append(errs, common.ErrInvalidAmount)
	}

	tTypes, err := ts.srv.masterDataRepo.GetListTransactionTypeCode(ctx)
	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("unable to get transaction type from master data: %v", err))
	}

	oTypes, err := ts.srv.masterDataRepo.GetListOrderTypeCode(ctx)
	if err != nil {
		errs = multierror.Append(errs, fmt.Errorf("unable to get order type from master data: %v", err))
	}

	acceptedOrderType := append(ts.srv.conf.TransactionValidationConfig.AcceptedOrderType, oTypes...)
	acceptedTransactionType := append(ts.srv.conf.TransactionValidationConfig.AcceptedTransactionType, tTypes...)

	if ok := slices.Contains(acceptedOrderType, req.OrderType); !ok {
		errs = multierror.Append(errs, fmt.Errorf("%w: %v", common.ErrInvalidOrderType, req.OrderType))
	}

	if ok := slices.Contains(acceptedTransactionType, req.TypeTransaction); !ok {
		errs = multierror.Append(errs, fmt.Errorf("%w: %v", common.ErrInvalidTransactionType, req.TypeTransaction))
	}

	return errs.ErrorOrNil()
}
