package repositories

var (
	queryListByDate = `select
		"accountNumber", "date", "balance"
	from
		account_balance_daily
	where
		"date" = $1;`

	queryABDGetLast = `select "accountNumber", "date", "balance"
	from account_balance_daily abd 
	order by "date" desc limit 1`
)
