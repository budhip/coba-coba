package repositories

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting/mock"
	mockFlag "bitbucket.org/Amartha/go-fp-transaction/internal/common/flag/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/Unleash/unleash-client-go/v3/api"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestBalanceRepositoryTestSuite(t *testing.T) {
	t.Helper()
	suite.Run(t, new(balanceTestSuite))
}

type balanceTestSuite struct {
	suite.Suite
	t        *testing.T
	writeDB  *sql.DB
	readDB   *sql.DB
	mock     sqlmock.Sqlmock
	mockFlag *mockFlag.MockClient
	repo     BalanceRepository
}

func (suite *balanceTestSuite) SetupTest() {
	var err error
	var cfg config.Config

	cfg.AccountFeatureConfig = map[string]config.FeatureConfig{
		models.DefaultPresetWalletFeature: {},
	}

	suite.writeDB, suite.mock, err = sqlmock.New()
	require.NoError(suite.T(), err)

	suite.readDB = suite.writeDB
	require.NoError(suite.T(), err)

	suite.t = suite.T()

	mockCtrl := gomock.NewController(suite.t)
	defer mockCtrl.Finish()

	mockAccounting := mock.NewMockClient(mockCtrl)

	suite.mockFlag = mockFlag.NewMockClient(mockCtrl)
	suite.repo = NewSQLRepository(suite.writeDB, suite.readDB, cfg, suite.mockFlag, mockAccounting).GetBalanceRepository()
}

func (suite *balanceTestSuite) TearDownTest() {
	defer suite.writeDB.Close()
	defer suite.readDB.Close()
}

