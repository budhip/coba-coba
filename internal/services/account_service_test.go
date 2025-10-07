package services_test

import (
	"context"
	"database/sql"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories/mock"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestAccountService_GetList(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		opts models.AccountFilterOptions
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		{
			name: "test success",
			args: args{
				opts: models.AccountFilterOptions{Search: "123456"},
			},
			doMock: func(args args) {
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository)
				testHelper.mockAccRepository.EXPECT().GetList(gomock.AssignableToTypeOf(context.Background()), args.opts).Return([]models.GetAccountOut{{AccountNumber: "231231313"}}, nil)
				testHelper.mockAccRepository.EXPECT().CountAll(gomock.AssignableToTypeOf(context.Background()), args.opts).Return(1, nil)
			},
			wantErr: false,
		},
		{
			name: "test error GetList",
			args: args{
				opts: models.AccountFilterOptions{
					Search: "123456",
				},
			},
			doMock: func(args args) {
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository)
				testHelper.mockAccRepository.EXPECT().GetList(gomock.AssignableToTypeOf(context.Background()), args.opts).Return([]models.GetAccountOut{{AccountNumber: "231231313"}}, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "test error CountAll",
			args: args{
				opts: models.AccountFilterOptions{Search: "123456"},
			},
			doMock: func(args args) {
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository)
				testHelper.mockAccRepository.EXPECT().GetList(gomock.AssignableToTypeOf(context.Background()), args.opts).Return([]models.GetAccountOut{{AccountNumber: "231231313"}}, nil)
				testHelper.mockAccRepository.EXPECT().CountAll(gomock.AssignableToTypeOf(context.Background()), args.opts).Return(1, assert.AnError)
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

			_, _, err := testHelper.accountService.GetList(context.Background(), tt.args.opts)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestAccountService_GetOneByAccountNumber(t *testing.T) {
	testHelper := serviceTestHelper(t)

	accountNumber := "[TEST]"
	result := models.GetAccountOut{
		AccountNumber: "",
	}

	type args struct {
		ctx           context.Context
		accountNumber string
	}
	type mockData struct {
	}
	tests := []struct {
		name     string
		args     args
		mockData mockData
		doMock   func(args args, mockData mockData)
		wantErr  bool
	}{
		{
			name: "success - get from pg",
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			mockData: mockData{},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(args.ctx, args.accountNumber).Return(result, nil)
			},
			wantErr: false,
		},
		{
			name: "fail - get from pg",
			args: args{
				ctx:           context.Background(),
				accountNumber: accountNumber,
			},
			mockData: mockData{},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(args.ctx, args.accountNumber).Return(result, common.ErrDataNotFound)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}
			_, err := testHelper.accountService.GetOneByAccountNumber(tt.args.ctx, tt.args.accountNumber)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestAccountService_Create(t *testing.T) {
	testHelper := serviceTestHelper(t)
	type args struct {
		ctx context.Context
		req models.CreateAccount
	}
	type mockData struct {
	}
	tests := []struct {
		name     string
		args     args
		mockData mockData
		doMock   func(args args, mockData mockData)
		wantErr  bool
	}{
		{
			name: "success create new account",
			args: args{
				ctx: context.Background(),
				req: models.CreateAccount{
					AccountNumber:   "22200100000001",
					Name:            "John Doe",
					OwnerID:         "12345",
					CategoryCode:    "222",
					SubCategoryCode: "100",
					EntityCode:      "001",
					Currency:        "IDR",
					Status:          common.AccountStatusActive,
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAccRepository.EXPECT().Create(args.ctx, args.req).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "fail create new account - Create - database error",
			args: args{
				ctx: context.Background(),
				req: models.CreateAccount{
					AccountNumber:   "22200100000001",
					Name:            "John Doe",
					OwnerID:         "12345",
					CategoryCode:    "222",
					SubCategoryCode: "100",
					EntityCode:      "001",
					Currency:        "IDR",
					Status:          common.AccountStatusActive,
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockAccRepository.EXPECT().Create(args.ctx, args.req).Return(common.ErrNoRowsAffected)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}

			_, err := testHelper.accountService.Create(tt.args.ctx, tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestAccountService_GetTotalBalance(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type Args struct {
		ctx  context.Context
		opts models.AccountFilterOptions
	}
	tests := []struct {
		name    string
		args    Args
		doMock  func(args Args)
		wantErr bool
	}{
		{
			name: "happy path",
			args: Args{
				ctx:  context.Background(),
				opts: models.AccountFilterOptions{},
			},
			doMock: func(args Args) {
				totalBalance := decimal.NewFromFloat(100)
				testHelper.mockAccRepository.EXPECT().GetTotalBalance(gomock.AssignableToTypeOf(args.ctx), args.opts).
					Return(&totalBalance, nil)
			},
			wantErr: false,
		},
		{
			name: "failed - error repo",
			args: Args{
				ctx:  context.Background(),
				opts: models.AccountFilterOptions{},
			},
			doMock: func(args Args) {
				testHelper.mockAccRepository.EXPECT().GetTotalBalance(gomock.AssignableToTypeOf(args.ctx), args.opts).
					Return(nil, assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.args)
			}
			_, err := testHelper.accountService.GetTotalBalance(tc.args.ctx, tc.args.opts)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func TestAccountService_Upsert(t *testing.T) {
	testHelper := serviceTestHelper(t)
	type args struct {
		ctx context.Context
		req models.AccountUpsert
	}
	type mockData struct {
	}
	tests := []struct {
		name     string
		args     args
		mockData mockData
		doMock   func(args args, mockData mockData)
		wantErr  bool
	}{
		{
			name: "success - update account",
			args: args{
				ctx: context.Background(),
				req: models.AccountUpsert{
					AccountNumber:   "1202517699",
					Name:            "Account Transaction 1",
					OwnerID:         "444444",
					CategoryCode:    "555555",
					SubCategoryCode: "666666",
					EntityCode:      "AMF",
					Currency:        "IDR",
					AltID:           "123321",
					LegacyId: &models.AccountLegacyId{
						"t24AccountNumber": "111000035909",
						"t24ArrangementId": "AA123",
					},
					Status: "ACTIVE",
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository)
				testHelper.mockAccRepository.EXPECT().Upsert(args.ctx, args.req).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - update account (without status)",
			args: args{
				ctx: context.Background(),
				req: models.AccountUpsert{
					AccountNumber:   "1202517699",
					Name:            "Account Transaction 1",
					OwnerID:         "444444",
					CategoryCode:    "555555",
					SubCategoryCode: "666666",
					EntityCode:      "AMF",
					Currency:        "IDR",
					AltID:           "123321",
					LegacyId: &models.AccountLegacyId{
						"t24AccountNumber": "111000035909",
						"t24ArrangementId": "AA123",
					},
				},
			},
			doMock: func(args args, mockData mockData) {
				args.req.Status = common.AccountStatusActive
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository)
				testHelper.mockAccRepository.EXPECT().Upsert(args.ctx, args.req).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "success - update account (with hvt)",
			args: args{
				ctx: context.Background(),
				req: models.AccountUpsert{
					AccountNumber:   "1202517699",
					Name:            "Account Transaction 1",
					OwnerID:         "444444",
					CategoryCode:    "555555",
					SubCategoryCode: "21103", // hvt
					EntityCode:      "AMF",
					Currency:        "IDR",
					AltID:           "123321",
					LegacyId: &models.AccountLegacyId{
						"t24AccountNumber": "111000035909",
						"t24ArrangementId": "AA123",
					},
				},
			},
			doMock: func(args args, mockData mockData) {
				args.req.Status = common.AccountStatusActive
				args.req.IsHVT = true
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository)
				testHelper.mockAccRepository.EXPECT().Upsert(args.ctx, args.req).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed - update error",
			args: args{
				ctx: context.Background(),
				req: models.AccountUpsert{
					AccountNumber:   "1202517699",
					Name:            "Account Transaction 1",
					OwnerID:         "444444",
					CategoryCode:    "555555",
					SubCategoryCode: "666666",
					EntityCode:      "AMF",
					Currency:        "IDR",
					AltID:           "123321",
					LegacyId: &models.AccountLegacyId{
						"t24AccountNumber": "111000035909",
						"t24ArrangementId": "AA123",
					},
					Status: "ACTIVE",
				},
			},
			mockData: mockData{},
			doMock: func(args args, mockData mockData) {
				testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository)
				testHelper.mockAccRepository.EXPECT().Upsert(args.ctx, args.req).Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}
			err := testHelper.accountService.Upsert(tt.args.ctx, tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestAccountService_GetACuanAccountNumber(t *testing.T) {
	testHelper := serviceTestHelper(t)
	tests := []struct {
		name    string
		param   string
		doMock  func(accountNumber string)
		wantErr bool
		wantRes string
	}{
		{
			name:  "happy path - found legacyId",
			param: "initial accountNumber",
			doMock: func(accountNumber string) {
				testHelper.mockAccRepository.EXPECT().GetOneByLegacyId(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(&models.Account{AccountNumber: "found legacyId"}, nil)
			},
			wantErr: false,
			wantRes: "found legacyId",
		},
		{
			name:  "happy path - found newAccountNumber",
			param: "initial accountNumber",
			doMock: func(accountNumber string) {
				testHelper.mockAccRepository.EXPECT().GetOneByLegacyId(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(nil, sql.ErrNoRows)

				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{AccountNumber: "found newAccountNumber"}, nil)
			},
			wantErr: false,
			wantRes: "found newAccountNumber",
		},
		{
			name:  "failed - GetOneByLegacyId err",
			param: "initial accountNumber",
			doMock: func(accountNumber string) {
				testHelper.mockAccRepository.EXPECT().GetOneByLegacyId(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(nil, assert.AnError)
			},
			wantErr: true,
			wantRes: "initial accountNumber",
		},
		{
			name:  "failed - GetOneByAccountNumber err",
			param: "initial accountNumber",
			doMock: func(accountNumber string) {
				testHelper.mockAccRepository.EXPECT().GetOneByLegacyId(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(nil, sql.ErrNoRows)

				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{}, assert.AnError)
			},
			wantErr: true,
			wantRes: "initial accountNumber",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock(tc.param)
			}

			actual, err := testHelper.accountService.GetACuanAccountNumber(context.Background(), tc.param)
			assert.Equal(t, tc.wantErr, err != nil)
			assert.Equal(t, tc.wantRes, actual)
		})
	}
}

func TestAccountService_Update(t *testing.T) {
	testHelper := serviceTestHelper(t)
	testHelper.mockSQLRepository.EXPECT().GetAccountRepository().Return(testHelper.mockAccRepository).AnyTimes()

	acc := models.GetAccountOut{
		ID:            99,
		AccountNumber: "123",
	}

	tests := []struct {
		name    string
		doMock  func(updatePayload models.UpdateAccountIn)
		wantErr bool
	}{
		{
			name: "happy path",
			doMock: func(updatePayload models.UpdateAccountIn) {
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)
						atomicFeatRepo := mock.NewMockFeatureRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo)
						atomicRepo.EXPECT().GetFeatureRepository().Return(atomicFeatRepo)

						atomicAccRepo.EXPECT().
							GetOneByAccountNumber(gomock.Any(), acc.AccountNumber).
							Return(acc, nil)

						atomicAccRepo.EXPECT().Update(
							gomock.AssignableToTypeOf(context.Background()),
							acc.ID,
							updatePayload,
						).Return(nil)

						atomicFeatRepo.EXPECT().Update(
							gomock.AssignableToTypeOf(context.Background()),
							&models.UpdateWalletIn{
								AccountNumber: updatePayload.AccountNumber,
								Feature:       updatePayload.Feature,
							},
						).Return(models.WalletOut{}, nil)

						return f(ctx, atomicRepo)
					})

			},
			wantErr: false,
		},
		{
			name: "failed - err get account",
			doMock: func(updatePayload models.UpdateAccountIn) {
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)
						atomicFeatRepo := mock.NewMockFeatureRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo)
						atomicRepo.EXPECT().GetFeatureRepository().Return(atomicFeatRepo)

						atomicAccRepo.EXPECT().
							GetOneByAccountNumber(gomock.Any(), acc.AccountNumber).
							Return(acc, assert.AnError)

						return f(ctx, atomicRepo)
					})
			},
			wantErr: true,
		},
		{
			name: "failed - err update",
			doMock: func(updatePayload models.UpdateAccountIn) {
				testHelper.mockSQLRepository.EXPECT().
					Atomic(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, f func(ctx context.Context, r repositories.SQLRepository) error) error {
						atomicRepo := mock.NewMockSQLRepository(testHelper.mockCtrl)
						atomicAccRepo := mock.NewMockAccountRepository(testHelper.mockCtrl)
						atomicFeatRepo := mock.NewMockFeatureRepository(testHelper.mockCtrl)

						atomicRepo.EXPECT().GetAccountRepository().Return(atomicAccRepo)
						atomicRepo.EXPECT().GetFeatureRepository().Return(atomicFeatRepo)

						atomicAccRepo.EXPECT().
							GetOneByAccountNumber(gomock.Any(), acc.AccountNumber).
							Return(acc, nil)

						atomicAccRepo.EXPECT().Update(
							gomock.AssignableToTypeOf(context.Background()),
							acc.ID,
							updatePayload,
						).Return(assert.AnError)

						return f(ctx, atomicRepo)
					})

			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			updatePayload := models.UpdateAccountIn{
				AccountNumber: acc.AccountNumber,
				IsHVT:         new(bool),
			}
			if tc.doMock != nil {
				tc.doMock(updatePayload)
			}
			_, err := testHelper.accountService.Update(context.Background(), updatePayload)
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func TestAccountService_RemoveDuplicateAccountMigration(t *testing.T) {
	testHelper := serviceTestHelper(t)
	tests := []struct {
		name    string
		doMock  func(accountNumber string)
		wantErr bool
	}{
		{
			name: "failed - GetOneByAccountNumber 1st err",
			doMock: func(accountNumber string) {
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{}, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "success - accountNumber not found",
			doMock: func(accountNumber string) {
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{}, sql.ErrNoRows)
			},
			wantErr: false,
		},
		{
			name: "success - accountNumber not registered",
			doMock: func(accountNumber string) {
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{ID: 0}, nil)
			},
			wantErr: false,
		},
		{
			name: "success - legacyId nil",
			doMock: func(accountNumber string) {
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{ID: 1, LegacyId: nil}, nil)
			},
			wantErr: false,
		},
		{
			name: "success - t24AccountNumber not found",
			doMock: func(accountNumber string) {
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{ID: 1, LegacyId: &models.AccountLegacyId{}}, nil)
			},
			wantErr: false,
		},
		{
			name: "success - t24AccountNumber empty",
			doMock: func(accountNumber string) {
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{ID: 1, LegacyId: &models.AccountLegacyId{"t24AccountNumber": ""}}, nil)
			},
			wantErr: false,
		},
		{
			name: "failed - GetOneByAccountNumber 2nd err",
			doMock: func(accountNumber string) {
				t24AccountNumber := "t24AccountNumber"
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{ID: 1, LegacyId: &models.AccountLegacyId{"t24AccountNumber": t24AccountNumber}}, nil).Times(1)
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					t24AccountNumber,
				).Return(models.GetAccountOut{}, assert.AnError).Times(1)
			},
			wantErr: true,
		},
		{
			name: "success - legacyID not found",
			doMock: func(accountNumber string) {
				t24AccountNumber := "t24AccountNumber"
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{ID: 1, LegacyId: &models.AccountLegacyId{"t24AccountNumber": t24AccountNumber}}, nil).Times(1)
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					t24AccountNumber,
				).Return(models.GetAccountOut{}, sql.ErrNoRows).Times(1)
			},
			wantErr: false,
		},
		{
			name: "success - legacyID not registered",
			doMock: func(accountNumber string) {
				t24AccountNumber := "t24AccountNumber"
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{ID: 1, LegacyId: &models.AccountLegacyId{"t24AccountNumber": t24AccountNumber}}, nil).Times(1)
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					t24AccountNumber,
				).Return(models.GetAccountOut{ID: 0}, nil).Times(1)
			},
			wantErr: false,
		},
		{
			name: "success - legacyID and accountNumber same ID",
			doMock: func(accountNumber string) {
				t24AccountNumber := "t24AccountNumber"
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{ID: 1, LegacyId: &models.AccountLegacyId{"t24AccountNumber": t24AccountNumber}}, nil).Times(1)
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					t24AccountNumber,
				).Return(models.GetAccountOut{ID: 1}, nil).Times(1)
			},
			wantErr: false,
		},
		{
			name: "failed - Delete err",
			doMock: func(accountNumber string) {
				t24AccountNumber := "t24AccountNumber"
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{ID: 1, LegacyId: &models.AccountLegacyId{"t24AccountNumber": t24AccountNumber}}, nil).Times(1)
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					t24AccountNumber,
				).Return(models.GetAccountOut{ID: 2}, nil).Times(1)
				testHelper.mockAccRepository.EXPECT().Delete(
					gomock.AssignableToTypeOf(context.Background()),
					2,
				).Return(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "success - delete legacy",
			doMock: func(accountNumber string) {
				t24AccountNumber := "t24AccountNumber"
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					accountNumber,
				).Return(models.GetAccountOut{ID: 1, LegacyId: &models.AccountLegacyId{"t24AccountNumber": t24AccountNumber}}, nil).Times(1)
				testHelper.mockAccRepository.EXPECT().GetOneByAccountNumber(
					gomock.AssignableToTypeOf(context.Background()),
					t24AccountNumber,
				).Return(models.GetAccountOut{ID: 2}, nil).Times(1)
				testHelper.mockAccRepository.EXPECT().Delete(
					gomock.AssignableToTypeOf(context.Background()),
					2,
				).Return(nil)
			},
			wantErr: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock("123")
			}

			err := testHelper.accountService.RemoveDuplicateAccountMigration(context.Background(), "123")
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}

