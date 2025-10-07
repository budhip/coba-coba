package repositories

import (
	sq "github.com/Masterminds/squirrel"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

var (
	queryReconToolHistoryCreate = `
		INSERT INTO recon_tool_history(
			"orderType", "transactionType", "transactionDate", "uploadedFilePath", "status", "createdAt", "updatedAt" 
		)
		VALUES(
			$1, $2, $3, $4, $5, NOW(), NOW()
		)
		RETURNING 
			"id", "transactionDate", "createdAt", "updatedAt";
	`

	queryReconToolHistoryDeleteByID = "DELETE FROM recon_tool_history WHERE id = $1"

	queryReconToolHistoryGetById = `SELECT 
		  "id",
		  "orderType",
		  "transactionType",
		  "transactionDate",
		  COALESCE("resultFilePath", '') as "resultFilePath",
		  COALESCE("uploadedFilePath", '') as "uploadedFilePath",
		  COALESCE("status", '') as "status",
		  "createdAt",
		  "updatedAt"
		FROM "recon_tool_history"
		WHERE id = $1;`

	queryReconToolHistoryUpdate = `UPDATE recon_tool_history
		SET 
		  "orderType" = $2,
		  "transactionType" = $3,
		  "transactionDate" = $4,
		  "uploadedFilePath" = $5,
		  "resultFilePath" = $6,
		  "status" = $7,
		  "updatedAt" = NOW()
		WHERE
		  id = $1`
)

func buildFilteredReconToolHistoryQuery(cols []string, opts models.ReconToolHistoryFilterOptions) sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.Select(cols...).From("recon_tool_history")

	if opts.OrderType != "" {
		query = query.Where(sq.Eq{`"orderType"`: opts.OrderType})
	}

	if opts.TransactionType != "" {
		query = query.Where(sq.Eq{`"transactionType"`: opts.TransactionType})
	}

	if opts.StartReconDate != nil {
		query = query.Where(sq.GtOrEq{`DATE("createdAt")`: opts.StartReconDate})
	}

	if opts.EndReconDate != nil {
		query = query.Where(sq.LtOrEq{`DATE("createdAt")`: opts.EndReconDate})
	}

	return query
}

func buildListReconToolHistoryQuery(opts models.ReconToolHistoryFilterOptions) (sql string, args []interface{}, err error) {
	columns := []string{
		`"id"`,
		`"orderType"`,
		`"transactionType"`,
		`"transactionDate"`,
		`COALESCE("resultFilePath", '') as "resultFilePath"`,
		`COALESCE("uploadedFilePath", '') as "uploadedFilePath"`,
		`COALESCE("status", '') as "status"`,
		`"createdAt"`,
		`"updatedAt"`,
	}

	query := buildFilteredReconToolHistoryQuery(columns, opts)

	if opts.AfterCreatedAt != nil {
		query = query.Where(sq.Lt{`"createdAt"`: opts.AfterCreatedAt})
	}

	if opts.BeforeCreatedAt != nil {
		query = query.Where(sq.Gt{`"createdAt"`: opts.BeforeCreatedAt})
	}

	if opts.AscendingOrder {
		query = query.OrderBy(`"createdAt" ASC`)
	} else {
		query = query.OrderBy(`"createdAt" DESC`)
	}

	query = query.Limit(uint64(opts.Limit))

	return query.ToSql()
}

func buildCountReconToolHistoryQuery(opts models.ReconToolHistoryFilterOptions) (sql string, args []interface{}, err error) {
	columns := []string{
		`count(1)`,
	}

	query := buildFilteredReconToolHistoryQuery(columns, opts)

	return query.ToSql()
}
