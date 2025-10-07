package repositories

import (
	"context"
	"database/sql"
	"regexp"
	"testing"

	"go.uber.org/mock/gomock"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting/mock"
	mockFlag "bitbucket.org/Amartha/go-fp-transaction/internal/common/flag/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestFeatureRepositoryTestSuite(t *testing.T) {
	t.Helper()
	suite.Run(t, new(featureTestSuite))
}

type featureTestSuite struct {
	suite.Suite
	t        *testing.T
	writeDB  *sql.DB
	readDB   *sql.DB
	mock     sqlmock.Sqlmock
	mockFlag *mockFlag.MockClient
	repo     FeatureRepository
}

func (suite *featureTestSuite) SetupTest() {
	var err error
	var cfg config.Config

	cfg.AccountFeatureConfig = map[string]config.FeatureConfig{
		"customer": {
			BalanceRangeMin:        10000,
			AllowedNegativeTrxType: []string{"TUPVA"},
			NegativeBalanceAllowed: true,
			NegativeLimit:          10000,
		},
	}
	suite.writeDB, suite.mock, err = sqlmock.New()
	require.NoError(suite.T(), err)

	suite.readDB = suite.writeDB
	require.NoError(suite.T(), err)

	suite.t = suite.T()
	mockCtrl := gomock.NewController(suite.t)
	suite.mockFlag = mockFlag.NewMockClient(mockCtrl)

	mockAccounting := mock.NewMockClient(mockCtrl)

	suite.repo = NewSQLRepository(suite.writeDB, suite.readDB, cfg, suite.mockFlag, mockAccounting).GetFeatureRepository()
}

func (suite *featureTestSuite) TearDownTest() {
	defer suite.writeDB.Close()
	defer suite.readDB.Close()
}

func (suite *featureTestSuite) TestRepository_GetFeatureByAccountNumbers() {
	type args struct {
		ctx            context.Context
		accountNumbers []string
	}

	columns := []string{"account_number", "preset", "balance_range_min", "negative_balance_allowed", "negative_balance_limit"}

	testCases := []struct {
		name       string
		args       args
		setupMocks func(a args)
		wantErr    bool
	}{
		{
			name: "success get feature by account numbers",
			args: args{
				ctx:            context.TODO(),
				accountNumbers: []string{"123456", "654321"},
			},
			setupMocks: func(a args) {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryGetFeatureByAccountNumber)).
					WillReturnRows(sqlmock.NewRows(columns).
						AddRow("123456", "customer", decimal.Zero, true, decimal.Zero).
						AddRow("654321", "customer", decimal.Zero, true, decimal.Zero))
			},
			wantErr: false,
		},
		{
			name: "success get feature by account numbers with empty result (use default value from config)",
			args: args{
				ctx:            context.TODO(),
				accountNumbers: []string{"123456", "654321"},
			},
			setupMocks: func(a args) {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryGetFeatureByAccountNumber)).
					WillReturnRows(sqlmock.NewRows(columns))
			},
			wantErr: false,
		},
		{
			name: "failed get data from database",
			args: args{
				ctx: context.TODO(),
			},
			setupMocks: func(a args) {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryGetFeatureByAccountNumber)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks(tt.args)

			_, err := suite.repo.GetFeatureByAccountNumbers(tt.args.ctx, tt.args.accountNumbers)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *featureTestSuite) TestRepository_Create() {
	mockAccountNumber := "40000133920"
	mockPreset := "customer"
	testCases := []struct {
		name    string
		in      *models.CreateWalletIn
		wantErr bool
		doMock  func(args models.CreateWalletIn)
	}{
		{
			name: "success",
			in: &models.CreateWalletIn{
				AccountNumber: mockAccountNumber,
				Feature: &models.WalletFeature{
					Preset: &mockPreset,
				},
			},
			doMock: func(in models.CreateWalletIn) {
				rows := sqlmock.
					NewRows([]string{"account_number", "preset", "balance_range_min", "balance_range_max", "negative_balance_allowed", "negative_balance_limit"}).
					AddRow(mockAccountNumber, "customer", 10000.0, 100000.0, false, 0.0)

				suite.mock.
					ExpectQuery(regexp.QuoteMeta(createFeatureQuery)).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "error",
			in: &models.CreateWalletIn{
				AccountNumber: mockAccountNumber,
				Feature: &models.WalletFeature{
					Preset: &mockPreset,
				},
			},
			doMock: func(in models.CreateWalletIn) {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(createFeatureQuery)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			tc.doMock(*tc.in)

			_, err := suite.repo.Register(context.Background(), tc.in)
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