func TestAccountService_UpdateBySubCategory(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type Args struct {
		ctx    context.Context
		params models.UpdateAccountBySubCategoryIn
	}
	tests := []struct {
		name    string
		args    Args
		doMock  func(args Args)
		wantErr bool
	}{
		{
			name: "success",
			args: Args{
				ctx: context.Background(),
				params: models.UpdateAccountBySubCategoryIn{
					Code:            "10000",
					ProductTypeName: &[]string{"test"}[0],
					Currency:        &[]string{"IDR"}[0],
				},
			},
			doMock: func(args Args) {
				testHelper.mockAccRepository.EXPECT().UpdateBySubCategory(args.ctx, args.params).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "error",
			args: Args{
				ctx: context.Background(),
				params: models.UpdateAccountBySubCategoryIn{
					Code:            "10000",
					ProductTypeName: &[]string{"test"}[0],
					Currency:        &[]string{"IDR"}[0],
				},
			},
			doMock: func(args Args) {
				testHelper.mockAccRepository.EXPECT().UpdateBySubCategory(args.ctx, args.params).
					Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}
			err := testHelper.accountService.UpdateBySubCategory(tt.args.ctx, tt.args.params)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestAccountService_Delete(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx           context.Context
		accountNumber string
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx:           context.Background(),
				accountNumber: "123456",
			},
			doMock: func(args args) {
				testHelper.mockAccRepository.EXPECT().
					DeleteByAccountNumber(args.ctx, args.accountNumber).
					Return(nil)
			},
			wantErr: false,
		},
		{
			name: "failed to delete account",
			args: args{
				ctx:           context.Background(),
				accountNumber: "123456",
			},
			doMock: func(args args) {
				testHelper.mockAccRepository.EXPECT().
					DeleteByAccountNumber(args.ctx, args.accountNumber).
					Return(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}
			err := testHelper.accountService.Delete(tt.args.ctx, tt.args.accountNumber)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
