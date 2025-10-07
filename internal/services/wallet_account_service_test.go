package services_test

import (
	"context"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_WalletAccountService_CreateAccountFeature(t *testing.T) {
	testHelper := serviceTestHelper(t)
	validPreset := "customer"
	invalidPreset := "ConSumER"

	tests := []struct {
		name    string
		args    models.CreateWalletIn
		doMock  func(args models.CreateWalletIn)
		wantErr bool
	}{
		{
			name: "success",
			args: models.CreateWalletIn{
				AccountNumber: "40000075177",
				Feature: &models.WalletFeature{
					Preset: &validPreset,
				},
			},
			doMock: func(args models.CreateWalletIn) {
				testHelper.mockSQLRepository.EXPECT().GetFeatureRepository().Return(testHelper.mockFeatureRepository)

				testHelper.mockFeatureRepository.EXPECT().Register(gomock.Any(), &args).Return(models.WalletOut{}, nil)
			},
			wantErr: false,
		},
		{
			name: "invalid preset",
			args: models.CreateWalletIn{
				AccountNumber: "40000075177",
				Feature: &models.WalletFeature{
					Preset: &invalidPreset,
				},
			},
			doMock: func(args models.CreateWalletIn) {
			},
			wantErr: true,
		},
		{
			name: "repo fail",
			args: models.CreateWalletIn{
				AccountNumber: "40000075177",
				Feature: &models.WalletFeature{
					Preset: &validPreset,
				},
			},
			doMock: func(args models.CreateWalletIn) {
				testHelper.mockSQLRepository.EXPECT().GetFeatureRepository().Return(testHelper.mockFeatureRepository)

				testHelper.mockFeatureRepository.EXPECT().Register(gomock.Any(), &args).Return(models.WalletOut{}, assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			_, err := testHelper.walletAccountService.CreateAccountFeature(context.Background(), tt.args)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
