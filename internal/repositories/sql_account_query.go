package repositories

import (
	"fmt"
	"sort"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	sq "github.com/Masterminds/squirrel"
)

// query to account database
var (
	queryAccountGetAllWithoutPagination = `
	SELECT 
		"accountNumber",
		"ownerId",
		"actualBalance",
		"pendingBalance"
	FROM "account";`

	GetOneByAccountNumber = `select 
	a."id",
	a."accountNumber",
	COALESCE(a."ownerId", '') as "ownerId",
	'' as "categoryName",
	'' as "subCategoryName",
	a."entityCode" as "entityCode",
	COALESCE(a."currency", '') as "currency",
	COALESCE(a."status", '') as "status",
	a."isHvt",
	a."actualBalance",
	a."pendingBalance",
	a."createdAt",
	a."updatedAt",
	COALESCE(a."legacyId", '{}') "legacyId",
	LOWER(f."preset") as "featurePreset",
	f."balance_range_min" as "featureBalanceRangeMin",
	f."balance_range_max" as "featureBalanceRangeMax",
	f."negative_balance_allowed" as "featureNegativeBalanceAllowed",
	f."negative_balance_limit" as "featureNegativeBalanceLimit",
	COALESCE(a."name", '') as "accountName"
		FROM "account" a
		LEFT JOIN feature f ON f."account_number" = a."accountNumber"
	WHERE
		a."accountNumber" = $1;`

	queryAccountCreate = `
		INSERT INTO account(
			"accountNumber", "name", "ownerId", "productTypeName", "categoryCode", "subCategoryCode", "entityCode", "currency", "altId", 
		    "legacyId", "isHvt", "status", "metadata", "createdAt", "updatedAt"
		)
		VALUES(
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, now(), now()
		);
	`

	queryAccountUpsert = `
		INSERT INTO account(
			"accountNumber", "name", "ownerId", "productTypeName", "categoryCode", "subCategoryCode", "entityCode", "currency", "altId", 
			"legacyId", "isHvt", "status", "metadata", "createdAt", "updatedAt"
		)
		VALUES(
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, now(), now()
		) ON CONFLICT ("accountNumber") DO UPDATE SET 
			"name" = EXCLUDED."name", "ownerId" = EXCLUDED."ownerId", "productTypeName" = EXCLUDED."productTypeName", 
			"categoryCode" = EXCLUDED."categoryCode", "subCategoryCode" = EXCLUDED."subCategoryCode", 
			"entityCode" = EXCLUDED."entityCode", "currency" = EXCLUDED."currency", "altId" = EXCLUDED."altId", 
			"legacyId" = EXCLUDED."legacyId", "isHvt" = EXCLUDED."isHvt", "status" = EXCLUDED."status", 
			"metadata" = EXCLUDED."metadata", "updatedAt" = now();
`

	QueryAccountCheckDataById = `SELECT "id" FROM "account" WHERE "id" = $1`

	// get list of account by account numbers
	queryCheckByAccountNumbers = `
		SELECT 
			"accountNumber"
		FROM "account"
		WHERE
			"accountNumber" = ANY($1)`

	queryGetOneByLegacyId = `
		SELECT 
			"id",
			"accountNumber",
			"actualBalance",
			"pendingBalance"
		FROM "account"
		WHERE "legacyId"->>'t24AccountNumber' = $1;`

	queryGetAccountBalance = `
	SELECT 
		"accountNumber",
		"actualBalance",
		"pendingBalance",
		COALESCE("name", '') as "name",
		COALESCE("productTypeName", '') as "productTypeName",
		COALESCE("subCategoryCode", '') as "subCategoryCode"
	FROM "account"
	WHERE "accountNumber" = ANY($1)`

	queryGetAccountVersion = `
	SELECT 
		"version"
	FROM "account"
	WHERE "accountNumber" = $1 LIMIT 1`

	queryUpdateAccountBalance = `
	UPDATE "account"
	SET
		"actualBalance" = $1,
		"pendingBalance" = $2,
		"version" = $3,
		"updatedAt" = $4
	WHERE "accountNumber" = $5 AND "version" = $6`

	queryUpdate = `
	UPDATE "account"
	SET
		"isHvt" = ?,
	`

	queryEstimateCountAccount = `
		SELECT reltuples::bigint AS estimate
		FROM pg_class
		WHERE relname = 'account'
		order by reltuples desc
		limit 1
	`

	queryAccountDelete = "DELETE FROM account WHERE id = $1"

	queryDeleteAccountByAccountNumber = `DELETE FROM account WHERE "accountNumber" = $1`

	queryUpdateBySubCategory = `
	UPDATE "account"
	SET
`

	queryUpdateBySubCategoryWhere = `
		"updatedAt" = now()
	WHERE "subCategoryCode" = ?;
	`

	newQueryGetOneByAccountNumber = `
	SELECT a."id",
       a."accountNumber",
       COALESCE(a."ownerId", '')  as "ownerId",
       ''                         as "categoryName",
       ''                         as "subCategoryName",
       a."entityCode"             as "entityName",
       COALESCE(a."currency", '') as "currency",
       COALESCE(a."status", '')   as "status",
       a."isHvt",
       a."actualBalance",
       a."pendingBalance",
       a."createdAt",
       a."updatedAt",
       COALESCE(a."legacyId", '{}')  "legacyId",
       COALESCE(a."name", '')     as "accountName",
       LOWER(f."preset")            as "featurePreset",
       f."balance_range_min"        as "featureBalanceRangeMin",
       f."balance_range_max"        as "featureBalanceRangeMax",
       f."negative_balance_allowed" as "featureNegativeBalanceAllowed",
       f."negative_balance_limit"   as "featureNegativeBalanceLimit"
	FROM "account" a
        LEFT JOIN feature f ON f."account_number" = a."accountNumber"
	WHERE a."accountNumber" = $1
	LIMIT 1;`

	newQueryGetOneByAccountNumberLegacy = `
	SELECT a."id",
       a."accountNumber",
       COALESCE(a."ownerId", '')  as "ownerId",
       ''                         as "categoryName",
       ''                         as "subCategoryName",
       a."entityCode"             as "entityName",
       COALESCE(a."currency", '') as "currency",
       COALESCE(a."status", '')   as "status",
       a."isHvt",
       a."actualBalance",
       a."pendingBalance",
       a."createdAt",
       a."updatedAt",
       COALESCE(a."legacyId", '{}')  "legacyId",
       COALESCE(a."name", '')     as "accountName",
       LOWER(f."preset")            as "featurePreset",
       f."balance_range_min"        as "featureBalanceRangeMin",
       f."balance_range_max"        as "featureBalanceRangeMax",
       f."negative_balance_allowed" as "featureNegativeBalanceAllowed",
       f."negative_balance_limit"   as "featureNegativeBalanceLimit"
	FROM "account" a
         LEFT JOIN feature f ON f."account_number" = a."accountNumber"
	WHERE a."legacyId" ->> 't24AccountNumber' = $1
	LIMIT 1;`
)

