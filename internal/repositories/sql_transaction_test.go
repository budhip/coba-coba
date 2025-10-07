package repositories

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/lib/pq"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting/mock"
	mockFlag "bitbucket.org/Amartha/go-fp-transaction/internal/common/flag/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

func TestTransactionRepositoryTestSuite(t *testing.T) {
	t.Helper()
	suite.Run(t, new(TransactionTestSuite))
}

type TransactionTestSuite struct {
	suite.Suite
	t        *testing.T
	writeDB  *sql.DB
	readDB   *sql.DB
	mock     sqlmock.Sqlmock
	mockFlag *mockFlag.MockClient
	repo     TransactionRepository
}

func (suite *TransactionTestSuite) SetupTest() {
	var err error
	var cfg config.Config

	mockCtrl := gomock.NewController(suite.T())
	defer mockCtrl.Finish()

	suite.writeDB, suite.mock, err = sqlmock.New()
	require.NoError(suite.T(), err)

	suite.readDB = suite.writeDB

	require.NoError(suite.T(), err)

	suite.t = suite.T()
	suite.mockFlag = mockFlag.NewMockClient(mockCtrl)

	mockAccounting := mock.NewMockClient(mockCtrl)

	suite.repo = NewSQLRepository(suite.writeDB, suite.readDB, cfg, suite.mockFlag, mockAccounting).GetTransactionRepository()

}

func (suite *TransactionTestSuite) TearDownTest() {
	defer suite.writeDB.Close()
	defer suite.readDB.Close()

}

func (suite *TransactionTestSuite) SetupModel() *models.Transaction {
	parse, err := time.Parse("2006-01-02", "2023-02-01")
	assert.NoError(suite.T(), err)
	return &models.Transaction{
		FromAccount:     "1202517699",
		ToAccount:       "123233333",
		FromNarrative:   "TOPUP.TRX",
		ToNarrative:     "TOPUP",
		TransactionDate: parse,
		Amount:          decimal.NewNullDecimal(decimal.NewFromFloat(20000)),
		Status:          "",
		Method:          "TOPUP",
		TypeTransaction: "ACRF",
		Description:     "TOP UP",
		RefNumber:       "FT2303000001",
	}
}

func (suite *TransactionTestSuite) SetupModelArray() []*models.Transaction {
	parse, err := time.Parse("2006-01-02", "2023-02-01")
	assert.NoError(suite.T(), err)
	return []*models.Transaction{
		{
			FromAccount:     "1202517699",
			ToAccount:       "123233333",
			FromNarrative:   "TOPUP.TRX",
			ToNarrative:     "TOPUP",
			TransactionDate: parse,
			Amount:          decimal.NewNullDecimal(decimal.NewFromFloat(20000)),
			Status:          "",
			Method:          "TOPUP",
			TypeTransaction: "ACRF",
			Description:     "TOP UP",
			RefNumber:       "FT2303000001",
		}, {
			FromAccount:     "1202517699",
			ToAccount:       "123233333",
			FromNarrative:   "TOPUP.TRX",
			ToNarrative:     "TOPUP",
			TransactionDate: parse,
			Amount:          decimal.NewNullDecimal(decimal.NewFromFloat(20000)),
			Status:          "",
			Method:          "TOPUP",
			TypeTransaction: "ACRF",
			Description:     "TOP UP",
			RefNumber:       "FT2303000001",
		},
	}
}

