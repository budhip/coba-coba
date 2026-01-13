package repositories

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting/mock"
	mockFlag "bitbucket.org/Amartha/go-fp-transaction/internal/common/flag/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/go-cmp/cmp"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestAccountRepositoryTestSuite(t *testing.T) {
	t.Helper()
	suite.Run(t, new(accountTestSuite))
}

type accountTestSuite struct {
	suite.Suite
	t        *testing.T
	writeDB  *sql.DB
	readDB   *sql.DB
	mock     sqlmock.Sqlmock
	mockFlag *mockFlag.MockClient
	repo     AccountRepository
}

func (suite *accountTestSuite) SetupTest() {
	var err error
	var cfg config.Config

	cfg.AccountFeatureConfig = map[string]config.FeatureConfig{
		models.DefaultPresetWalletFeature: {},
	}

	suite.writeDB, suite.mock, err = sqlmock.New()
	require.NoError(suite.T(), err)

	suite.readDB = suite.writeDB

	suite.t = suite.T()
	mockCtrl := gomock.NewController(suite.t)
	suite.mockFlag = mockFlag.NewMockClient(mockCtrl)

	mockAccounting := mock.NewMockClient(mockCtrl)

	suite.repo = NewSQLRepository(suite.writeDB, suite.readDB, cfg, suite.mockFlag, mockAccounting).GetAccountRepository()
}

func (suite *accountTestSuite) TearDownTest() {
	defer suite.writeDB.Close()
	defer suite.readDB.Close()

}