func buildAccountsEntityQuery(accountNumbers []string) (string, []interface{}, error) {
	queryBuilder := sq.
		Select(
			`"accountNumber"`,
			`COALESCE("name", '') AS "name"`,
			`"ownerId"`,
			`COALESCE("productTypeName", '') AS "productTypeName"`,
			`"categoryCode"`,
			`COALESCE("subCategoryCode", '') AS "subCategoryCode"`,
			`"entityCode"`,
			`COALESCE("altId", '') AS "altId"`,
			`COALESCE("legacyId", '{}') AS "legacyId"`,
			`"isHvt"`,
			`"status"`,
			`"metadata"`,
		).
		From("account").
		Where(sq.Eq{`"accountNumber"`: accountNumbers}).
		PlaceholderFormat(sq.Dollar)

	sql, args, err := queryBuilder.ToSql()
	if err != nil {
		return "", nil, err
	}
	return sql, args, nil

}

func buildGetAccountBalancesQuery(req models.GetAccountBalanceRequest) (sql string, args []any, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	// prevent deadlock by sorting the account numbers
	sort.Strings(req.AccountNumbers)

	query := psql.
		Select(`"accountNumber"`, `"actualBalance"`, `"pendingBalance"`, `"version"`, `"updatedAt"`).
		From(`"account"`).
		Where(sq.Eq{`"accountNumber"`: req.AccountNumbers})

	if req.ExcludeHVT {
		query = query.Where(sq.Eq{`"isHvt"`: false})
	}

	if req.ForUpdate {
		query = query.Suffix("FOR UPDATE")
	}

	return query.ToSql()
}

