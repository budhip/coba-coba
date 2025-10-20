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

	if opts.TransactionSourceCreationDate != nil {
		query = query.Where(sq.Eq{`mfs."transaction_source_creation_date"`: opts.TransactionSourceCreationDate})
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
			query = query.Where(sq.Lt{`mfs."id"`: cursor.ID}).OrderBy(`mfs."id" ASC`)
		} else {
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

// BuildListQuery builds the query for fetching money flow summaries list
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
	}

	query := qb.psql.Select(columns...).From("money_flow_summaries as mfs")
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

// BuildDetailedTransactionsQuery builds query for detailed transactions
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
		InnerJoin(`transaction t ON t."transactionId" = dmfs.acuan_transaction_id`).
		Where(sq.Eq{`dmfs."summary_id"`: opts.SummaryID})

	// Apply cursor pagination for detailed transactions
	if opts.Cursor != nil {
		if opts.Cursor.IsBackward {
			query = query.Where(sq.Gt{`dmfs."id"`: opts.Cursor.ID}).OrderBy(`dmfs."id" ASC`)
		} else {
			query = query.Where(sq.Gt{`dmfs."id"`: opts.Cursor.ID}).OrderBy(`dmfs."id" DESC`)
		}
	} else {
		query = query.OrderBy(`dmfs."id" DESC`)
	}

	if opts.Limit > 0 {
		query = query.Limit(uint64(opts.Limit))
	}

	return query.ToSql()
}