func (suite *TransactionTestSuite) TestRepository_Store() {
	type args struct {
		ctx        context.Context
		entity     *models.Transaction
		setupMocks func(entity *models.Transaction)
	}

	ct := time.Now()

	testCases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test success",
			args: args{
				ctx:    context.TODO(),
				entity: suite.SetupModel(),
				setupMocks: func(entity *models.Transaction) {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(storeTrxQuery)).
						WillReturnRows(sqlmock.
							NewRows([]string{"id", "createdAt", "updatedAt"}).
							AddRow(1, ct, ct))
				},
			},
			wantErr: false,
		},
		{
			name: "test error storeTrx",
			args: args{
				ctx:    context.TODO(),
				entity: suite.SetupModel(),
				setupMocks: func(entity *models.Transaction) {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(storeTrxQuery)).
						WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks(tt.args.entity)

			err := suite.repo.Store(tt.args.ctx, suite.SetupModel())
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *TransactionTestSuite) TestRepository_GetAllTransaction() {
	type args struct {
		opts models.TransactionFilterOptions
	}

	defaultColumns := []string{
		"id",
		"transactionId",
		"refNumber",
		"orderType",
		"method",
		"typeTransaction",
		"transactionTime",
		"transactionDate",
		"fromAccount",
		"fromAccountProductTypeName",
		"fromAccountName",
		"toAccount",
		"toAccountProductTypeName",
		"toAccountName",
		"amount",
		"status",
		"description",
		"metadata",
		"createdAt",
		"updatedAt",
		"currency",
	}
	defaultDecimal := decimal.NewNullDecimal(decimal.NewFromInt(100000))
	defaultCreated := common.Now()

	testCases := []struct {
		name       string
		args       args
		setupMocks func(a args)
		wantErr    bool
		expected   []models.Transaction
	}{
		{
			name: "success get all transaction",
			args: args{
				opts: models.TransactionFilterOptions{
					Search:          "123456",
					OrderType:       "TOPUP",
					TransactionType: "TOPUP",
					StartDate:       &defaultCreated,
					EndDate:         &defaultCreated,
				},
			},
			setupMocks: func(a args) {
				listQuery, _, _ := buildListTransactionQuery(a.opts)
				rows := sqlmock.
					NewRows(defaultColumns).
					AddRow(
						123456,
						"c172ca84-9ae2-489c-ae4f-8ef372a109ae",
						"55aa66bb-e6e0-4065-9f4a-64182e97e9d9",
						"TOPUP",
						"TOPUP.VA",
						"TOPUP",
						defaultCreated,
						defaultCreated,
						"189513",
						"189513-product-name",
						"189513-name",
						"222000000069",
						"222000000069-product-name",
						"222000000069-name",
						defaultDecimal,
						"1",
						"transfer",
						"{}",
						defaultCreated,
						defaultCreated,
						models.IDRCurrency,
					)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(listQuery)).
					WillReturnRows(rows)
			},
			expected: []models.Transaction{
				{
					ID:                         123456,
					TransactionID:              "c172ca84-9ae2-489c-ae4f-8ef372a109ae",
					TransactionDate:            defaultCreated,
					FromAccount:                "189513",
					FromAccountProductTypeName: "189513-product-name",
					FromAccountName:            "189513-name",
					ToAccount:                  "222000000069",
					ToAccountProductTypeName:   "222000000069-product-name",
					ToAccountName:              "222000000069-name",
					Currency:                   models.IDRCurrency,
					Amount:                     defaultDecimal,
					Status:                     "SUCCESS",
					Method:                     "TOPUP.VA",
					TypeTransaction:            "TOPUP",
					Description:                "transfer",
					RefNumber:                  "55aa66bb-e6e0-4065-9f4a-64182e97e9d9",
					Metadata:                   "{}",
					CreatedAt:                  defaultCreated,
					UpdatedAt:                  defaultCreated,
					OrderType:                  "TOPUP",
					TransactionTime:            defaultCreated,
				},
			},
			wantErr: false,
		},
		{
			name: "error query get all transaction",
			setupMocks: func(a args) {
				listQuery, _, _ := buildListTransactionQuery(a.opts)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(listQuery)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "error scan row get all transaction",
			setupMocks: func(a args) {
				listQuery, _, _ := buildListTransactionQuery(a.opts)
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
			assert.Equal(t, tt.expected, actual)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *TransactionTestSuite) TestRepository_GetStatusCount() {
	type args struct {
		ctx        context.Context
		threshold  uint
		opts       models.TransactionFilterOptions
		setupMocks func(a args)
	}

	testCases := []struct {
		name         string
		args         args
		wantResponse models.StatusCountTransaction
		wantErr      bool
	}{
		{
			name: "test success",
			args: args{
				ctx: context.TODO(),
				setupMocks: func(a args) {
					query, _, _ := buildStatusCountTransactionQuery(a.threshold, a.opts)
					suite.mock.ExpectQuery(regexp.QuoteMeta(query)).
						WillReturnRows(sqlmock.NewRows([]string{"exceed_threshold"}).AddRow(true))
				},
			},
			wantResponse: models.StatusCountTransaction{
				ExceedThreshold: true,
			},
			wantErr: false,
		},
		{
			name: "error db",
			args: args{
				ctx: context.TODO(),
				setupMocks: func(a args) {
					query, _, _ := buildStatusCountTransactionQuery(a.threshold, a.opts)
					suite.mock.ExpectQuery(regexp.QuoteMeta(query)).
						WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks(tt.args)

			got, err := suite.repo.GetStatusCount(tt.args.ctx, tt.args.threshold, tt.args.opts)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantResponse, got)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *TransactionTestSuite) TestRepository_CountAll() {
	type args struct {
		ctx        context.Context
		opts       models.TransactionFilterOptions
		setupMocks func(a args)
	}

	countQuery := queryEstimateCountData

	testCases := []struct {
		name         string
		args         args
		wantResponse int
		wantErr      bool
	}{
		{
			name: "test success",
			args: args{
				ctx: context.TODO(),
				setupMocks: func(a args) {
					suite.mock.ExpectQuery(regexp.QuoteMeta(countQuery)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
				},
			},
			wantResponse: 10,
			wantErr:      false,
		},
		{
			name: "test success without result",
			args: args{
				ctx: context.TODO(),
				setupMocks: func(a args) {
					suite.mock.ExpectQuery(regexp.QuoteMeta(countQuery)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				},
			},
			wantResponse: 0,
			wantErr:      false,
		},
		{
			name: "error db",
			args: args{
				ctx: context.TODO(),
				setupMocks: func(a args) {
					suite.mock.ExpectQuery(regexp.QuoteMeta(countQuery)).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1)).WillReturnError(assert.AnError)
				},
			},
			wantResponse: 0,
			wantErr:      true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks(tt.args)

			got, err := suite.repo.CountAll(tt.args.ctx, tt.args.opts)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.wantResponse, got)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *TransactionTestSuite) TestRepository_GetByID() {
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
					timeParse, err := time.Parse(common.DateFormatYYYYMMDD, "2021-02-01")
					if err != nil {
						suite.t.Error(err)
					}
					rows := sqlmock.NewRows(
						[]string{
							"id",
							"transactionId",
							"transactionDate",
							"transactionTime",
							"fromAccount",
							"toAccount",
							"fromNarrative",
							"toNarrative",
							"amount",
							"status",
							"method",
							"typeTransaction",
							"description",
							"refNumber",
							"orderTime",
							"orderType",
							"currency",
							"metadata",
							"createdAt",
							"updatedAt"}).
						AddRow(
							1,
							"TRX1678947359CiBM08mvRqi0z5fD1VdQng",
							timeParse,
							timeParse,
							"1234567890",
							"0987654321",
							"from narrative",
							"to narrative",
							100000,
							"success",
							"transfer",
							"debit",
							"transfer",
							"FT2303000001",
							timeParse,
							"INV",
							models.IDRCurrency,
							"{}",
							time.Now(),
							time.Now())
					suite.mock.ExpectQuery(regexp.QuoteMeta(getByIDQuery)).WithArgs(1).WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
		{
			name: "test error result",
			args: args{
				ctx: context.TODO(),
				id:  1,
				setupMocks: func() {
					suite.mock.ExpectQuery(regexp.QuoteMeta(getByIDQuery)).WithArgs(1).WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()
			_, err := suite.repo.GetByID(tt.args.ctx, tt.args.id)
			assert.Equal(t, tt.wantErr, err != nil)
			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *TransactionTestSuite) TestRepository_GetTrxId() {
	type args struct {
		ctx        context.Context
		id         int64
		setupMocks func()
	}
	testCases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success - findTrxById",
			args: args{
				ctx: context.TODO(),
				id:  1,
				setupMocks: func() {
					timeParse, err := time.Parse(common.DateFormatYYYYMMDD, "2021-02-01")
					if err != nil {
						suite.t.Error(err)
					}
					rows := sqlmock.NewRows(
						[]string{
							"id",
							"fromAccount",
							"toAccount",
							"fromNarrative",
							"toNarrative",
							"transactionDate",
							"amount",
							"status",
							"method",
							"typeTransaction",
							"description",
							"refNumber",
							"metadata"}).
						AddRow(
							1,
							"1234567890",
							"0987654321",
							"from narrative",
							"to narrative",
							timeParse,
							100000,
							"success",
							"transfer",
							"debit",
							"transfer",
							"FT2303000001",
							"{}")
					suite.mock.ExpectQuery(regexp.QuoteMeta(findTrxById)).WithArgs(1).WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
		{
			name: "test error result",
			args: args{
				ctx: context.TODO(),
				id:  1,
				setupMocks: func() {
					suite.mock.ExpectQuery(regexp.QuoteMeta(findTrxById)).WithArgs(1).WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			_, err := suite.repo.GetTrxId(tt.args.ctx, tt.args.id)
			t.Log("errerrerrerr", err)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *TransactionTestSuite) Test_transactionRepository_CheckRefNumbers() {
	type args struct {
		ctx        context.Context
		refNumbers []string
		setupMocks func()
	}

	testCases := []struct {
		name    string
		args    args
		want    map[string]bool
		wantErr bool
	}{
		{
			name: "success - get ref numbers",
			args: args{
				ctx:        context.TODO(),
				refNumbers: []string{"123456", "654321"},
				setupMocks: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryCheckByRefNumbers)).
						WithArgs(pq.Array([]string{"123456", "654321"})).
						WillReturnRows(sqlmock.
							NewRows([]string{"refNumber"}).
							AddRow("654321"))
				},
			},
			wantErr: false,
			want: map[string]bool{
				"123456": false,
				"654321": true,
			},
		},
		{
			name: "failed - error query get ref numbers",
			args: args{
				ctx:        context.TODO(),
				refNumbers: []string{"123456", "654321"},
				setupMocks: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryCheckByRefNumbers)).
						WithArgs(pq.Array([]string{"123456", "654321"})).
						WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
			want:    nil,
		},
		{
			name: "failed - error scan values from db",
			args: args{
				ctx:        context.TODO(),
				refNumbers: []string{"123456", "654321"},
				setupMocks: func() {
					suite.mock.
						ExpectQuery(regexp.QuoteMeta(queryCheckByRefNumbers)).
						WithArgs(pq.Array([]string{"123456", "654321"})).
						WillReturnRows(sqlmock.
							NewRows([]string{"InvalidColumn"}).
							AddRow(nil))
				},
			},
			wantErr: true,
			want:    nil,
		},
	}
	for _, tt := range testCases {
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			got, err := suite.repo.CheckRefNumbers(tt.args.ctx, tt.args.refNumbers...)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want, got)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *TransactionTestSuite) Test_TransactionRepository_GetByTransactionTypeAndRefNumber() {
	date := common.Now()
	testCases := []struct {
		name       string
		setupMocks func(param *models.TransactionGetByTypeAndRefNumberRequest)
		wantErr    bool
	}{
		{
			name: "happy path",
			setupMocks: func(param *models.TransactionGetByTypeAndRefNumberRequest) {
				cols := []string{
					"id", "refNumber", "orderType", "method", "typeTransaction",
					"transactionTime", "transactionDate",
					"fromAccount", "toAccount", "amount",
					"status", "description", "metadata",
					"createdAt", "updatedAt"}

				suite.mock.ExpectQuery(regexp.QuoteMeta(queryGetByTransactionTypeAndRefNumber)).
					WithArgs(param.TransactionType, param.RefNumber).
					WillReturnRows(sqlmock.
						NewRows(cols).
						AddRow("id", "refNumber", "orderType", "method", "typeTransaction", date, date, "fromAccount", "toAccount", 1, "status", "transfer", "{}", date, date),
					)
			},
			wantErr: false,
		},
		{
			name: "failed - err db",
			setupMocks: func(param *models.TransactionGetByTypeAndRefNumberRequest) {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryGetByTransactionTypeAndRefNumber)).
					WithArgs(param.TransactionType, param.RefNumber).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed - err scan",
			setupMocks: func(param *models.TransactionGetByTypeAndRefNumberRequest) {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryGetByTransactionTypeAndRefNumber)).
					WithArgs(param.TransactionType, param.RefNumber).
					WillReturnRows(sqlmock.
						NewRows([]string{"InvalidColumn"}).
						AddRow(nil),
					)
			},
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			param := &models.TransactionGetByTypeAndRefNumberRequest{}
			tc.setupMocks(param)

			_, err := suite.repo.GetByTransactionTypeAndRefNumber(context.Background(), param)
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *TransactionTestSuite) Test_TransactionRepository_GetByTransactionID() {
	date := common.Now()
	testCases := []struct {
		name       string
		setupMocks func(trxId string)
		wantErr    bool
	}{
		{
			name: "happy path",
			setupMocks: func(trxId string) {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryTransactionByTransactionID)).
					WithArgs(trxId).
					WillReturnRows(sqlmock.NewRows(
						[]string{"id", "transactionId", "transactionDate", "fromAccount", "toAccount", "fromNarrative", "toNarrative", "amount", "status", "method", "typeTransaction", "description", "refNumber", "metadata", "createdAt", "updatedAt"}).
						AddRow(1, "TRX1678947359CiBM08mvRqi0z5fD1VdQng", date, "1234567890", "0987654321", "from narrative", "to narrative", 100000, "success", "transfer", "debit", "transfer", "FT2303000001", "{}", time.Now(), time.Now()),
					)
			},
			wantErr: false,
		},
		{
			name: "failed - err db",
			setupMocks: func(trxId string) {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryTransactionByTransactionID)).
					WithArgs(trxId).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed - err scan",
			setupMocks: func(trxId string) {
				suite.mock.ExpectQuery(regexp.QuoteMeta(queryTransactionByTransactionID)).
					WithArgs(trxId).
					WillReturnRows(sqlmock.
						NewRows([]string{"InvalidColumn"}).
						AddRow(nil),
					)
			},
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			trxId := tc.name
			tc.setupMocks(trxId)

			_, err := suite.repo.GetByTransactionID(context.Background(), trxId)
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *TransactionTestSuite) Test_TransactionRepository_UpdateStatus() {
	testCases := []struct {
		name         string
		wantErr      bool
		rowsAffected int64
		doMock       func(id uint64, status string, rowsAffected int64)
	}{
		{
			name:         "happy path",
			rowsAffected: 1,
			doMock: func(id uint64, status string, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryUpdateTransactionStatus)).
					WithArgs(status, id).
					WillReturnResult(sqlmock.NewResult(0, rowsAffected))
				suite.mock.ExpectQuery(regexp.QuoteMeta(getByIDQuery)).
					WithArgs(id).
					WillReturnRows(sqlmock.NewRows(
						[]string{"id", "transactionId", "transactionDate", "transactionTime", "fromAccount", "toAccount", "fromNarrative", "toNarrative", "amount", "status", "method", "typeTransaction", "description", "refNumber", "orderTime", "orderType", "currency", "metadata", "createdAt", "updatedAt"}).
						AddRow(1, "TRX1678947359CiBM08mvRqi0z5fD1VdQng", time.Now(), time.Now(), "1234567890", "0987654321", "from narrative", "to narrative", 100000, "success", "transfer", "debit", "transfer", "FT2303000001", time.Now(), "INV", models.IDRCurrency, "{}", time.Now(), time.Now()),
					)

			},
			wantErr: false,
		},
		{
			name:         "failed - no rows affected",
			rowsAffected: 0,
			doMock: func(id uint64, status string, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryUpdateTransactionStatus)).
					WithArgs(status, id).
					WillReturnResult(sqlmock.NewResult(0, rowsAffected))
			},
			wantErr: true,
		},
		{
			name:         "failed - err sql",
			rowsAffected: 0,
			doMock: func(id uint64, status string, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryUpdateTransactionStatus)).
					WithArgs(status, id).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			status := "status"
			var id uint64 = 1
			tc.doMock(id, status, tc.rowsAffected)

			_, err := suite.repo.UpdateStatus(context.Background(), id, status)
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
