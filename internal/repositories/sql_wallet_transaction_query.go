package repositories

import (
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

var (
	queryWalletTrxCreate = `
		INSERT INTO "wallet_transaction"(
			"id", "accountNumber", "refNumber", "transactionType", "transactionFlow", "transactionTime", 
			"netAmount", "breakdownAmounts", "status", "destinationAccountNumber", "description",
			"metadata", "createdAt", "updatedAt"
		)
		VALUES(
			$1, $2, $3, $4, $5, $6,
			$7, $8, $9, NULLIF($10, ''), NULLIF($11, ''),
			$12, now(), now()
		)
		RETURNING
			"id", "status", "accountNumber", "destinationAccountNumber", "refNumber", 
			"transactionType", "transactionTime", "transactionFlow",
			"netAmount", "breakdownAmounts",
			"description", "metadata", "createdAt";
	`

	queryWalletTrxGetByID = `
		SELECT
			"id", "status", "accountNumber", "destinationAccountNumber", "refNumber", 
			"transactionType", "transactionTime", "transactionFlow",
			"netAmount", "breakdownAmounts",
			"description", "metadata", "createdAt"
		FROM "wallet_transaction"
		WHERE "id" = $1;
	`

	queryWalletTrxUpdateStatus = `
		UPDATE "wallet_transaction"
		SET "status" = $2, "updatedAt" = now()
		WHERE "id" = $1
		RETURNING
			"id", "status", "accountNumber", "destinationAccountNumber", "refNumber", 
			"transactionType", "transactionTime", "transactionFlow",
			"netAmount", "breakdownAmounts",
			"description", "metadata", "createdAt";
	`

	queryWalletTrxGetByRefNumber = `SELECT "id", "status" FROM "wallet_transaction" w WHERE w."refNumber" = $1;`
)

func buildListWalletTrxQuery(opts models.WalletTrxFilterOptions) (sql string, args []interface{}, err error) {
	columns := []string{
		`"id"`,
		`"status"`,
		`"accountNumber"`,
		`COALESCE("destinationAccountNumber", '') as "destinationAccountNumber"`,
		`"refNumber"`,
		`"transactionType"`,
		`"transactionTime"`,
		`"transactionFlow"`,
		`"netAmount"`,
		`"breakdownAmounts"`,
		`COALESCE("description", '') as "description"`,
		`"metadata"`,
		`"createdAt"`,
	}
	query := buildFilteredWalletTrxQuery(columns, opts)

	// Sort column
	sortColumnMap := map[string]string{
		"createdDate": "transactionTime",
	}
	sortColumn := sortColumnMap[opts.SortBy]
	if sortColumn == "" {
		sortColumn = "transactionTime"
	}

	// Sort direction
	sortDirection := opts.SortDirection
	if sortDirection != models.SortByDESC && sortDirection != models.SortByASC {
		sortDirection = models.SortByDESC
	}

	// Offset
	cursorValue := opts.Cursor
	if cursorValue != nil {
		operator := "<"
		if (sortDirection == models.SortByASC && !opts.Cursor.IsBackward) ||
			(sortDirection == models.SortByDESC && opts.Cursor.IsBackward) {
			operator = ">"
		}

		statement := fmt.Sprintf(
			`("transactionTime", "id") %s (?, ?)`,
			operator)
		query = query.Where(statement, opts.Cursor.TransactionTime, opts.Cursor.Id)
	}

	// Order
	getStatementOrder := func(direction string) string {
		return fmt.Sprintf(
			`"transactionTime" %s, "id" %s`,
			direction,
			direction,
		)
	}

	if opts.Cursor != nil && opts.Cursor.IsBackward {
		query = query.OrderBy(getStatementOrder(models.ReverseSortMap[sortDirection]))
	} else {
		query = query.OrderBy(getStatementOrder(sortDirection))
	}

	if opts.Limit >= 0 {
		query = query.Limit(uint64(opts.Limit))
	}

	return query.ToSql()
}

func buildCountWalletTrxQuery(opts models.WalletTrxFilterOptions) (sql string, args []interface{}, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	if len(opts.AccountNumbers) > 0 {
		// Build the UNION ALL query first
		unionQuery := buildFilteredWalletTrxQuery([]string{"*"}, opts)

		// Wrap UNION ALL query in a subquery to count results
		query := psql.
			Select("COUNT(1)").
			FromSelect(unionQuery, "count_trx")

		return query.ToSql()
	}

	query := buildFilteredWalletTrxQuery([]string{`COUNT(1)`}, opts)

	return query.ToSql()
}

func buildFilteredWalletTrxQuery(cols []string, opts models.WalletTrxFilterOptions) sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Select(cols...).
		From("wallet_transaction")

	if opts.AccountNumber != "" {
		query = query.Where(sq.Eq{`"accountNumber"`: opts.AccountNumber})
	}

	if opts.Status != "" {
		query = query.Where(sq.Eq{`"status"`: opts.Status})
	}

	if opts.TransactionType != "" {
		query = query.Where(sq.Eq{`"transactionType"`: opts.TransactionType})
	}

	if opts.StartDate != nil {
		query = query.Where(sq.GtOrEq{`"transactionTime"`: opts.StartDate})
	}

	if opts.EndDate != nil {
		query = query.Where(sq.LtOrEq{`"transactionTime"`: opts.EndDate})
	}

	if len(opts.TransactionTypes) > 0 {
		query = query.Where(sq.Eq{`"transactionType"`: opts.TransactionTypes})
	}

	if opts.RefNumber != "" {
		query = query.Where(sq.Eq{`"refNumber"`: opts.RefNumber})
	}

	if len(opts.AccountNumbers) > 0 {
		accountQuery := query.Where(sq.Eq{`wallet_transaction."accountNumber"`: opts.AccountNumbers})
		destinationQuery := query.Where(sq.Eq{`wallet_transaction."destinationAccountNumber"`: opts.AccountNumbers})

		unionQuery := accountQuery.Suffix("UNION").SuffixExpr(destinationQuery)

		return psql.Select("*").FromSelect(unionQuery, "union_trx")
	}

	return query
}

func buildUpdateWalletTrx(id string, data models.WalletTransactionUpdate) (sql string, args []interface{}, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.
		Update("wallet_transaction").
		Where(sq.Eq{"id": id})

	if data.TransactionTime != nil {
		query = query.Set(`"transactionTime"`, data.TransactionTime)
	}

	if data.Status != nil {
		query = query.Set(`"status"`, data.Status)
	}

	if data.Metadata != nil {
		query = query.Set(`"metadata"`, data.Metadata)
	}

	query = query.Suffix(`RETURNING
			"id", "status", "accountNumber", "destinationAccountNumber", "refNumber", 
			"transactionType", "transactionTime", "transactionFlow",
			"netAmount", "breakdownAmounts",
			"description", "metadata", "createdAt"`)

	return query.ToSql()
}