func (suite *balanceTestSuite) TestRepository_Get() {
	type args struct {
		ctx           context.Context
		accountNumber string
		setupMocks    func()
	}

	testCases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test success",
			args: args{
				ctx:           context.Background(),
				accountNumber: "211",
				setupMocks: func() {
					suite.mockFlag.EXPECT().
						GetVariant(gomock.Any()).
						Return(&api.Variant{
							Payload: api.Payload{Value: `["123456"]`},
							Enabled: true,
						})

					cols := []string{
						"accountNumber",
						"t24AccountNumber",
						"actual",
						"pending",
						"isHVT",
						"version",
						"lastUpdatedAt",
						"preset",
						"allowedNegativeBalance",
						"balanceRangeMin",
						"negativeBalanceLimit",
						"balanceRangeMax",
					}

					rows := sqlmock.
						NewRows(cols).
						AddRow("211", "", 100, 200, nil, nil, time.Now(), nil, nil, nil, nil, nil)
					suite.mockFlag.EXPECT().
						IsEnabled(gomock.Any()).
						Return(true)
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryGetAccountBalanceWithFeature)).
						WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
		{
			name: "test get variant error",
			args: args{
				ctx:           context.Background(),
				accountNumber: "211",
				setupMocks: func() {
					suite.mockFlag.EXPECT().
						GetVariant(gomock.Any()).
						Return(&api.Variant{
							Payload: api.Payload{Value: "!@#!@#!@#!@#INVALID_JSON"},
							Enabled: false,
						})
				},
			},
			wantErr: true,
		},
		{
			name: "test data not found",
			args: args{
				ctx: context.TODO(),
				setupMocks: func() {
					suite.mockFlag.EXPECT().
						GetVariant(gomock.Any()).
						Return(&api.Variant{
							Payload: api.Payload{Value: "[]"},
						})

					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryGetAccountBalanceWithFeature)).
						WillReturnError(sql.ErrNoRows)
				},
			},
			wantErr: true,
		},
		{
			name: "test error result",
			args: args{
				ctx: context.TODO(),
				setupMocks: func() {
					suite.mockFlag.EXPECT().
						GetVariant(gomock.Any()).
						Return(&api.Variant{
							Payload: api.Payload{Value: "[]"},
						})

					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryGetAccountBalanceWithFeature)).
						WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			_, err := suite.repo.Get(tt.args.ctx, tt.args.accountNumber)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *balanceTestSuite) TestRepository_GetMany() {
	type args struct {
		ctx        context.Context
		req        models.GetAccountBalanceRequest
		setupMocks func()
	}

	testCases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test success",
			args: args{
				ctx: context.Background(),
				req: models.GetAccountBalanceRequest{
					AccountNumbers: []string{"211"},
				},
				setupMocks: func() {
					suite.mockFlag.EXPECT().
						IsEnabled(gomock.Any()).
						Return(true)

					suite.mockFlag.EXPECT().
						GetVariant(gomock.Any()).
						Return(&api.Variant{
							Payload: api.Payload{Value: `["123456"]`},
							Enabled: true,
						})

					cols := []string{
						"accountNumber",
						"t24AccountNumber",
						"actual",
						"pending",
						"isHVT",
						"version",
						"lastUpdatedAt",
						"preset",
						"allowedNegativeBalance",
						"balanceRangeMin",
						"negativeBalanceLimit",
						"balanceRangeMax",
					}

					query, _, _ := buildGetManyAccountBalanceQuery(models.GetAccountBalanceRequest{
						AccountNumbers: []string{"211"},
					}, nil)

					rows := sqlmock.
						NewRows(cols).
						AddRow("211", "", 100, 200, nil, nil, time.Now(), nil, nil, nil, nil, nil)
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(query)).
						WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
		{
			name: "test data not found",
			args: args{
				ctx: context.TODO(),
				req: models.GetAccountBalanceRequest{
					AccountNumbers: []string{"211"},
				},
				setupMocks: func() {
					suite.mockFlag.EXPECT().
						GetVariant(gomock.Any()).
						Return(&api.Variant{
							Payload: api.Payload{Value: `["123456"]`},
							Enabled: true,
						})

					query, _, _ := buildGetManyAccountBalanceQuery(models.GetAccountBalanceRequest{
						AccountNumbers: []string{"211"},
					}, nil)
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(query)).
						WillReturnError(sql.ErrNoRows)
				},
			},
			wantErr: true,
		},
		{
			name: "test error result",
			args: args{
				ctx: context.TODO(),
				req: models.GetAccountBalanceRequest{
					AccountNumbers: []string{"211"},
				},
				setupMocks: func() {
					suite.mockFlag.EXPECT().
						GetVariant(gomock.Any()).
						Return(&api.Variant{
							Payload: api.Payload{Value: `["123456"]`},
							Enabled: true,
						})

					query, _, _ := buildGetManyAccountBalanceQuery(models.GetAccountBalanceRequest{
						AccountNumbers: []string{"211"},
					}, nil)
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(query)).
						WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			_, err := suite.repo.GetMany(tt.args.ctx, tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *balanceTestSuite) TestRepository_AdjustAccountBalance() {
	type args struct {
		ctx           context.Context
		accountNumber string
		updateAmount  models.Decimal
		setupMocks    func(string, models.Decimal)
	}

	testCases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test success",
			args: args{
				ctx:           context.Background(),
				accountNumber: "211",
				updateAmount:  models.NewDecimalFromExternal(decimal.NewFromFloat(100.4)),
				setupMocks: func(accountNumber string, updateAmount models.Decimal) {
					query := queryAdjustAccountBalance

					suite.mock.
						ExpectExec(regexp.QuoteMeta(query)).
						WithArgs(updateAmount, accountNumber).
						WillReturnResult(sqlmock.NewResult(0, 1))
				},
			},
			wantErr: false,
		},
		{
			name: "test data not found",
			args: args{
				ctx:           context.Background(),
				accountNumber: "211",
				updateAmount:  models.NewDecimalFromExternal(decimal.NewFromFloat(100.4)),
				setupMocks: func(accountNumber string, updateAmount models.Decimal) {
					query := queryAdjustAccountBalance

					suite.mock.
						ExpectExec(regexp.QuoteMeta(query)).
						WithArgs(updateAmount, accountNumber).
						WillReturnError(sql.ErrNoRows)
				},
			},
			wantErr: true,
		},
		{
			name: "test error result",
			args: args{
				ctx: context.Background(),
				setupMocks: func(accountNumber string, updateAmount models.Decimal) {
					query := queryAdjustAccountBalance

					suite.mock.
						ExpectExec(regexp.QuoteMeta(query)).
						WithArgs(updateAmount, accountNumber).
						WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks(tt.args.accountNumber, tt.args.updateAmount)

			err := suite.repo.AdjustAccountBalance(tt.args.ctx, tt.args.accountNumber, tt.args.updateAmount)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
