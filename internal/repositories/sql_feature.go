package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"
	"github.com/shopspring/decimal"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type FeatureRepository interface {
	Register(context.Context, *models.CreateWalletIn) (models.WalletOut, error)
	Update(context.Context, *models.UpdateWalletIn) (out models.WalletOut, err error)
	GetFeatureByAccountNumbers(ctx context.Context, accountNumbers []string) (out models.MapAccountFeature, err error)

	//Update(ctx context.Context, accountNumber string, param models.UpdateWalletIn) (out models.UpdateWalletOut, err error)
}
type featureRepository sqlRepo

var _ FeatureRepository = (*featureRepository)(nil)

func (fr *featureRepository) Register(ctx context.Context, in *models.CreateWalletIn) (out models.WalletOut, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	var (
		args []interface{}
		now  = time.Now()
	)

	// unused now
	//db := wr.r.extractTx(ctx)

	args = append(args, in.AccountNumber)
	args = append(args, in.Feature.Preset)
	args = append(args, in.Feature.BalanceRangeMin)
	args = append(args, in.Feature.BalanceRangeMax)
	args = append(args, in.Feature.AllowedNegativeBalance)
	args = append(args, in.Feature.NegativeBalanceLimit)
	args = append(args, now)

	result := models.WalletFeature{}

	err = fr.r.dbWrite.QueryRowContext(ctx, createFeatureQuery, args...).Scan(
		&out.AccountNumber,
		&result.Preset,
		&result.BalanceRangeMin,
		&result.BalanceRangeMax,
		&result.AllowedNegativeBalance,
		&result.NegativeBalanceLimit,
	)
	if err != nil {
		//err = fmt.Errorf("%w: %w", common.ErrUnableToCreate, err)
		return
	}
	out.Feature = &result

	return out, err
}

func (fr *featureRepository) Update(ctx context.Context, in *models.UpdateWalletIn) (out models.WalletOut, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	var (
		values []interface{}
		now    = time.Now()
		query  = updateFeatureQuery
	)

	db := fr.r.extractTxWrite(ctx)

	if in.Feature.Preset != nil {
		query += ` preset = ?,`
		values = append(values, in.Feature.Preset)
	}
	if in.Feature.BalanceRangeMin != nil {
		query += ` balance_range_min = ?,`
		values = append(values, in.Feature.BalanceRangeMin)
	}
	if in.Feature.BalanceRangeMax != nil {
		query += ` balance_range_max = ?,`
		values = append(values, in.Feature.BalanceRangeMax)
	}
	if in.Feature.AllowedNegativeBalance != nil {
		query += ` negative_balance_allowed = ?,`
		values = append(values, in.Feature.AllowedNegativeBalance)
	}
	if in.Feature.NegativeBalanceLimit != nil {
		query += ` negative_balance_limit = ?,`
		values = append(values, in.Feature.NegativeBalanceLimit)
	}

	if len(values) == 0 {
		return out, nil
	}

	query += ` updated_on = ?`
	values = append(values, now)
	values = append(values, in.AccountNumber)

	query += " WHERE account_number = ? RETURNING account_number, preset, balance_range_min, balance_range_max, negative_balance_allowed, negative_balance_limit;"
	query = fr.r.SubstitutePlaceholder(query, 1)

	result := models.WalletFeature{}

	err = db.QueryRowContext(ctx, query, values...).Scan(
		&out.AccountNumber,
		&result.Preset,
		&result.BalanceRangeMin,
		&result.BalanceRangeMax,
		&result.AllowedNegativeBalance,
		&result.NegativeBalanceLimit,
	)
	if err != nil {
		return
	}
	upperPreset := strings.ToUpper(*result.Preset)
	result.Preset = &upperPreset
	out.Feature = &result

	return out, err
}

func (fr *featureRepository) GetFeatureByAccountNumbers(ctx context.Context, accountNumbers []string) (out models.MapAccountFeature, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	out = make(models.MapAccountFeature)
	db := fr.r.extractTxWrite(ctx)

	rows, err := db.QueryContext(ctx, queryGetFeatureByAccountNumber, pq.Array(accountNumbers))
	if err != nil {
		return nil, err
	}

	defer rows.Close()
	for rows.Next() {
		feature := struct {
			AccountNumber          string
			Preset                 sql.NullString
			AllowedNegativeBalance sql.NullBool
			BalanceRangeMin        decimal.NullDecimal
			NegativeBalanceLimit   decimal.NullDecimal
		}{}

		err = rows.Scan(
			&feature.AccountNumber,
			&feature.Preset,
			&feature.BalanceRangeMin,
			&feature.AllowedNegativeBalance,
			&feature.NegativeBalanceLimit,
		)
		if err != nil {
			return nil, err
		}

		defaultFeature, ok := fr.r.config.AccountFeatureConfig[feature.Preset.String]
		if !ok {
			return nil, fmt.Errorf("preset %s not found", feature.Preset.String)
		}

		allowedNegativeBalance := defaultFeature.NegativeBalanceAllowed
		balanceRangeMin := decimal.NewFromFloat(defaultFeature.BalanceRangeMin)
		negativeBalanceLimit := decimal.NewFromFloat(defaultFeature.NegativeLimit)

		if feature.AllowedNegativeBalance.Valid {
			allowedNegativeBalance = feature.AllowedNegativeBalance.Bool
		}

		if feature.BalanceRangeMin.Valid {
			balanceRangeMin = feature.BalanceRangeMin.Decimal
		}

		if feature.NegativeBalanceLimit.Valid {
			negativeBalanceLimit = feature.NegativeBalanceLimit.Decimal
		}

		out[feature.AccountNumber] = models.WalletFeature{
			Preset:                 &feature.Preset.String,
			AllowedNegativeBalance: &allowedNegativeBalance,
			BalanceRangeMin:        &balanceRangeMin,
			NegativeBalanceLimit:   &negativeBalanceLimit,
		}
	}

	// handle for account number that not exists on db
	preset := models.DefaultPresetWalletFeature
	for _, accountNumber := range accountNumbers {
		defaultFeature, ok := fr.r.config.AccountFeatureConfig[preset]
		if !ok {
			return nil, fmt.Errorf("preset %s not found", preset)
		}

		_, existsOnDB := out[accountNumber]
		if !existsOnDB {
			balanceRangeMin := decimal.NewFromFloat(defaultFeature.BalanceRangeMin)
			negativeLimit := decimal.NewFromFloat(defaultFeature.NegativeLimit)

			out[accountNumber] = models.WalletFeature{
				Preset:                 &preset,
				AllowedNegativeBalance: &defaultFeature.NegativeBalanceAllowed,
				BalanceRangeMin:        &balanceRangeMin,
				NegativeBalanceLimit:   &negativeLimit,
			}
		}
	}

	return out, nil
}
