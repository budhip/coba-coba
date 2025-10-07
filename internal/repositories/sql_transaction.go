package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

	"github.com/lib/pq"
)

type TransactionRepository interface {
	Store(ctx context.Context, en *models.Transaction) (err error)
	StoreBulkTransaction(ctx context.Context, en []*models.Transaction) (err error)
	CheckRefNumbers(ctx context.Context, refNumbers ...string) (exists map[string]bool, err error)
	GetByID(ctx context.Context, id uint64) (en *models.Transaction, err error)
	GetByTransactionTypeAndRefNumber(ctx context.Context, req *models.TransactionGetByTypeAndRefNumberRequest) (*models.GetTransactionOut, error)
	GetList(ctx context.Context, opts models.TransactionFilterOptions) ([]models.Transaction, error)
	GetStatusCount(ctx context.Context, threshold uint, opts models.TransactionFilterOptions) (out models.StatusCountTransaction, err error)
	CountAll(ctx context.Context, opts models.TransactionFilterOptions) (total int, err error)
	GetTrxId(ctx context.Context, id int64) (object models.Transaction, err error)
	StreamAll(ctx context.Context, opts models.TransactionFilterOptions) <-chan models.TransactionStreamResult
	GetByTransactionID(ctx context.Context, transactionId string) (trx *models.Transaction, err error)
	UpdateStatus(ctx context.Context, id uint64, status string) (trx *models.Transaction, err error)
	GetReportRepayment(ctx context.Context, startDate, endDate time.Time) ([]models.ReportRepayment, error)
	ColectRepayment(ctx context.Context, date time.Time) (res *models.CollectRepayment, err error)
}

type transactionRepository sqlRepo

var _ TransactionRepository = (*transactionRepository)(nil)

func (tr *transactionRepository) Store(ctx context.Context, en *models.Transaction) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := tr.r.extractTxWrite(ctx)

	var transactionID int
	err = db.
		QueryRowContext(ctx, storeTrxQuery,
			en.TransactionID,
			en.FromAccount,
			en.ToAccount,
			en.FromNarrative,
			en.ToNarrative,
			en.RefNumber,
			en.Amount,
			en.Status,
			en.Method,
			en.TypeTransaction,
			en.OrderTime,
			en.OrderType,
			en.TransactionDate,
			en.TransactionTime,
			en.Currency,
			en.Description,
			en.Metadata).
		Scan(&transactionID,
			&en.CreatedAt,
			&en.UpdatedAt)
	if err != nil {
		return
	}

	return
}

func (tr *transactionRepository) StoreBulkTransaction(ctx context.Context, en []*models.Transaction) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := tr.r.extractTxWrite(ctx)

	valueStrings := []string{}
	valueArgs := []interface{}{}

	for _, req := range en {
		valueStrings = append(valueStrings, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
		valueArgs = append(valueArgs, req.TransactionID)
		valueArgs = append(valueArgs, req.TransactionDate)
		valueArgs = append(valueArgs, req.FromAccount)
		valueArgs = append(valueArgs, req.ToAccount)
		valueArgs = append(valueArgs, req.FromNarrative)
		valueArgs = append(valueArgs, req.ToNarrative)
		valueArgs = append(valueArgs, req.Amount)
		valueArgs = append(valueArgs, req.Status)
		valueArgs = append(valueArgs, req.Method)
		valueArgs = append(valueArgs, req.TypeTransaction)
		valueArgs = append(valueArgs, req.Description)
		valueArgs = append(valueArgs, req.RefNumber)
		valueArgs = append(valueArgs, req.Metadata)
		valueArgs = append(valueArgs, req.OrderTime)
		valueArgs = append(valueArgs, req.OrderType)
		valueArgs = append(valueArgs, req.TransactionTime)
		valueArgs = append(valueArgs, req.Currency)
	}

	storeTrxQueryBulk := fmt.Sprintf(`INSERT INTO "transaction" ("transactionId", "transactionDate", "fromAccount", "toAccount", "fromNarrative", "toNarrative", 
		"amount", "status", "method", "typeTransaction", "description", "refNumber", "metadata", "orderTime", "orderType", "transactionTime", "currency") VALUES %s`, strings.Join(valueStrings, ","))

	sqlStr := common.ReplaceSQL(storeTrxQueryBulk, "?")

	//var transactionID int
	if _, err = db.ExecContext(ctx, sqlStr, valueArgs...); err != nil {
		return err
	}

	return
}

func (tr *transactionRepository) CheckRefNumbers(ctx context.Context, refNumbers ...string) (exists map[string]bool, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := tr.r.extractTxWrite(ctx)

	exists = make(map[string]bool)
	for _, an := range refNumbers {
		exists[an] = false
	}

	rows, err := db.QueryContext(ctx, queryCheckByRefNumbers, pq.Array(refNumbers))
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		var rn string
		err = rows.Scan(&rn)
		if err != nil {
			return nil, err
		}

		exists[rn] = true
	}

	return exists, nil
}

