package repositories

import (
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	sq "github.com/Masterminds/squirrel"
)

const (
	storeTrxQuery = `INSERT INTO "transaction" 
    	(
    	 "transactionId",
    	 "fromAccount",
    	 "toAccount",
    	 "fromNarrative",
    	 "toNarrative",
    	 "refNumber",
		 "amount",
    	 "status",
    	 "method",
    	 "typeTransaction",
    	 "orderTime",
    	 "orderType",
    	 "transactionDate",
    	 "transactionTime",
    	 "currency",
    	 "description",
    	 "metadata"
    	 ) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING id, "createdAt", "updatedAt"`

	getByIDQuery = `SELECT
						"id", 
						"transactionId",
						"transactionDate",
						"transactionTime",
						"fromAccount",
						"toAccount",
						"fromNarrative",
						"toNarrative",
						"amount",
						"status",
						"method",
						"typeTransaction",
						"description",
						"refNumber",
						"orderTime",
						"orderType",
						"currency",
						"metadata",
						"createdAt",
						"updatedAt"
					FROM "transaction" 
					WHERE "id" = $1 
					`

	queryTransactionByTransactionID = `SELECT "id", "transactionId", "transactionDate", "fromAccount", "toAccount", "fromNarrative", "toNarrative", "amount", "status", "method", "typeTransaction", "description", "refNumber", "metadata", "createdAt", "updatedAt"
		FROM "transaction" WHERE "transactionId" = $1;`

	queryUpdateTransactionStatus = `
		UPDATE "transaction"
		SET
			"status" = $1,
			"updatedAt" = now()
		WHERE "id" = $2;`

	queryGetByTransactionTypeAndRefNumber = `select 
		t."id",
		t."refNumber",
		t."orderType",
		t."method",
		t."typeTransaction",
		t."transactionDate",
		t."transactionTime",
		t."fromAccount",
		t."toAccount",
		t."amount",
		t."status",
		t."description",
		t."metadata",
		t."createdAt",
		t."updatedAt"
	from "transaction" t 
	where t."typeTransaction" = $1 and t."refNumber" = $2;`

	findTrxById = `SELECT id, "fromAccount", "toAccount", "fromNarrative", "toNarrative", "transactionDate", amount, 
			"status", "method", "typeTransaction", "description", "refNumber", "metadata" FROM "transaction" WHERE "id" = $1 `

	// get list of trx by ref numbers
	queryCheckByRefNumbers = `
		SELECT 
			"refNumber"
		FROM "transaction"
		WHERE
			"refNumber" = ANY($1)`

	queryEstimateCountData = `
		SELECT reltuples::bigint AS estimate
		FROM pg_class
		WHERE relname = 'transaction' or relname = 'transaction_default'
		order by reltuples desc
		limit 1;`

	queryCollectRepayment = `
WITH transaction_summary AS (
    SELECT
        "transactionDate" AS "transactionDate",
        SUM(CASE WHEN "typeTransaction" = 'RPYAE' THEN amount ELSE 0 END) AS "outstanding",
        SUM(CASE WHEN "typeTransaction" = 'RPYAD' THEN amount ELSE 0 END) AS "principal",
        SUM(CASE WHEN "typeTransaction" = 'RPYAF' THEN amount ELSE 0 END) AS "amartha",
        SUM(CASE WHEN "typeTransaction" = 'RPYAB' THEN amount ELSE 0 END) AS "lender",
        SUM(CASE WHEN "typeTransaction" = 'RPYAG' THEN amount ELSE 0 END) AS "ppn",
        SUM(CASE WHEN "typeTransaction" = 'RPYAC' THEN amount ELSE 0 END) AS "pph"
    FROM transaction
    WHERE "typeTransaction" IN ('RPYAE', 'RPYAD', 'RPYAF', 'RPYAB', 'RPYAG', 'RPYAC')
    AND "transactionDate" = $1::date
    GROUP BY "transactionDate"
)
SELECT "transactionDate", "outstanding", "principal", "amartha", "lender", "ppn", "pph"
FROM transaction_summary;
`

	queryReportRepayment = `
	WITH transaction_summary AS (
    SELECT
        "transactionDate" AS "transactionDate",
        SUM(CASE WHEN "typeTransaction" = 'RPYAE' THEN amount ELSE 0 END) AS "outstanding",
        SUM(CASE WHEN "typeTransaction" = 'RPYAD' THEN amount ELSE 0 END) AS "principal",
        SUM(CASE WHEN "typeTransaction" = 'RPYAF' THEN amount ELSE 0 END) AS "amartha",
        SUM(CASE WHEN "typeTransaction" = 'RPYAB' THEN amount ELSE 0 END) AS "lender",
        SUM(CASE WHEN "typeTransaction" = 'RPYAG' THEN amount ELSE 0 END) AS "ppn",
        SUM(CASE WHEN "typeTransaction" = 'RPYAC' THEN amount ELSE 0 END) AS "pph"
    FROM transaction
    WHERE "typeTransaction" IN ('RPYAE', 'RPYAD', 'RPYAF', 'RPYAB', 'RPYAG', 'RPYAC')
      AND "transactionDate" BETWEEN $1::date AND $2::date
    GROUP BY "transactionDate"
)
SELECT *,
       ("principal" + "amartha" + "lender" + "ppn" + "pph") AS "total"
FROM transaction_summary
ORDER BY "transactionDate" DESC;
`
)

