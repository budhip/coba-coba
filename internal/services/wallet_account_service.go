package services

import (
	"context"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
)

type WalletAccountService interface {
	CreateAccountFeature(context.Context, models.CreateWalletIn) (*models.WalletOut, error)
}

type walletAccount service

var _ WalletAccountService = (*walletAccount)(nil)

func (wa *walletAccount) CreateAccountFeature(ctx context.Context, payload models.CreateWalletIn) (out *models.WalletOut, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	// not found in config and eligible list
	if _, ok := wa.srv.conf.AccountFeatureConfig[*payload.Feature.Preset]; !ok {
		err = common.ErrInvalidPreset
		return
	}

	resp, err := wa.srv.sqlRepo.GetFeatureRepository().Register(ctx, &payload)
	if err != nil {
		return out, err
	}

	out = &resp
	return out, nil
}