func buildFilteredAccountQuery(cols []string, opts models.AccountFilterOptions) sq.SelectBuilder {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.Select(cols...).From("account")

	if opts.Search != "" {
		query = query.Where(sq.Eq{
			`account."accountNumber"`: opts.Search,
		})
	}

	if opts.AccountNumber != "" {
		query = query.Where(sq.Eq{
			`account."accountNumber"`: opts.AccountNumber,
		})
	}

	if opts.AccountName != "" {
		query = query.Where("LOWER(account.name) = LOWER(?)", opts.AccountName)
	}

	if opts.OwnerID != "" {
		query = query.Where(sq.Eq{
			`account."ownerId"`: opts.OwnerID,
		})
	}

	return query
}

func buildListAccountQuery(opts models.AccountFilterOptions) (sql string, args []interface{}, err error) {
	columns := []string{
		`account."id"`,
		`account."accountNumber"`,
		`COALESCE(account."ownerId", '') as "ownerId"`,
		`'' as "categoryName"`,
		`'' as "subCategoryName"`,
		`'' as "entityName"`,
		`COALESCE(account."currency", '') as "currency"`,
		`account."actualBalance"`,
		`account."pendingBalance"`,
		`COALESCE(account."status", '') as "status"`,
		`account."createdAt"`,
		`account."updatedAt"`,
		`COALESCE(account."name", '') as "accountName"`,
	}

	query := buildFilteredAccountQuery(columns, opts)

	// sort column
	// key(input from user), val(column db)
	sortColumnMap := map[string]string{
		"createdAt":       `account."id"`,
		"updatedAt":       `account."updatedAt"`,
		"lastUpdatedDate": `account."updatedAt"`, // for backward compatibility
	}
	sortColumn := sortColumnMap[opts.SortBy]
	if sortColumn == "" {
		sortColumn = `account."id"`
	}

	// Sort direction
	sortDirection := opts.Sort
	if sortDirection != "desc" && sortDirection != "asc" {
		sortDirection = "desc"
	}

	// Offset
	if opts.Cursor != nil {
		val, ok := map[string]any{
			`account."id"`:        opts.Cursor.Id,
			`account."updatedAt"`: opts.Cursor.UpdatedAt,
		}[sortColumn]
		if !ok {
			return "", nil, fmt.Errorf("invalid sort column: %s", sortColumn)
		}

		operator := "<"
		if (sortDirection == "asc" && !opts.Cursor.IsBackward) ||
			(sortDirection == "desc" && opts.Cursor.IsBackward) {
			operator = ">"
		}

		query = query.Where(fmt.Sprintf(`%s %s ?`, sortColumn, operator), val)
	}

	// Order
	if opts.Cursor != nil && opts.Cursor.IsBackward {
		query = query.OrderBy(fmt.Sprintf("%s %s", sortColumn, opts.GetReversedSortDirection()))
	} else {
		query = query.OrderBy(fmt.Sprintf("%s %s", sortColumn, sortDirection))
	}

	query = query.Limit(uint64(opts.Limit))

	return query.ToSql()
}

func buildTotalBalanceAccountQuery(opts models.AccountFilterOptions) (sql string, args []interface{}, err error) {
	columns := []string{
		`SUM(account."actualBalance") - SUM(account."pendingBalance") as "totalBalance"`,
	}

	query := buildFilteredAccountQuery(columns, opts)

	return query.ToSql()
}

func buildCountAccountQuery(opts models.AccountFilterOptions) (sql string, args []interface{}, err error) {
	columns := []string{
		`count(1)`,
	}

	query := buildFilteredAccountQuery(columns, opts)

	return query.ToSql()
}

func buildUpdateBySubCategoryQuery(opts models.UpdateAccountBySubCategoryIn) (sql string, args []interface{}, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	query := psql.Update("account")
	if opts.ProductTypeName != nil {
		query = query.Set(`"productTypeName"`, *opts.ProductTypeName)
	}
	if opts.Currency != nil {
		query = query.Set(`"currency"`, *opts.Currency)
	}
	query = query.Set(`"updatedAt"`, "now()").Where(sq.Eq{
		`"subCategoryCode"`: opts.Code,
	})

	return query.ToSql()
}