func buildStreamAllTransactionQuery(opts models.TransactionStreamAllOptions) (sql string, args []interface{}, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	columns := []string{
		`"id"`,
		`"transactionDate"`,
		`"fromAccount"`,
		`"fromNarrative"`,
		`"toAccount"`,
		`"toNarrative"`,
		`"amount"`,
		`"status"`,
		`"method"`,
		`"typeTransaction"`,
		`"description"`,
		`"refNumber"`,
		`"createdAt"`,
		`"updatedAt"`,
		`"metadata"`,
		`"transactionId"`,
	}
	query := psql.Select(columns...).From("transaction")

	td := common.FormatDatetimeToString(opts.TransactionDate, common.DateFormatYYYYMMDD)
	query = query.Where(sq.Eq{`transaction."transactionDate"`: td})

	if opts.TransactionType != "" {
		query = query.Where(sq.Eq{`transaction."typeTransaction"`: opts.TransactionType})
	}

	query = query.OrderBy("id ASC")

	return query.ToSql()
}

func buildListTransactionQuery(opts models.TransactionFilterOptions) (sql string, args []interface{}, err error) {
	columns := []string{
		`transaction."id"`,
		`COALESCE(transaction."transactionId", '00000000-0000-0000-0000-000000000000') as "transactionId"`,
		`COALESCE(transaction."refNumber", '') as "refNumber"`,
		`COALESCE(transaction."orderType", '') as "orderType"`,
		`COALESCE(transaction."method", '') as "method"`,
		`COALESCE(transaction."typeTransaction", '') as "typeTransaction"`,
		`COALESCE(transaction."transactionDate", '1970-01-01'::date) as "transactionDate"`,
		`COALESCE(transaction."transactionTime", '1970-01-01 00:00:00+00'::timestamptz) as "transactionTime"`,
		`COALESCE(transaction."fromAccount", '') as "fromAccount"`,
		`COALESCE(af."productTypeName", '') as "fromAccountProductTypeName"`,
		`COALESCE(af."name", '') as "fromAccountName"`,
		`COALESCE(transaction."toAccount", '') as "toAccount"`,
		`COALESCE(at."productTypeName", '') as "toAccountProductTypeName"`,
		`COALESCE(at."name", '') as "toAccountName"`,
		`transaction."amount"`,
		`COALESCE(transaction."status", '') as "status"`,
		`COALESCE(transaction."description", '') as "description"`,
		`COALESCE(transaction."metadata", '{}'::jsonb) as "metadata"`,
		`COALESCE(transaction."createdAt", '1970-01-01 00:00:00+00'::timestamptz) as "createdAt"`,
		`COALESCE(transaction."updatedAt", '1970-01-01 00:00:00+00'::timestamptz) as "updatedAt"`,
		`COALESCE(transaction."currency", 'IDR') as "currency"`,
	}

	if opts.OnlyAMF {
		columns = []string{
			`transaction."id"`,
			`COALESCE(transaction."transactionId", '00000000-0000-0000-0000-000000000000') as "transactionId"`,
			`COALESCE(transaction."refNumber", '') as "refNumber"`,
			`COALESCE(transaction."orderType", '') as "orderType"`,
			`COALESCE(transaction."method", '') as "method"`,
			`COALESCE(transaction."typeTransaction", '') as "typeTransaction"`,
			`COALESCE(transaction."transactionDate", '1970-01-01'::date) as "transactionDate"`,
			`COALESCE(transaction."transactionTime", '1970-01-01 00:00:00+00'::timestamptz) as "transactionTime"`,
			`COALESCE(transaction."fromAccount", '') as "fromAccount"`,
			`'' as "fromAccountProductTypeName"`,
			`'' as "fromAccountName"`,
			`COALESCE(transaction."toAccount", '') as "toAccount"`,
			`'' as "toAccountProductTypeName"`,
			`'' as "toAccountName"`,
			`transaction."amount"`,
			`COALESCE(transaction."status", '') as "status"`,
			`COALESCE(transaction."description", '') as "description"`,
			`COALESCE(transaction."metadata", '{}'::jsonb) as "metadata"`,
			`COALESCE(transaction."createdAt", '1970-01-01 00:00:00+00'::timestamptz) as "createdAt"`,
			`COALESCE(transaction."updatedAt", '1970-01-01 00:00:00+00'::timestamptz) as "updatedAt"`,
			`COALESCE(transaction."currency", 'IDR') as "currency"`,
		}
	}

	query := buildFilteredTransactionQuery(columns, opts)

	if opts.Cursor != nil {
		if opts.Cursor.IsBackward {
			query = query.Where(
				`(transaction."transactionDate", transaction."id") > (?, ?)`,
				opts.Cursor.TransactionDate,
				opts.Cursor.DatabaseID)
		} else {
			query = query.Where(
				`(transaction."transactionDate", transaction."id") < (?, ?)`,
				opts.Cursor.TransactionDate,
				opts.Cursor.DatabaseID)
		}
	}

	if opts.Cursor != nil && opts.Cursor.IsBackward {
		query = query.OrderBy(`transaction."transactionDate" ASC, transaction."id" ASC`)
	} else {
		query = query.OrderBy(`transaction."transactionDate" DESC, transaction."id" DESC`)
	}

	if opts.Limit > 0 {
		query = query.Limit(uint64(opts.Limit))
	}

	return query.ToSql()
}

