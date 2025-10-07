package repositories

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting/mock"
	mockFlag "bitbucket.org/Amartha/go-fp-transaction/internal/common/flag/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

func TestWalletTransactionRepositoryTestSuite(t *testing.T) {
	t.Helper()
	suite.Run(t, new(walletTransactionTestSuite))
}

type walletTransactionTestSuite struct {
	suite.Suite
	t        *testing.T
	writeDB  *sql.DB
	readDB   *sql.DB
	mock     sqlmock.Sqlmock
	mockFlag *mockFlag.MockClient
	repo     WalletTransactionRepository
}

func (suite *walletTransactionTestSuite) SetupTest() {
	var err error
	var cfg config.Config

	suite.writeDB, suite.mock, err = sqlmock.New()
	require.NoError(suite.T(), err)

	suite.readDB = suite.writeDB
	require.NoError(suite.T(), err)

	suite.t = suite.T()
	mockCtrl := gomock.NewController(suite.t)
	suite.mockFlag = mockFlag.NewMockClient(mockCtrl)

	mockAccounting := mock.NewMockClient(mockCtrl)

	suite.repo = NewSQLRepository(suite.writeDB, suite.readDB, cfg, suite.mockFlag, mockAccounting).GetWalletTransactionRepository()
}

func (suite *walletTransactionTestSuite) TearDownTest() {
	defer suite.writeDB.Close()
	defer suite.readDB.Close()

}