func (suite *accountTestSuite) TestRepository_CountAll() {
	type args struct {
		ctx  context.Context
		opts models.AccountFilterOptions
	}

	testCases := []struct {
		name       string
		args       args
		setupMocks func(a args)
		wantErr    bool
	}{
		{
			name: "test success",
			args: args{
				ctx: context.TODO(),
			},
			setupMocks: func(a args) {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryEstimateCountAccount)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			wantErr: false,
		},
		{
			name: "test error result",
			args: args{
				ctx: context.TODO(),
			},
			setupMocks: func(a args) {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryEstimateCountAccount)).WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks(tt.args)

			_, err := suite.repo.CountAll(tt.args.ctx, tt.args.opts)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_Create() {
	type args struct {
		ctx        context.Context
		req        models.CreateAccount
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
				ctx: context.TODO(),
				req: models.CreateAccount{},
				setupMocks: func() {
					suite.mock.ExpectExec(regexp.QuoteMeta(queryAccountCreate)).WillReturnResult(sqlmock.NewResult(0, 1))
				},
			},
			wantErr: false,
		},
		{
			name: "test error result",
			args: args{
				ctx: context.TODO(),
				req: models.CreateAccount{},
				setupMocks: func() {
					suite.mock.ExpectExec(regexp.QuoteMeta(queryAccountCreate)).WillReturnResult(sqlmock.NewErrorResult(assert.AnError))
				},
			},
			wantErr: true,
		},
		{
			name: "test error no row affected",
			args: args{
				ctx: context.TODO(),
				req: models.CreateAccount{},
				setupMocks: func() {
					suite.mock.ExpectExec(regexp.QuoteMeta(queryAccountCreate)).WillReturnResult(sqlmock.NewResult(0, 0))
				},
			},
			wantErr: true,
		},
		{
			name: "test error db",
			args: args{
				ctx: context.TODO(),
				req: models.CreateAccount{},
				setupMocks: func() {
					suite.mock.ExpectExec(regexp.QuoteMeta(queryAccountCreate)).WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			err := suite.repo.Create(tt.args.ctx, tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_CheckDataByID() {
	type args struct {
		ctx        context.Context
		id         uint64
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
				ctx: context.TODO(),
				id:  1,
				setupMocks: func() {
					rows := sqlmock.NewRows([]string{"id"}).AddRow(1)
					suite.mock.ExpectQuery(regexp.QuoteMeta(QueryAccountCheckDataById)).WithArgs(1).WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
		{
			name: "test error no rows",
			args: args{
				ctx: context.TODO(),
				id:  1,
				setupMocks: func() {
					rows := sqlmock.NewRows([]string{"id"})
					suite.mock.ExpectQuery(regexp.QuoteMeta(QueryAccountCheckDataById)).WithArgs(1).WillReturnRows(rows)
				},
			},
			wantErr: true,
		},
		{
			name: "test error result",
			args: args{
				ctx: context.TODO(),
				id:  1,
				setupMocks: func() {
					suite.mock.ExpectQuery(regexp.QuoteMeta(QueryAccountCheckDataById)).WithArgs(1).WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()
			err := suite.repo.CheckDataByID(tt.args.ctx, tt.args.id)
			assert.Equal(t, tt.wantErr, err != nil)
			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_GetAll() {
	type args struct {
		opts models.AccountFilterOptions
	}

	defaultColumns := []string{
		"id", "accountNumber", "ownerId", "categoryName",
		"subCategoryName", "entityName", "currency",
		"actualBalance", "pendingBalance", "status",
		"createdAt", "updatedAt", "accountName"}
	defaultCreated := common.Now()

	testCases := []struct {
		name       string
		args       args
		setupMocks func(a args)
		wantErr    bool
		expected   []models.GetAccountOut
	}{
		{
			name: "success get All Account",
			args: args{
				opts: models.AccountFilterOptions{
					Search:      "123456",
					AccountName: "John Doe",
				},
			},
			setupMocks: func(a args) {
				listQuery, _, _ := buildListAccountQuery(a.opts)
				rows := sqlmock.
					NewRows(defaultColumns).
					AddRow(
						1, "123456", "666", "category name here",
						"subCategory name here", "entity name here",
						"IDR", "420.69", "0",
						"active", defaultCreated, defaultCreated, "John")
				suite.mockFlag.EXPECT().
					IsEnabled(gomock.Any()).
					Return(true)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(listQuery)).
					WillReturnRows(rows)
			},
			expected: []models.GetAccountOut{
				{
					ID:            1,
					AccountName:   "John",
					AccountNumber: "123456",
					OwnerID:       "666",
					Category:      "category name here",
					SubCategory:   "subCategory name here",
					Entity:        "entity name here",
					Currency:      "IDR",
					Status:        "active",
					Balance:       models.NewBalance(decimal.NewFromFloat(420.69), decimal.Zero),
					CreatedAt:     defaultCreated,
					UpdatedAt:     defaultCreated,
				},
			},
			wantErr: false,
		},
		{
			name: "error query get All Account",
			setupMocks: func(a args) {
				listQuery, _, _ := buildListAccountQuery(a.opts)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(listQuery)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "error scan row get All Account",
			setupMocks: func(a args) {
				listQuery, _, _ := buildListAccountQuery(a.opts)
				rows := sqlmock.NewRows([]string{"InvalidColumn"}).AddRow(nil)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(listQuery)).
					WillReturnRows(rows)
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks(tt.args)

			actual, err := suite.repo.GetList(context.Background(), tt.args.opts)
			assert.Equal(t, tt.wantErr, err != nil)

			if !cmp.Equal(tt.expected, actual, balanceComparer()) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(tt.expected, actual, balanceComparer()))
			}

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_CheckAccountNumbers() {
	type args struct {
		ctx        context.Context
		req        []string
		setupMocks func()
	}

	testCases := []struct {
		name     string
		args     args
		wantErr  bool
		expected map[string]bool
	}{
		{
			name: "success check account numbers",
			args: args{
				ctx: context.TODO(),
				req: []string{"123456", "654321"},
				setupMocks: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryCheckByAccountNumbers)).
						WillReturnRows(sqlmock.NewRows([]string{"accountNumber"}).
							AddRow("123456").
							AddRow(("654321")))
				},
			},
			expected: map[string]bool{"123456": true, "654321": true},
			wantErr:  false,
		},
		{
			name: "success check account numbers (not exists 1 result)",
			args: args{
				ctx: context.TODO(),
				req: []string{"123456", "654321"},
				setupMocks: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryCheckByAccountNumbers)).
						WithArgs(pq.Array([]string{"123456", "654321"})).
						WillReturnRows(sqlmock.NewRows([]string{"accountNumber"}).AddRow("123456"))
				},
			},
			expected: map[string]bool{"123456": true, "654321": false},
			wantErr:  false,
		},
		{
			name: "error query acount number",
			args: args{
				ctx: context.TODO(),
				req: []string{"123456"},
				setupMocks: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryCheckByAccountNumbers)).
						WithArgs(pq.Array([]string{"123456"})).
						WillReturnError(assert.AnError)
				},
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "failed to scan result account numbers",
			args: args{
				ctx: context.TODO(),
				req: []string{"123456"},
				setupMocks: func() {

					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryCheckByAccountNumbers)).
						WithArgs(pq.Array([]string{"123456"})).
						WillReturnRows(sqlmock.
							NewRows([]string{"accountNumber"}).
							AddRow(nil))
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			actual, err := suite.repo.CheckAccountNumbers(tt.args.ctx, tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.expected, actual)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_GetAccountBalances() {
	type args struct {
		ctx        context.Context
		req        models.GetAccountBalanceRequest
		setupMocks func()
	}

	updatedAt := time.Now()
	testCases := []struct {
		name     string
		args     args
		wantErr  bool
		expected map[string]models.Balance
	}{
		{
			name: "success get account balance",
			args: args{
				ctx: context.TODO(),
				req: models.GetAccountBalanceRequest{
					AccountNumbers: []string{"123456", "654321"},
				},
				setupMocks: func() {
					query, _, _ := buildGetAccountBalancesQuery(models.GetAccountBalanceRequest{
						AccountNumbers: []string{"123456", "654321"},
					})
					suite.mockFlag.EXPECT().
						IsEnabled(gomock.Any()).
						Return(true)
					suite.mockFlag.EXPECT().
						IsEnabled(gomock.Any()).
						Return(true)
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(query)).
						WillReturnRows(sqlmock.NewRows([]string{"accountNumber", "actualBalance", "pendingBalance", "version", "updatedAt"}).
							AddRow("123456", "420.69", "0", 1, updatedAt).
							AddRow("654321", "69.420", "0", 1, updatedAt))
				},
			},
			expected: map[string]models.Balance{
				"123456": models.NewBalance(decimal.NewFromFloat(420.69), decimal.Zero),
				"654321": models.NewBalance(decimal.NewFromFloat(69.420), decimal.Zero),
			},
			wantErr: false,
		},
		{
			name: "success get account balance for update",
			args: args{
				ctx: context.TODO(),
				req: models.GetAccountBalanceRequest{
					AccountNumbers: []string{"123456", "654321"},
					ForUpdate:      true,
				},
				setupMocks: func() {
					query, _, _ := buildGetAccountBalancesQuery(models.GetAccountBalanceRequest{
						AccountNumbers: []string{"123456", "654321"},
					})
					suite.mockFlag.EXPECT().
						IsEnabled(gomock.Any()).
						Return(true)
					suite.mockFlag.EXPECT().
						IsEnabled(gomock.Any()).
						Return(true)
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(query)).
						WillReturnRows(sqlmock.NewRows([]string{"accountNumber", "actualBalance", "pendingBalance", "version", "updatedAt"}).
							AddRow("123456", "420.69", "0", 1, updatedAt).
							AddRow("654321", "69.420", "0", 1, updatedAt))
				},
			},
			expected: map[string]models.Balance{
				"123456": models.NewBalance(decimal.NewFromFloat(420.69), decimal.Zero),
				"654321": models.NewBalance(decimal.NewFromFloat(69.420), decimal.Zero),
			},
			wantErr: false,
		},
		{
			name: "error query account balances",
			args: args{
				ctx: context.TODO(),
				req: models.GetAccountBalanceRequest{
					AccountNumbers: []string{"123456"},
				},
				setupMocks: func() {
					query, _, _ := buildGetAccountBalancesQuery(models.GetAccountBalanceRequest{
						AccountNumbers: []string{"123456"},
					})

					suite.mock.
						ExpectQuery(regexp.QuoteMeta(query)).
						WillReturnError(assert.AnError)
				},
			},
			expected: nil,
			wantErr:  true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			actual, err := suite.repo.GetAccountBalances(tt.args.ctx, tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)
			if !cmp.Equal(tt.expected, actual, balanceComparer()) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(tt.expected, actual, balanceComparer()))
			}

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_UpdateAccountBalance() {
	defaultBalance := models.NewBalance(decimal.NewFromFloat(420.69), decimal.Zero)

	type args struct {
		ctx           context.Context
		accountNumber string
		balance       models.Balance
		setupMocks    func()
	}
	testCases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success - update account balance",
			args: args{
				ctx:           context.TODO(),
				accountNumber: "123456",
				balance:       defaultBalance,
				setupMocks: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryGetAccountVersion)).
						WillReturnRows(sqlmock.NewRows([]string{"version"}).
							AddRow(1))
					suite.mockFlag.EXPECT().
						IsEnabled(gomock.Any()).
						Return(true)
					suite.mock.ExpectExec(regexp.QuoteMeta(queryUpdateAccountBalance)).WillReturnResult(sqlmock.NewResult(0, 1))
				},
			},
			wantErr: false,
		},
		{
			name: "error - get account version",
			args: args{
				ctx:           context.TODO(),
				accountNumber: "123456",
				balance:       defaultBalance,
				setupMocks: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryGetAccountVersion)).WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
		{
			name: "error - update account balance",
			args: args{
				ctx:           context.TODO(),
				accountNumber: "123456",
				balance:       defaultBalance,
				setupMocks: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryGetAccountVersion)).
						WillReturnRows(sqlmock.NewRows([]string{"version"}).
							AddRow(1))
					suite.mock.ExpectExec(regexp.QuoteMeta(queryUpdateAccountBalance)).WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
		{
			name: "error - update account balance no result",
			args: args{
				ctx:           context.TODO(),
				accountNumber: "123456",
				balance:       defaultBalance,
				setupMocks: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryGetAccountVersion)).
						WillReturnRows(sqlmock.NewRows([]string{"version"}).
							AddRow(1))
					suite.mock.ExpectExec(regexp.QuoteMeta(queryUpdateAccountBalance)).WillReturnResult(sqlmock.NewResult(0, 0))
				},
			},
			wantErr: true,
		},
		{
			name: "error - update account balance error result",
			args: args{
				ctx:           context.TODO(),
				accountNumber: "123456",
				balance:       defaultBalance,
				setupMocks: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryGetAccountVersion)).
						WillReturnRows(sqlmock.NewRows([]string{"version"}).
							AddRow(1))
					suite.mock.ExpectExec(regexp.QuoteMeta(queryUpdateAccountBalance)).WillReturnResult(sqlmock.NewErrorResult(assert.AnError))
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			_, err := suite.repo.UpdateAccountBalance(tt.args.ctx, tt.args.accountNumber, tt.args.balance)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_GetOneByAccountNumber() {
	accountNumberTest := "[TEST]"

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
			name: "failed - test error result",
			args: args{
				ctx:           context.TODO(),
				accountNumber: accountNumberTest,
				setupMocks: func() {
					suite.mock.ExpectQuery(regexp.QuoteMeta(GetOneByAccountNumber)).
						WithArgs(accountNumberTest).
						WillReturnError(assert.AnError)
					suite.mockFlag.EXPECT().
						IsEnabled(gomock.Any()).
						Return(true)
				},
			},
			wantErr: true,
		},

		{
			name: "success - test success query",
			args: args{
				ctx:           context.TODO(),
				accountNumber: accountNumberTest,
				setupMocks: func() {
					rows := sqlmock.
						NewRows([]string{"id", "accountNumber", "ownerId", "categoryName", "subCategoryName", "entityName", "currency", "status", "isHvt", "actualBalance", "pendingBalance", "createdAt", "updatedAt", "legacyId", "featurePreset", "featureBalanceRangeMin", "featureBalanceRangeMax", "featureNegativeBalanceAllowed", "featureNegativeBalanceLimit", "accountName"}).
						AddRow(1, "accountNumber", "ownerId", "category", "subCategory", "entity", "currency", "status", true, "420.69", "0", common.Now(), common.Now(), nil, "customer", 10000, 200000, false, 1000, "John")
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(GetOneByAccountNumber)).
						WithArgs(accountNumberTest).
						WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {

			tt.args.setupMocks()

			_, err := suite.repo.GetOneByAccountNumber(tt.args.ctx, tt.args.accountNumber)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_GetAllWithoutPagination() {
	type args struct {
		ctx        context.Context
		setupMocks func()
	}

	testCases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx: context.TODO(),
				setupMocks: func() {
					rows := sqlmock.NewRows([]string{
						"accountNumber",
						"ownerId",
						"ownerType",
						"category",
						"parentAccountNumber",
						"balance",
					})
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryAccountGetAllWithoutPagination)).
						WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
		{
			name: "error query",
			args: args{
				ctx: context.TODO(),
				setupMocks: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryAccountGetAllWithoutPagination)).
						WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
		{
			name: "error scan row",
			args: args{
				ctx: context.TODO(),
				setupMocks: func() {
					rows := sqlmock.NewRows([]string{"InvalidColumn"}).AddRow(nil)
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryAccountGetAllWithoutPagination)).
						WillReturnRows(rows)
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			_, err := suite.repo.GetAllWithoutPagination(tt.args.ctx)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_GetAllByAccountNumbers() {
	type args struct {
		ctx            context.Context
		accountNumbers []string
		setupMocks     func()
	}

	testCases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx:            context.TODO(),
				accountNumbers: []string{},
				setupMocks: func() {
					rows := sqlmock.NewRows([]string{
						"accountNumber",
						"ownerId",
						"ownerType",
						"category",
						"parentAccountNumber",
						"Balance",
					})
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryGetAccountBalance)).
						WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
		{
			name: "error query",
			args: args{
				ctx:            context.TODO(),
				accountNumbers: []string{},
				setupMocks: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryGetAccountBalance)).
						WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
		{
			name: "error scan row",
			args: args{
				ctx: context.TODO(),
				setupMocks: func() {
					rows := sqlmock.NewRows([]string{"InvalidColumn"}).AddRow(nil)
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryGetAccountBalance)).
						WillReturnRows(rows)
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			_, err := suite.repo.GetAllByAccountNumbers(tt.args.ctx, tt.args.accountNumbers)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_GetAccountNumberEntity() {
	type args struct {
		ctx            context.Context
		accountNumbers []string
		setupMock      func()
	}

	testCases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{
				ctx:            context.TODO(),
				accountNumbers: []string{"222000071045"},
				setupMock: func() {
					rows := sqlmock.NewRows([]string{
						"accountNumber",
						"name",
						"ownerId",
						"productTypeName",
						"categoryCode",
						"subCategoryCode",
						"entityCode",
						"altId",
						"legacyId",
						"isHvt",
						"status",
						"metadata",
					}).AddRow(
						"222000071045",
						"Test Account",
						"owner-1",
						"ABC",
						"CAT1",
						"SUBCAT1",
						"ENT1",
						"ALT123",
						"LEG123",
						true,
						"ACTIVE",
						`{"meta":"data"}`,
					)

					suite.mock.
						ExpectQuery(regexp.QuoteMeta(`SELECT "accountNumber", COALESCE("name", '') AS "name", "ownerId", COALESCE("productTypeName", '') AS "productTypeName", "categoryCode", COALESCE("subCategoryCode", '') AS "subCategoryCode", "entityCode", COALESCE("altId", '') AS "altId", COALESCE("legacyId", '{}') AS "legacyId", "isHvt", "status", "metadata" FROM account`)).
						WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
		{
			name: "error query",
			args: args{
				ctx:            context.TODO(),
				accountNumbers: []string{"failed"},
				setupMock: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(`SELECT "accountNumber", COALESCE("name", '') AS "name", "ownerId", COALESCE("productTypeName", '') AS "productTypeName", "categoryCode", COALESCE("subCategoryCode", '') AS "subCategoryCode", "entityCode", COALESCE("altId", '') AS "altId", COALESCE("legacyId", '{}') AS "legacyId", "isHvt", "status", "metadata" FROM account`)).
						WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMock()

			_, err := suite.repo.GetAccountNumberEntity(tt.args.ctx, tt.args.accountNumbers)
			assert.Equal(t, tt.wantErr, err != nil)

			if err := suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_GetTotalBalance() {
	type Args struct {
		ctx  context.Context
		opts models.AccountFilterOptions
	}
	totalBalance := decimal.NewFromFloat(100)

	testCases := []struct {
		name       string
		args       Args
		setupMocks func(a Args)
		wantErr    bool
		expected   *decimal.Decimal
	}{
		{
			name: "happy path",
			args: Args{
				ctx:  context.TODO(),
				opts: models.AccountFilterOptions{},
			},
			setupMocks: func(a Args) {
				listQuery, _, _ := buildTotalBalanceAccountQuery(a.opts)

				suite.mock.ExpectQuery(regexp.QuoteMeta(listQuery)).
					WillReturnRows(sqlmock.
						NewRows([]string{"totalBalance"}).
						AddRow(totalBalance))
			},
			expected: &totalBalance,
			wantErr:  false,
		},
		{
			name: "failed - error database",
			args: Args{
				ctx:  context.TODO(),
				opts: models.AccountFilterOptions{},
			},
			setupMocks: func(a Args) {
				listQuery, _, _ := buildTotalBalanceAccountQuery(a.opts)

				suite.mock.ExpectQuery(regexp.QuoteMeta(listQuery)).
					WillReturnError(assert.AnError)
			},
			expected: nil,
			wantErr:  true,
		},
		{
			name: "failed - err scan",
			args: Args{
				ctx:  context.TODO(),
				opts: models.AccountFilterOptions{},
			},
			setupMocks: func(a Args) {
				listQuery, _, _ := buildTotalBalanceAccountQuery(a.opts)

				suite.mock.ExpectQuery(regexp.QuoteMeta(listQuery)).
					WillReturnRows(sqlmock.
						NewRows([]string{"totalBalance"}).
						AddRow("totalBalance"))
			},
			expected: nil,
			wantErr:  true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks(tt.args)

			actual, err := suite.repo.GetTotalBalance(tt.args.ctx, tt.args.opts)
			assert.Equal(t, tt.wantErr, err != nil)
			if tt.expected != nil && !tt.expected.Equal(*actual) {
				t.Error("expected to equal")
			}

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_Upsert() {
	type args struct {
		ctx        context.Context
		entity     models.AccountUpsert
		setupMocks func()
	}
	testCases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success - upsert account",
			args: args{
				ctx: context.TODO(),
				entity: models.AccountUpsert{
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
				setupMocks: func() {
					suite.mock.ExpectExec(regexp.QuoteMeta(queryAccountUpsert)).WillReturnResult(sqlmock.NewResult(0, 1))
				},
			},
			wantErr: false,
		},
		{
			name: "failed - upsert account",
			args: args{
				ctx: context.TODO(),
				entity: models.AccountUpsert{
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
				setupMocks: func() {
					suite.mock.ExpectExec(regexp.QuoteMeta(queryAccountUpsert)).WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
		{
			name: "test error no row affected",
			args: args{
				ctx: context.TODO(),
				entity: models.AccountUpsert{
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
				setupMocks: func() {
					suite.mock.ExpectExec(regexp.QuoteMeta(queryAccountUpsert)).WillReturnResult(sqlmock.NewResult(0, 0))
				},
			},
			wantErr: true,
		},
		{
			name: "test error db",
			args: args{
				ctx: context.TODO(),
				entity: models.AccountUpsert{
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
				setupMocks: func() {
					suite.mock.ExpectExec(regexp.QuoteMeta(queryAccountUpsert)).WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()
			err := suite.repo.Upsert(tt.args.ctx, tt.args.entity)
			assert.Equal(t, tt.wantErr, err != nil)
			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_GetOneByLegacyId() {
	testCases := []struct {
		name       string
		setupMocks func()
		wantErr    bool
	}{
		{
			name: "happy path",
			setupMocks: func() {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryGetOneByLegacyId)).
					WillReturnRows(
						sqlmock.
							NewRows([]string{"id", "accountNumber", "actualBalance", "pendingBalance"}).
							AddRow("1", "accountNumber", "420.69", "0"),
					)
			},
			wantErr: false,
		},
		{
			name: "failed - err db",
			setupMocks: func() {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryGetOneByLegacyId)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed - err scan",
			setupMocks: func() {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryGetOneByLegacyId)).
					WillReturnRows(
						sqlmock.
							NewRows([]string{"INVALID"}).
							AddRow(nil),
					)
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			_, err := suite.repo.GetOneByLegacyId(context.Background(), "accNumber")
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_Update() {
	query := `
	UPDATE "account"
	SET "isHvt" = $1, "status" = $2, "updatedAt" = now() WHERE "id" = $3;`
	testCases := []struct {
		name       string
		wantErr    bool
		setupMocks func(id int, newData models.UpdateAccountIn)
	}{
		{
			name: "happy path",
			setupMocks: func(id int, newData models.UpdateAccountIn) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(query)).
					WithArgs(newData.IsHVT, newData.Status, id).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "failed - err query",
			setupMocks: func(id int, newData models.UpdateAccountIn) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(query)).
					WithArgs(newData.IsHVT, newData.Status, id).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed - no affected",
			setupMocks: func(id int, newData models.UpdateAccountIn) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(query)).
					WithArgs(newData.IsHVT, newData.Status, id).
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			id := 99
			newData := models.UpdateAccountIn{
				IsHVT:  new(bool),
				Status: "active",
			}
			if tt.setupMocks != nil {
				tt.setupMocks(id, newData)
			}
			err := suite.repo.Update(context.Background(), id, newData)
			assert.Equal(t, tt.wantErr, err != nil)
			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_Delete() {
	testCases := []struct {
		name         string
		wantErr      bool
		id           int
		rowsAffected int64
		doMock       func(id int, rowsAffected int64)
	}{
		{
			name:         "happy path",
			id:           1,
			rowsAffected: 1,
			doMock: func(id int, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryAccountDelete)).
					WithArgs(id).
					WillReturnResult(sqlmock.NewResult(0, rowsAffected))
			},
			wantErr: false,
		},
		{
			name:         "failed - no rows affected",
			id:           1,
			rowsAffected: 0,
			doMock: func(id int, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryAccountDelete)).
					WithArgs(id).
					WillReturnResult(sqlmock.NewResult(0, rowsAffected))
			},
			wantErr: true,
		},
		{
			name: "failed - err db",
			doMock: func(id int, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryAccountDelete)).
					WithArgs(id).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			tc.doMock(tc.id, tc.rowsAffected)

			err := suite.repo.Delete(context.Background(), tc.id)
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_UpdateBySubCategory() {
	query := `
	UPDATE account SET "productTypeName" = $1, "currency" = $2, "updatedAt" = $3 WHERE "subCategoryCode" = $4
	`

	testCases := []struct {
		name         string
		wantErr      bool
		args         models.UpdateAccountBySubCategoryIn
		rowsAffected int64
		doMock       func(args models.UpdateAccountBySubCategoryIn, rowsAffected int64)
	}{
		{
			name: "success",
			args: models.UpdateAccountBySubCategoryIn{
				Code:            "10000",
				ProductTypeName: &[]string{"test"}[0],
				Currency:        &[]string{"IDR"}[0],
			},
			rowsAffected: 1,
			doMock: func(args models.UpdateAccountBySubCategoryIn, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(query)).
					WithArgs(*args.ProductTypeName, *args.Currency, "now()", args.Code).
					WillReturnResult(sqlmock.NewResult(0, rowsAffected))
			},
			wantErr: false,
		},
		{
			name: "failed - err db",
			args: models.UpdateAccountBySubCategoryIn{
				Code:            "10000",
				ProductTypeName: &[]string{"test"}[0],
				Currency:        &[]string{"IDR"}[0],
			},
			doMock: func(args models.UpdateAccountBySubCategoryIn, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(query)).
					WithArgs(*args.ProductTypeName, *args.Currency, "now()", args.Code).
					WillReturnResult(sqlmock.NewResult(0, rowsAffected)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			tc.doMock(tc.args, tc.rowsAffected)

			err := suite.repo.UpdateBySubCategory(context.Background(), tc.args)
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_DeleteByAccountNumber() {
	testCases := []struct {
		name          string
		wantErr       bool
		accountNumber string
		rowsAffected  int64
		doMock        func(accountNumber string, rowsAffected int64)
	}{
		{
			name:          "happy path",
			accountNumber: "123456",
			rowsAffected:  1,
			doMock: func(accountNumber string, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryDeleteAccountByAccountNumber)).
					WithArgs(accountNumber).
					WillReturnResult(sqlmock.NewResult(0, rowsAffected))
			},
			wantErr: false,
		},
		{
			name:          "failed - no rows affected",
			accountNumber: "123456",
			rowsAffected:  0,
			doMock: func(accountNumber string, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryDeleteAccountByAccountNumber)).
					WithArgs(accountNumber).
					WillReturnResult(sqlmock.NewResult(0, rowsAffected))
			},
			wantErr: true,
		},
		{
			name: "failed - err db",
			doMock: func(accountNumber string, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryDeleteAccountByAccountNumber)).
					WithArgs(accountNumber).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			tc.doMock(tc.accountNumber, tc.rowsAffected)

			err := suite.repo.DeleteByAccountNumber(context.Background(), tc.accountNumber)
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *accountTestSuite) TestRepository_GetOneByAccountNumberOrLegacyId() {
	accountNumberTest := "[TEST]"

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
			name: "failed - test error result",
			args: args{
				ctx:           context.TODO(),
				accountNumber: accountNumberTest,
				setupMocks: func() {
					suite.mock.ExpectQuery(regexp.QuoteMeta(newQueryGetOneByAccountNumber)).
						WithArgs(accountNumberTest).
						WillReturnError(assert.AnError)
					suite.mockFlag.EXPECT().
						IsEnabled(gomock.Any()).
						Return(true)
				},
			},
			wantErr: true,
		},

		{
			name: "success - test success query",
			args: args{
				ctx:           context.TODO(),
				accountNumber: accountNumberTest,
				setupMocks: func() {
					rows := sqlmock.
						NewRows([]string{"id", "accountNumber", "ownerId", "categoryName", "subCategoryName", "entityName", "currency", "status", "isHvt", "actualBalance", "pendingBalance", "createdAt", "updatedAt", "legacyId", "accountName", "featurePreset", "featureBalanceRangeMin", "featureBalanceRangeMax", "featureNegativeBalanceAllowed", "featureNegativeBalanceLimit"}).
						AddRow(1, "accountNumber", "ownerId", "category", "subCategory", "entity", "currency", "status", true, "420.69", "0", common.Now(), common.Now(), nil, "John", "customer", 10000, 100000, false, 1000)
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(newQueryGetOneByAccountNumber)).
						WithArgs(accountNumberTest).
						WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {

			tt.args.setupMocks()

			_, err := suite.repo.GetOneByAccountNumberOrLegacyId(tt.args.ctx, tt.args.accountNumber)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