func (tr *transactionRepository) GetByID(ctx context.Context, id uint64) (en *models.Transaction, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := tr.r.extractTxWrite(ctx)

	en = &models.Transaction{}
	err = db.QueryRowContext(ctx, getByIDQuery, id).Scan(
		&en.ID,
		&en.TransactionID,
		&en.TransactionDate,
		&en.TransactionTime,
		&en.FromAccount,
		&en.ToAccount,
		&en.FromNarrative,
		&en.ToNarrative,
		&en.Amount,
		&en.Status,
		&en.Method,
		&en.TypeTransaction,
		&en.Description,
		&en.RefNumber,
		&en.OrderTime,
		&en.OrderType,
		&en.Currency,
		&en.Metadata,
		&en.CreatedAt,
		&en.UpdatedAt)
	if err != nil {
		return
	}

	en.TransactionTime = en.TransactionTime.In(common.GetLocation())
	en.OrderTime = en.OrderTime.In(common.GetLocation())
	en.CreatedAt = en.CreatedAt.In(common.GetLocation())
	en.UpdatedAt = en.UpdatedAt.In(common.GetLocation())

	return
}

func (tr *transactionRepository) GetList(ctx context.Context, opts models.TransactionFilterOptions) ([]models.Transaction, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := tr.r.extractTxWrite(ctx)

	query, args, err := buildListTransactionQuery(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	var result []models.Transaction
	for rows.Next() {
		var trx = models.Transaction{}
		var err = rows.Scan(
			&trx.ID,
			&trx.TransactionID,
			&trx.RefNumber,
			&trx.OrderType,
			&trx.Method,
			&trx.TypeTransaction,
			&trx.TransactionDate,
			&trx.TransactionTime,
			&trx.FromAccount,
			&trx.FromAccountProductTypeName,
			&trx.FromAccountName,
			&trx.ToAccount,
			&trx.ToAccountProductTypeName,
			&trx.ToAccountName,
			&trx.Amount,
			&trx.Status,
			&trx.Description,
			&trx.Metadata,
			&trx.CreatedAt,
			&trx.UpdatedAt,
			&trx.Currency,
		)
		if err != nil {
			return nil, err
		}
		trx.Status = models.MapTransactionStatus[models.TransactionStatus(trx.Status)]
		result = append(result, trx)
	}
	if rows.Err() != nil {
		return result, err
	}

	return result, nil
}

func (tr *transactionRepository) GetStatusCount(ctx context.Context, threshold uint, opts models.TransactionFilterOptions) (out models.StatusCountTransaction, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := tr.r.extractTxWrite(ctx)

	query, args, err := buildStatusCountTransactionQuery(threshold, opts)
	if err != nil {
		return out, fmt.Errorf("failed to build query: %w", err)
	}

	err = db.QueryRowContext(ctx, query, args...).Scan(&out.ExceedThreshold)
	if err != nil {
		return out, err
	}

	out.Threshold = threshold

	return
}

func (tr *transactionRepository) CountAll(ctx context.Context, opts models.TransactionFilterOptions) (total int, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := tr.r.extractTxWrite(ctx)

	// TODO: change this query using estimation count by using explain analyze
	//query, args, err := buildCountTransactionQuery(opts)
	//if err != nil {
	//	return total, fmt.Errorf("failed to build query: %w", err)
	//}

	if err = db.QueryRowContext(ctx, queryEstimateCountData).Scan(&total); err != nil {
		return
	}

	return
}

func (tr *transactionRepository) GetTrxId(ctx context.Context, id int64) (object models.Transaction, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := tr.r.extractTxWrite(ctx)

	err = db.QueryRowContext(ctx, findTrxById, id).Scan(
		&object.ID,
		&object.FromAccount,
		&object.ToAccount,
		&object.FromNarrative,
		&object.ToNarrative,
		&object.TransactionDate,
		&object.Amount,
		&object.Status,
		&object.Method,
		&object.TypeTransaction,
		&object.Description,
		&object.RefNumber,
		&object.Metadata,
	)
	if err != nil {
		return
	}

	return
}

func (tr *transactionRepository) StreamAll(ctx context.Context, opts models.TransactionFilterOptions) <-chan models.TransactionStreamResult {
	db := tr.r.extractTxWrite(ctx)
	ch := make(chan models.TransactionStreamResult)

	go func() {
		defer close(ch)

		query, args, err := buildListTransactionQuery(opts)
		if err != nil {
			ch <- models.TransactionStreamResult{Err: err}
			return
		}

		rows, err := db.QueryContext(ctx, query, args...)
		if err != nil {
			ch <- models.TransactionStreamResult{Err: err}
			return
		}
		defer rows.Close()
		for rows.Next() {
			select {
			case <-ctx.Done():
				return
			default:
				var value = models.Transaction{}
				var err = rows.Scan(
					&value.ID,
					&value.TransactionID,
					&value.RefNumber,
					&value.OrderType,
					&value.Method,
					&value.TypeTransaction,
					&value.TransactionDate,
					&value.TransactionTime,
					&value.FromAccount,
					&value.FromAccountProductTypeName,
					&value.FromAccountName,
					&value.ToAccount,
					&value.ToAccountProductTypeName,
					&value.ToAccountName,
					&value.Amount,
					&value.Status,
					&value.Description,
					&value.Metadata,
					&value.CreatedAt,
					&value.UpdatedAt,
					&value.Currency,
				)
				if err != nil {
					ch <- models.TransactionStreamResult{Err: err}
					return
				}

				value.Status = models.MapTransactionStatus[models.TransactionStatus(value.Status)]

				ch <- models.TransactionStreamResult{Data: value}
			}
		}
	}()

	return ch
}

func (tr *transactionRepository) GetByTransactionTypeAndRefNumber(ctx context.Context, req *models.TransactionGetByTypeAndRefNumberRequest) (*models.GetTransactionOut, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := tr.r.extractTxWrite(ctx)

	result := models.GetTransactionOut{}
	err = db.QueryRowContext(ctx, queryGetByTransactionTypeAndRefNumber, req.TransactionType, req.RefNumber).
		Scan(
			&result.TransactionID,
			&result.RefNumber,
			&result.OrderType,
			&result.Method,
			&result.TransactionType,
			&result.TransactionDate,
			&result.TransactionTime,
			&result.FromAccount,
			&result.ToAccount,
			&result.Amount,
			&result.Status,
			&result.Description,
			&result.Metadata,
			&result.CreatedAt,
			&result.UpdatedAt,
		)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// GetByTransactionID will get transaction by transactionId.
func (tr *transactionRepository) GetByTransactionID(ctx context.Context, transactionId string) (trx *models.Transaction, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := tr.r.extractTxWrite(ctx)

	trx = &models.Transaction{}
	err = db.QueryRowContext(ctx, queryTransactionByTransactionID, transactionId).Scan(
		&trx.ID,
		&trx.TransactionID,
		&trx.TransactionDate,
		&trx.FromAccount,
		&trx.ToAccount,
		&trx.FromNarrative,
		&trx.ToNarrative,
		&trx.Amount,
		&trx.Status,
		&trx.Method,
		&trx.TypeTransaction,
		&trx.Description,
		&trx.RefNumber,
		&trx.Metadata,
		&trx.CreatedAt,
		&trx.UpdatedAt)
	if err != nil {
		return
	}

	return
}

// UpdateStatus will update transaction status based on ID.
func (tr *transactionRepository) UpdateStatus(ctx context.Context, id uint64, status string) (trx *models.Transaction, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := tr.r.extractTxWrite(ctx)

	res, err := db.ExecContext(ctx, queryUpdateTransactionStatus, status, id)
	if err != nil {
		return
	}

	affectedRows, err := res.RowsAffected()
	if err != nil {
		return
	}

	if affectedRows == 0 {
		return nil, common.ErrNoRowsAffected
	}

	trx, err = tr.GetByID(ctx, id)
	return
}

func (tr *transactionRepository) GetReportRepayment(ctx context.Context, startDate, endDate time.Time) ([]models.ReportRepayment, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := tr.r.extractTxRead(ctx)

	rows, err := db.QueryContext(ctx, queryReportRepayment, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to run query: %w", err)
	}
	defer rows.Close()

	var result []models.ReportRepayment
	for rows.Next() {
		var rr models.ReportRepayment
		err = rows.Scan(
			&rr.TransactionDate,
			&rr.Outstanding,
			&rr.Principal,
			&rr.Amartha,
			&rr.Lender,
			&rr.PPN,
			&rr.PPh,
			&rr.Total,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, rr)
	}
	if err := rows.Err(); err != nil {
		return result, err
	}

	return result, nil
}

func (tr *transactionRepository) ColectRepayment(ctx context.Context, date time.Time) (res *models.CollectRepayment, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	db := tr.r.extractTxRead(ctx)

	res = &models.CollectRepayment{}
	err = db.QueryRowContext(ctx, queryCollectRepayment, date).Scan(
		&res.TransactionDate,
		&res.Outstanding,
		&res.Principal,
		&res.Amartha,
		&res.Lender,
		&res.PPN,
		&res.PPh,
	)

	if err != nil {
		return res, fmt.Errorf("failed to run query: %w", err)
	}

	return res, nil
}