func (suite *walletTransactionTestSuite) TestRepository_Create() {
	vAmount := decimal.NewFromFloat(100)
	id := uuid.New().String()

	testCases := []struct {
		name       string
		args       models.NewWalletTransaction
		setupMocks func()
		wantErr    bool
	}{
		{
			name: "happy path",
			args: models.NewWalletTransaction{
				ID:              id,
				AccountNumber:   "111",
				RefNumber:       "222",
				TransactionType: "333",
				TransactionFlow: "444",
				TransactionTime: time.Now(),
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(vAmount),
				},
				Status: "PENDING",
			},
			setupMocks: func() {
				rows := sqlmock.
					NewRows([]string{
						"id", "status", "accountNumber", "destinationAccountNumber", "refNumber",
						"transactionType", "transactionTime", "transactionFlow",
						"netAmount", "breakdownAmounts",
						"description", "metadata", "createdAt",
					}).
					AddRow(
						id, "PENDING", "666", "999", "ref_123",
						"DSBAB", time.Now(), "cashin",
						100, "[]",
						"desc", "{}", time.Now(),
					)
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryWalletTrxCreate)).WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "failed - err sql",
			args: models.NewWalletTransaction{
				ID:              id,
				AccountNumber:   "111",
				RefNumber:       "222",
				TransactionType: "333",
				TransactionFlow: "444",
				TransactionTime: time.Now(),
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(vAmount),
				},
				Status: "PENDING",
			},
			setupMocks: func() {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryWalletTrxCreate)).WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			_, err := suite.repo.Create(context.Background(), tt.args)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *walletTransactionTestSuite) TestRepository_GetById() {
	ct := time.Now()
	amount := decimal.NewFromFloat(100)

	testCases := []struct {
		name       string
		args       string
		setupMocks func()
		wantErr    bool
		wantData   *models.WalletTransaction
	}{
		{
			name: "success get by id",
			args: "123123",
			setupMocks: func() {
				rows := sqlmock.
					NewRows([]string{
						"id", "status", "accountNumber", "destinationAccountNumber", "refNumber",
						"transactionType", "transactionTime", "transactionFlow",
						"netAmount", "breakdownAmounts",
						"description", "metadata", "createdAt",
					}).
					AddRow(
						"123123", "PENDING", "666", "999", "ref_123",
						"DSBAB", ct, "cashin",
						100, "[]",
						"desc", "{}", ct,
					)
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryWalletTrxGetByID)).WillReturnRows(rows)
			},
			wantData: &models.WalletTransaction{
				ID:                       "123123",
				Status:                   "PENDING",
				AccountNumber:            "666",
				DestinationAccountNumber: "999",
				RefNumber:                "ref_123",
				TransactionType:          "DSBAB",
				TransactionTime:          ct,
				TransactionFlow:          "cashin",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(amount),
					Currency:     models.IDRCurrency,
				},
				Amounts:     models.Amounts{},
				Description: "desc",
				Metadata:    models.WalletMetadata{},
				CreatedAt:   ct,
			},
			wantErr: false,
		},
		{
			name: "failed - err sql",
			args: "123123",
			setupMocks: func() {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryWalletTrxGetByID)).WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			actual, err := suite.repo.GetById(context.Background(), tt.args)
			assert.Equal(t, tt.wantErr, err != nil)

			if !cmp.Equal(tt.wantData, actual) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(tt.wantData, actual))
			}

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *walletTransactionTestSuite) TestRepository_Update() {
	ct := time.Now()
	statusSuccess := models.WalletTransactionStatusSuccess
	amount := decimal.NewFromFloat(100)
	query := `UPDATE wallet_transaction SET "transactionTime" = $1, "status" = $2 WHERE id = $3 RETURNING
			"id", "status", "accountNumber", "destinationAccountNumber", "refNumber", 
			"transactionType", "transactionTime", "transactionFlow",
			"netAmount", "breakdownAmounts",
			"description", "metadata", "createdAt"`

	type args struct {
		id   string
		data models.WalletTransactionUpdate
	}

	testCases := []struct {
		name       string
		args       args
		setupMocks func(args args)
		wantErr    bool
		wantData   *models.WalletTransaction
	}{
		{
			name: "success update status",
			args: args{
				id: "123123",
				data: models.WalletTransactionUpdate{
					Status:          &statusSuccess,
					TransactionTime: &ct,
				},
			},
			setupMocks: func(args args) {
				rows := sqlmock.
					NewRows([]string{
						"id", "status", "accountNumber", "destinationAccountNumber", "refNumber",
						"transactionType", "transactionTime", "transactionFlow",
						"netAmount", "breakdownAmounts",
						"description", "metadata", "createdAt",
					}).
					AddRow(
						"123123", "SUCCESS", "666", "999", "ref_123",
						"DSBAB", ct, "cashin",
						100, "[]",
						"desc", "{}", ct,
					)

				suite.mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnRows(rows)
			},
			wantData: &models.WalletTransaction{
				ID:                       "123123",
				Status:                   "SUCCESS",
				AccountNumber:            "666",
				DestinationAccountNumber: "999",
				RefNumber:                "ref_123",
				TransactionType:          "DSBAB",
				TransactionTime:          ct,
				TransactionFlow:          "cashin",
				NetAmount: models.Amount{
					ValueDecimal: models.NewDecimalFromExternal(amount),
					Currency:     models.IDRCurrency,
				},
				Amounts:     models.Amounts{},
				Description: "desc",
				Metadata:    models.WalletMetadata{},
				CreatedAt:   ct,
			},
			wantErr: false,
		},
		{
			name: "failed - err sql",
			args: args{
				id: "123123",
				data: models.WalletTransactionUpdate{
					Status:          &statusSuccess,
					TransactionTime: &ct,
				},
			},
			setupMocks: func(args args) {
				suite.mock.ExpectQuery(regexp.QuoteMeta(query)).WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks(tt.args)
			}

			actual, err := suite.repo.Update(context.Background(), tt.args.id, tt.args.data)
			assert.Equal(t, tt.wantErr, err != nil)

			if !cmp.Equal(tt.wantData, actual) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(tt.wantData, actual))
			}

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *walletTransactionTestSuite) TestRepository_GetByRefNumber() {

	testCases := []struct {
		name       string
		setupMocks func()
		wantErr    bool
	}{
		{
			name: "happy path",
			setupMocks: func() {
				rows := sqlmock.NewRows(
					[]string{"id", "status"}).AddRow("1", "SUCCESS")
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryWalletTrxGetByRefNumber)).WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "happy path - not found",
			setupMocks: func() {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryWalletTrxGetByRefNumber)).WillReturnError(sql.ErrNoRows)
			},
			wantErr: false,
		},
		{
			name: "failed - err sql",
			setupMocks: func() {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryWalletTrxGetByRefNumber)).WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			_, err := suite.repo.GetByRefNumber(context.Background(), "refNumber")
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *walletTransactionTestSuite) TestRepository_List() {
	defaultColumns := []string{
		"id", "status", "accountNumber",
		"destinationAccountNumber", "refNumber", "transactionType",
		"transactionTime", "transactionFlow", "netAmount",
		"breakdownAmounts", "description", "metadata", "createdAt",
	}
	timeVal := common.Now()

	testCases := []struct {
		name       string
		setupMocks func(opts models.WalletTrxFilterOptions)
		opts       models.WalletTrxFilterOptions
		wantErr    bool
		expected   []models.WalletTransaction
	}{
		{
			name: "happy path",
			setupMocks: func(opts models.WalletTrxFilterOptions) {
				listQuery, _, _ := buildListWalletTrxQuery(opts)
				rows := sqlmock.
					NewRows(defaultColumns).
					AddRow(
						"1", "2", "3",
						"4", "5", "6",
						timeVal, "7", "100.02",
						`[{"type": "TUPVU", "amount": {"value": 2900, "currency": "IDR"}}]`, "8", "{}", timeVal,
					)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(listQuery)).
					WillReturnRows(rows)
			},
			expected: []models.WalletTransaction{
				{
					ID:                       "1",
					Status:                   "2",
					AccountNumber:            "3",
					DestinationAccountNumber: "4",
					RefNumber:                "5",
					TransactionType:          "6",
					TransactionTime:          timeVal,
					TransactionFlow:          "7",
					NetAmount: models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(decimal.NewFromFloat(100.02)),
						Currency:     "",
					},
					Amounts: []models.AmountDetail{
						{
							Type: "TUPVU",
							Amount: &models.Amount{
								ValueDecimal: models.NewDecimalFromExternal(decimal.NewFromInt(2900)),
								Currency:     "IDR",
							},
						},
					},
					Description: "8",
					CreatedAt:   timeVal,
					Metadata:    models.WalletMetadata{},
				},
			},
			wantErr: false,
		},
		{
			name: "happy path with accountNumbers",
			opts: models.WalletTrxFilterOptions{
				AccountNumbers: []string{"112233, 445566, 7788989"},
			},
			setupMocks: func(opts models.WalletTrxFilterOptions) {
				listQuery := `SELECT * FROM (SELECT "id", "status", "accountNumber", COALESCE("destinationAccountNumber", '') as "destinationAccountNumber", "refNumber", "transactionType", "transactionTime", "transactionFlow", "netAmount", "breakdownAmounts", COALESCE("description", '') as "description", "metadata", "createdAt" FROM wallet_transaction WHERE wallet_transaction."accountNumber" IN ($1) UNION SELECT "id", "status", "accountNumber", COALESCE("destinationAccountNumber", '') as "destinationAccountNumber", "refNumber", "transactionType", "transactionTime", "transactionFlow", "netAmount", "breakdownAmounts", COALESCE("description", '') as "description", "metadata", "createdAt" FROM wallet_transaction WHERE wallet_transaction."destinationAccountNumber" IN ($2)) AS union_trx ORDER BY "transactionTime" desc, "id" desc LIMIT 0`
				rows := sqlmock.
					NewRows(defaultColumns).
					AddRow(
						"1", "2", "3",
						"4", "5", "6",
						timeVal, "7", "100.02",
						`[{"type": "TUPVU", "amount": {"value": 2900, "currency": "IDR"}}]`, "8", "{}", timeVal,
					)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(listQuery)).
					WillReturnRows(rows)
			},
			expected: []models.WalletTransaction{
				{
					ID:                       "1",
					Status:                   "2",
					AccountNumber:            "3",
					DestinationAccountNumber: "4",
					RefNumber:                "5",
					TransactionType:          "6",
					TransactionTime:          timeVal,
					TransactionFlow:          "7",
					NetAmount: models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(decimal.NewFromFloat(100.02)),
						Currency:     "",
					},
					Amounts: []models.AmountDetail{
						{
							Type: "TUPVU",
							Amount: &models.Amount{
								ValueDecimal: models.NewDecimalFromExternal(decimal.NewFromInt(2900)),
								Currency:     "IDR",
							},
						},
					},
					Description: "8",
					CreatedAt:   timeVal,
					Metadata:    models.WalletMetadata{},
				},
			},
			wantErr: false,
		},
		{
			name: "happy path with transactionTypes",
			opts: models.WalletTrxFilterOptions{
				TransactionTypes: []string{"112233, 445566, 7788989"},
			},
			setupMocks: func(opts models.WalletTrxFilterOptions) {
				listQuery := `SELECT "id", "status", "accountNumber", COALESCE("destinationAccountNumber", '') as "destinationAccountNumber", "refNumber", "transactionType", "transactionTime", "transactionFlow", "netAmount", "breakdownAmounts", COALESCE("description", '') as "description", "metadata", "createdAt" FROM wallet_transaction WHERE "transactionType" IN ($1) ORDER BY "transactionTime" desc, "id" desc LIMIT 0`
				rows := sqlmock.
					NewRows(defaultColumns).
					AddRow(
						"1", "2", "3",
						"4", "5", "6",
						timeVal, "7", "100.02",
						`[{"type": "TUPVU", "amount": {"value": 2900, "currency": "IDR"}}]`, "8", "{}", timeVal,
					)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(listQuery)).
					WillReturnRows(rows)
			},
			expected: []models.WalletTransaction{
				{
					ID:                       "1",
					Status:                   "2",
					AccountNumber:            "3",
					DestinationAccountNumber: "4",
					RefNumber:                "5",
					TransactionType:          "6",
					TransactionTime:          timeVal,
					TransactionFlow:          "7",
					NetAmount: models.Amount{
						ValueDecimal: models.NewDecimalFromExternal(decimal.NewFromFloat(100.02)),
						Currency:     "",
					},
					Amounts: []models.AmountDetail{
						{
							Type: "TUPVU",
							Amount: &models.Amount{
								ValueDecimal: models.NewDecimalFromExternal(decimal.NewFromInt(2900)),
								Currency:     "IDR",
							},
						},
					},
					Description: "8",
					CreatedAt:   timeVal,
					Metadata:    models.WalletMetadata{},
				},
			},
			wantErr: false,
		},
		{
			name: "err db",
			setupMocks: func(opts models.WalletTrxFilterOptions) {
				listQuery, _, _ := buildListWalletTrxQuery(opts)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(listQuery)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "err scan",
			setupMocks: func(opts models.WalletTrxFilterOptions) {
				listQuery, _, _ := buildListWalletTrxQuery(opts)
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
			tt.setupMocks(models.WalletTrxFilterOptions{})

			actual, err := suite.repo.List(context.Background(), tt.opts)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.expected, actual)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *walletTransactionTestSuite) TestRepository_CountAll() {

	testCases := []struct {
		name       string
		setupMocks func(opts models.WalletTrxFilterOptions)
		wantErr    bool
	}{
		{
			name: "happy path",
			setupMocks: func(opts models.WalletTrxFilterOptions) {
				countQuery, _, _ := buildCountWalletTrxQuery(opts)
				suite.mock.ExpectQuery(regexp.QuoteMeta(countQuery)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			},
			wantErr: false,
		},
		{
			name: "err db",
			setupMocks: func(opts models.WalletTrxFilterOptions) {
				countQuery, _, _ := buildCountWalletTrxQuery(opts)
				suite.mock.ExpectQuery(regexp.QuoteMeta(countQuery)).WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks(models.WalletTrxFilterOptions{})

			_, err := suite.repo.CountAll(context.Background(), models.WalletTrxFilterOptions{})
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
