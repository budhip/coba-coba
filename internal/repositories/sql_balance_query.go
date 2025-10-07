package repositories

import (
	"regexp"

	sq "github.com/Masterminds/squirrel"
	"golang.org/x/exp/slices"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

var (
	queryGetAccountBalanceWithFeature = `
		WITH account_balances as (
			SELECT
				account."accountNumber",
				COALESCE(account."legacyId"->>'t24AccountNumber', '') as "t24AccountNumber",
				account."actualBalance",
				account."pendingBalance",
				account."isHvt",
				account."version",
				account."updatedAt"
			FROM account
			WHERE "legacyId"->>'t24AccountNumber' = $1
			UNION ALL
			SELECT
				account."accountNumber",
				COALESCE(account."legacyId"->>'t24AccountNumber', '') as "t24AccountNumber",
				account."actualBalance",
				account."pendingBalance",
				account."isHvt",
				account."version",
				account."updatedAt"
			FROM account
			WHERE account."accountNumber" = $1
		)
		SELECT
			account_balances."accountNumber",
			account_balances."t24AccountNumber",
			account_balances."actualBalance",
			account_balances."pendingBalance",
			account_balances."isHvt",
			account_balances."version",
			account_balances."updatedAt",
			LOWER(feature."preset"),
			feature."negative_balance_allowed",
			feature."balance_range_min",
			feature."negative_balance_limit",
			feature."balance_range_max"
		FROM account_balances
		LEFT JOIN feature ON feature."account_number" = account_balances."accountNumber"
		LIMIT 1;`
	queryAdjustAccountBalance = `
	UPDATE account
	SET
		"actualBalance" = account."actualBalance" + $1,
		"updatedAt" = now()
	WHERE "accountNumber" = $2`
)

func isAccountNumbersPASFormat(accountNumbers []string) bool {
	return All(accountNumbers, func(s string) bool {
		isNumeric, _ := regexp.MatchString(`^\d+$`, s)

		return isNumeric && len(s) > 14
	})
}

func buildGetManyAccountBalanceQuery(req models.GetAccountBalanceRequest, ignoredAccounts []string) (sql string, args []interface{}, err error) {
	psql := sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

	var accountNumbers []string
	for _, accountNumber := range req.AccountNumbers {
		accountNotExcluded := !slices.Contains(req.AccountNumbersExcludedFromDB, accountNumber) &&
			!slices.Contains(ignoredAccounts, accountNumber)

		if accountNotExcluded {
			accountNumbers = append(accountNumbers, accountNumber)
		}
	}

	finalCols := []string{
		`account."accountNumber"`,
		`COALESCE(account."legacyId"->>'t24AccountNumber', '')`,
		`account."actualBalance"`,
		`account."pendingBalance"`,
		`account."isHvt"`,
		`account."version"`,
		`account."updatedAt"`,
		`LOWER(feature."preset")`,
		`feature."negative_balance_allowed"`,
		`feature."balance_range_min"`,
		`feature."negative_balance_limit"`,
		`feature."balance_range_max"`,
	}

	//	handle query using PAS format
	if isAccountNumbersPASFormat(req.AccountNumbers) {
		query := psql.
			Select(finalCols...).
			From("account").
			LeftJoin(`feature ON feature."account_number" = account."accountNumber"`).
			Where(sq.Eq{`account."accountNumber"`: accountNumbers})

		if req.ExcludeHVT {
			query = query.Where(sq.Eq{`account."isHvt"`: false})
		}

		if req.ForUpdate {
			// lock row for account table only
			query = query.
				OrderBy(`account."accountNumber"`).
				Suffix("FOR UPDATE OF account")
		}

		return query.ToSql()
	}

	// handle query using legacy format
	queryAccount := psql.Select(`"accountNumber"`).From("account")
	qa1 := queryAccount.Where(sq.Eq{`account."accountNumber"`: accountNumbers})
	qa2 := queryAccount.Where(sq.Eq{`account."legacyId"->>'t24AccountNumber'`: accountNumbers})

	if req.ExcludeHVT {
		qa1 = qa1.Where(sq.Eq{`account."isHvt"`: false})
		qa2 = qa2.Where(sq.Eq{`account."isHvt"`: false})
	}

	unionQuery := WithUnion(qa1, qa2)

	combinedQ := psql.Select(finalCols...).
		From("cte_1").
		Join(`account using ("accountNumber")`).
		LeftJoin(`feature ON feature."account_number" = cte_1."accountNumber"`).
		OrderBy(`account."accountNumber"`)

	if req.ForUpdate {
		// lock row for account table only
		combinedQ = combinedQ.
			OrderBy(`account."accountNumber"`).
			Suffix("FOR UPDATE OF account")
	}

	return WithCTE(unionQuery).
		Do(combinedQ).
		ToSql()
}
