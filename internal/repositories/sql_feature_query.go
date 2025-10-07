package repositories

const (
	createFeatureQuery = `INSERT INTO "feature" AS t (
		account_number,
		preset,
		balance_range_min,
		balance_range_max,
		negative_balance_allowed,
		negative_balance_limit,
		created_on,
		updated_on
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $7
	) ON CONFLICT (account_number) DO UPDATE
	SET
		preset = EXCLUDED.preset,
		balance_range_min = COALESCE(EXCLUDED.balance_range_min, t.balance_range_min),
		balance_range_max = COALESCE(EXCLUDED.balance_range_max, t.balance_range_max),
		negative_balance_allowed = COALESCE(EXCLUDED.negative_balance_allowed, t.negative_balance_allowed),
		negative_balance_limit = COALESCE(EXCLUDED.negative_balance_limit, t.negative_balance_limit),
		updated_on = EXCLUDED.updated_on
	RETURNING
		account_number, preset, balance_range_min, balance_range_max, negative_balance_allowed, negative_balance_limit;`

	updateFeatureQuery             = `UPDATE "feature" SET `
	queryGetFeatureByAccountNumber = `
		SELECT account_number, LOWER(preset), balance_range_min, negative_balance_allowed, negative_balance_limit
		FROM "feature"
		WHERE account_number = ANY($1);`
)
