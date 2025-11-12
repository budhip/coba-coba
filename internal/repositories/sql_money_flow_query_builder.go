package repositories

import (
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	sq "github.com/Masterminds/squirrel"
)

// MoneyFlowQueryBuilder handles query building for money flow summaries
type MoneyFlowQueryBuilder struct {
	psql sq.StatementBuilderType
}

func NewMoneyFlowQueryBuilder() *MoneyFlowQueryBuilder {
	return &MoneyFlowQueryBuilder{
		psql: sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

// applyCommonFilters applies filters common to both list and count queries
func (qb *MoneyFlowQueryBuilder) applyCommonFilters(query sq.SelectBuilder, opts models.MoneyFlowSummaryFilterOptions) sq.SelectBuilder {
	// Always filter by transaction_source_creation_date < today
	query = query.Where(sq.Lt{`mfs."transaction_source_creation_date"`: time.Now().Truncate(24 * time.Hour)})

	if opts.PaymentType != "" {
		query = query.Where(sq.Eq{`mfs."payment_type"`: opts.PaymentType})
	}

	// Handle date range filtering
	if opts.TransactionSourceCreationDateStart != nil && opts.TransactionSourceCreationDateEnd != nil {
		// Both start and end dates provided - use BETWEEN
		query = query.Where(sq.And{
			sq.GtOrEq{`mfs."transaction_source_creation_date"`: opts.TransactionSourceCreationDateStart},
			sq.LtOrEq{`mfs."transaction_source_creation_date"`: opts.TransactionSourceCreationDateEnd},
		})
	} else if opts.TransactionSourceCreationDateStart != nil {
		// Only start date provided
		query = query.Where(sq.GtOrEq{`mfs."transaction_source_creation_date"`: opts.TransactionSourceCreationDateStart})
	} else if opts.TransactionSourceCreationDateEnd != nil {
		// Only end date provided
		query = query.Where(sq.LtOrEq{`mfs."transaction_source_creation_date"`: opts.TransactionSourceCreationDateEnd})
	}

	if opts.Status != "" {
		query = query.Where(sq.Eq{`mfs."money_flow_status"`: opts.Status})
	}

	return query
}

// applyCursorPagination applies cursor-based pagination
func (qb *MoneyFlowQueryBuilder) applyCursorPagination(query sq.SelectBuilder, cursor *models.MoneyFlowSummaryCursor, limit int) sq.SelectBuilder {
	if cursor != nil {
		if cursor.IsBackward {
			// Backward: get records GREATER than cursor, order ASC (then reverse in code)
			query = query.Where(sq.Gt{`mfs."id"`: cursor.ID}).OrderBy(`mfs."id" ASC`)
		} else {
			// Forward: get records LESS than cursor, order DESC
			query = query.Where(sq.Lt{`mfs."id"`: cursor.ID}).OrderBy(`mfs."id" DESC`)
		}
	} else {
		query = query.OrderBy(`mfs."id" DESC`)
	}

	if limit > 0 {
		query = query.Limit(uint64(limit))
	}

	return query
}

// BuildListQuery builds the query for fetching money flow summaries list with related summary support
func (qb *MoneyFlowQueryBuilder) BuildListQuery(opts models.MoneyFlowSummaryFilterOptions) (string, []interface{}, error) {
	columns := []string{
		`mfs."id"`,
		`mfs."transaction_source_creation_date"`,
		`mfs."payment_type"`,
		`mfs."total_transfer"`,
		`mfs."money_flow_status"`,
		`mfs."requested_date"`,
		`mfs."actual_date"`,
		`mfs."created_at"`,
		`mfs."related_failed_or_rejected_summary_id"`,
		`COALESCE(related."total_transfer", 0) as related_total_transfer`,
	}

	query := qb.psql.Select(columns...).
		From("money_flow_summaries as mfs").
		LeftJoin(`money_flow_summaries as related ON mfs."related_failed_or_rejected_summary_id" = related."id"`)

	query = qb.applyCommonFilters(query, opts)
	query = qb.applyCursorPagination(query, opts.Cursor, opts.Limit)

	return query.ToSql()
}

// BuildCountQuery builds the query for counting money flow summaries
func (qb *MoneyFlowQueryBuilder) BuildCountQuery(opts models.MoneyFlowSummaryFilterOptions) (string, []interface{}, error) {
	query := qb.psql.Select("COUNT(*)").From("money_flow_summaries as mfs")
	query = qb.applyCommonFilters(query, opts)

	return query.ToSql()
}

// applyDetailedTransactionFilters applies filters for detailed transactions
func (qb *MoneyFlowQueryBuilder) applyDetailedTransactionFilters(query sq.SelectBuilder, opts models.DetailedTransactionFilterOptions) sq.SelectBuilder {
	// Build WHERE condition to include both summaryID and relatedSummaryID if exists
	if opts.RelatedFailedOrRejectedSummaryID != nil && *opts.RelatedFailedOrRejectedSummaryID != "" {
		query = query.Where(sq.Or{
			sq.Eq{`dmfs."summary_id"`: opts.SummaryID},
			sq.Eq{`dmfs."summary_id"`: *opts.RelatedFailedOrRejectedSummaryID},
		})
	} else {
		query = query.Where(sq.Eq{`dmfs."summary_id"`: opts.SummaryID})
	}

	// Add refNumber filter if provided
	if opts.RefNumber != "" {
		query = query.Where(sq.Eq{`t."refNumber"`: opts.RefNumber})
	}

	return query
}

// BuildDetailedTransactionsQuery builds query for detailed transactions with related summary support
func (qb *MoneyFlowQueryBuilder) BuildDetailedTransactionsQuery(opts models.DetailedTransactionFilterOptions) (string, []interface{}, error) {
	columns := []string{
		`dmfs."id"`,
		`t."transactionId"`,
		`t."transactionDate"`,
		`t."refNumber"`,
		`t."typeTransaction"`,
		`t."fromAccount"`,
		`t."toAccount"`,
		`t."amount"`,
		`t."description"`,
		`COALESCE(t."metadata", '{}'::jsonb) as "metadata"`,
	}

	query := qb.psql.Select(columns...).
		From("detailed_money_flow_summaries as dmfs").
		InnerJoin(`transaction t ON t."transactionId" = dmfs.acuan_transaction_id`)

	query = qb.applyDetailedTransactionFilters(query, opts)

	// Apply cursor pagination for detailed transactions
	if opts.Cursor != nil {
		if opts.Cursor.IsBackward {
			// Backward: get records GREATER than cursor, order ASC (then reverse in code)
			query = query.Where(sq.Gt{`dmfs."id"`: opts.Cursor.ID}).OrderBy(`dmfs."id" ASC`)
		} else {
			// Forward: get records LESS than cursor, order DESC
			query = query.Where(sq.Lt{`dmfs."id"`: opts.Cursor.ID}).OrderBy(`dmfs."id" DESC`)
		}
	} else {
		query = query.OrderBy(`dmfs."id" DESC`)
	}

	if opts.Limit > 0 {
		query = query.Limit(uint64(opts.Limit))
	}

	return query.ToSql()
}

// BuildCountDetailedTransactionsQuery builds query for counting detailed transactions with related summary support
func (qb *MoneyFlowQueryBuilder) BuildCountDetailedTransactionsQuery(opts models.DetailedTransactionFilterOptions) (string, []interface{}, error) {
	query := qb.psql.Select("COUNT(*)").
		From("detailed_money_flow_summaries as dmfs").
		InnerJoin(`transaction t ON t."transactionId" = dmfs.acuan_transaction_id`)

	query = qb.applyDetailedTransactionFilters(query, opts)

	return query.ToSql()
}
