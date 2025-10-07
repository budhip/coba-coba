package repositories

import (
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/flag"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/shopspring/decimal"
	"golang.org/x/exp/slices"
)

func createBalanceOptions(abf models.AccountBalanceFeature, ignoredAccounts []string, flag flag.Client, config config.Config) ([]models.BalanceOption, error) {
	var opts []models.BalanceOption

	if slices.Contains(ignoredAccounts, abf.AccountNumber) {
		opts = append(opts, models.WithIgnoreBalanceSufficiency())
	}

	preset := models.DefaultPresetWalletFeature
	if abf.Preset.Valid {
		preset = abf.Preset.String
	}

	defaultFeature, ok := config.AccountFeatureConfig[preset]
	if !ok {
		return nil, fmt.Errorf("preset not found: %s", abf.Preset.String)
	}

	if len(defaultFeature.AllowedNegativeTrxType) > 0 {
		opts = append(opts, models.WithAllowedNegativeBalanceTransactionTypes(defaultFeature.AllowedNegativeTrxType))
	}

	if abf.AllowedNegativeBalance.Valid {
		if abf.AllowedNegativeBalance.Bool && abf.NegativeBalanceLimit.Valid {
			opts = append(opts, models.WithNegativeBalanceLimit(abf.NegativeBalanceLimit.Decimal))
		}
	} else {
		if defaultFeature.NegativeBalanceAllowed {
			negativeLimit := decimal.NewFromFloat(defaultFeature.NegativeLimit)
			opts = append(opts, models.WithNegativeBalanceLimit(negativeLimit))
		}
	}

	if abf.IsHVT.Valid && abf.IsHVT.Bool {
		opts = append(opts, models.WithHVT())
	}

	if abf.Version.Valid {
		opts = append(opts, models.WithVersion(int(abf.Version.Int64)))
	}

	if !abf.BalanceRangeMax.Valid || abf.BalanceRangeMax.Decimal.LessThanOrEqual(decimal.Zero) {
		opts = append(opts, models.WithBalanceRangeMax(decimal.NewFromFloat(defaultFeature.BalanceRangeMax)))
	} else {
		opts = append(opts, models.WithBalanceRangeMax(abf.BalanceRangeMax.Decimal))
	}

	opts = append(opts, models.WithLastUpdatedAt(abf.LastUpdatedAt))
	opts = append(opts, models.WithBalanceLimitEnabled(flag.IsEnabled(config.FeatureFlagKeyLookup.BalanceLimitToggle)))

	return opts, nil
}

func WithUnion(firstQuery sq.SelectBuilder, secondQuery sq.SelectBuilder) sq.SelectBuilder {
	return firstQuery.SuffixExpr(secondQuery.Prefix("UNION"))
}

type CTEBuilder struct {
	ctes []sq.SelectBuilder
	q    sq.SelectBuilder
}

func WithCTE(b ...sq.SelectBuilder) CTEBuilder {
	return CTEBuilder{
		ctes: b,
		q:    sq.SelectBuilder{},
	}
}

func (c CTEBuilder) Do(q sq.SelectBuilder) CTEBuilder {
	c.q = q

	return c
}

func (c CTEBuilder) ToSql() (string, []any, error) {
	var combinedArgs []any
	var combinedSql string

	for i := 0; i < len(c.ctes); i++ {
		sql, args, err := c.ctes[i].ToSql()
		if err != nil {
			return "", nil, err
		}
		combinedArgs = append(combinedArgs, args...)

		if i == 0 {
			combinedSql += fmt.Sprintf("with cte_%v as (%s)", i+1, sql)
		} else {
			combinedSql += fmt.Sprintf(", cte_%v as (%s) ", i+1, sql)
		}
	}

	sql, args, err := c.q.ToSql()
	if err != nil {
		return "", nil, err
	}

	combinedArgs = append(combinedArgs, args...)
	combinedSql += sql

	return combinedSql, combinedArgs, nil
}

func All[T any](ts []T, pred func(T) bool) bool {
	for _, t := range ts {
		if !pred(t) {
			return false
		}
	}
	return true
}

func getMapFromConfig(configMapAccount map[string]map[string]string, key string) map[string]string {
	for k, v := range configMapAccount {
		if strings.EqualFold(k, key) {
			return v
		}
	}

	return map[string]string{}
}

// getAccountNumberFromConfig returns case-insensitive account number based on the key
// we use case-insensitive comparison for this because there still open issue on viper 3rd party library (go-config-loader)
// [link issue](https://github.com/spf13/viper/issues/1014), and this will not be fixed [link](https://github.com/spf13/viper?tab=readme-ov-file#does-viper-support-case-sensitive-keys)
func getAccountNumberFromConfig(configMapAccount map[string]string, key string) (string, error) {
	for k, v := range configMapAccount {
		if strings.EqualFold(k, key) {
			if v == "" {
				return "", fmt.Errorf("%w: account number for %s is empty", common.ErrConfigAccountNumberNotFound, key)
			}

			return v, nil
		}
	}

	return "", fmt.Errorf("%w: no account found for %s", common.ErrConfigAccountNumberNotFound, key)
}