func buildFilteredTransactionQuery(cols []string, opts models.TransactionFilterOptions) sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	query := psql.Select(cols...).
		From("transaction").
		LeftJoin(`account af ON transaction."fromAccount" = af."accountNumber"`).
		LeftJoin(`account at ON transaction."toAccount" = at."accountNumber"`)

	if opts.OnlyAMF {
		query = query.Where(sq.And{
			sq.Eq{`af."entityCode"`: "001"},
			sq.Eq{`at."entityCode"`: "001"},
		})
	}

	if opts.SearchBy != "" && opts.Search != "" {
		switch opts.SearchBy {
		case "accountNumber":
			query = query.Where(sq.Or{
				sq.Eq{`transaction."fromAccount"`: opts.Search},
				sq.Eq{`transaction."toAccount"`: opts.Search},
			})
		case "transactionId", "refNumber":
			query = query.Where(sq.Eq{
				fmt.Sprintf(`transaction."%s"`, opts.SearchBy): opts.Search,
			})
		}
	}

	if opts.OrderType != "" {
		query = query.Where(sq.Eq{`transaction."orderType"`: opts.OrderType})
	}

	if len(opts.TransactionTypes) > 0 {
		query = query.Where(sq.Eq{`transaction."typeTransaction"`: opts.TransactionTypes})
	}

	if opts.ProductTypeName != "" {
		query = query.Where(
			sq.Or{
				sq.Eq{`af."productTypeName"`: opts.ProductTypeName},
				sq.Eq{`at."productTypeName"`: opts.ProductTypeName},
			},
		)
	}

	if opts.StartDate == nil && opts.EndDate == nil {
		now, _ := common.NowZeroTime()
		query = query.Where(sq.GtOrEq{`transaction."transactionDate"`: now.AddDate(0, 0, -7)})
		query = query.Where(sq.LtOrEq{`transaction."transactionDate"`: now})
	}

	if opts.StartDate != nil {
		query = query.Where(sq.GtOrEq{`transaction."transactionDate"`: opts.StartDate})
	}

	if opts.EndDate != nil {
		query = query.Where(sq.LtOrEq{`transaction."transactionDate"`: opts.EndDate})
	}

	if opts.TransactionDate != nil {
		td := common.FormatDatetimeToString(*opts.TransactionDate, common.DateFormatYYYYMMDD)
		query = query.Where(sq.Eq{`transaction."transactionDate"`: td})
	}

	return query
}

func buildStatusCountTransactionQuery(threshold uint, opts models.TransactionFilterOptions) (sql string, args []interface{}, err error) {
	subQuery := buildFilteredTransactionQuery([]string{"1"}, opts)
	subQuery = subQuery.Limit(uint64(threshold + 1))

	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)
	query := psql.Select().
		Column(`count(1) > ? as exceed_threshold`, threshold).
		FromSelect(subQuery, "sub")

	return query.ToSql()
}

func buildCountTransactionQuery(opts models.TransactionFilterOptions) (sql string, args []interface{}, err error) {
	columns := []string{
		`count(1)`,
	}
	query := buildFilteredTransactionQuery(columns, opts)

	return query.ToSql()
}
