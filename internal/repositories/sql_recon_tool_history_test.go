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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestReconToolHistoryRepositoryTestSuite(t *testing.T) {
	t.Helper()
	suite.Run(t, new(reconToolHistoryRepoTestSuite))
}

type reconToolHistoryRepoTestSuite struct {
	suite.Suite
	t        *testing.T
	writeDB  *sql.DB
	readDB   *sql.DB
	mock     sqlmock.Sqlmock
	mockFlag *mockFlag.MockClient
	repo     ReconToolHistoryRepository
}

func (suite *reconToolHistoryRepoTestSuite) SetupTest() {
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

	suite.repo = NewSQLRepository(suite.writeDB, suite.readDB, cfg, suite.mockFlag, mockAccounting).GetReconToolHistoryRepository()
}

func (suite *reconToolHistoryRepoTestSuite) TearDownTest() {
	defer suite.writeDB.Close()
	defer suite.readDB.Close()

}

func (suite *reconToolHistoryRepoTestSuite) TestRepository_Create() {
	testCases := []struct {
		name    string
		in      models.CreateReconToolHistoryIn
		wantErr bool
		doMock  func(args models.CreateReconToolHistoryIn)
	}{
		{
			name: "happy path",
			in: models.CreateReconToolHistoryIn{
				TransactionDate: "2023-01-01",
			},
			doMock: func(in models.CreateReconToolHistoryIn) {
				txDate, err := common.ParseStringToDatetime(common.DateFormatYYYYMMDD, in.TransactionDate)
				assert.NoError(suite.t, err)
				rows := sqlmock.
					NewRows([]string{"id", "transactionDate", "createdAt", "updatedAt"}).
					AddRow(1, txDate, time.Now(), time.Now())

				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryReconToolHistoryCreate)).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "error scan row",
			in:   models.CreateReconToolHistoryIn{},
			doMock: func(args models.CreateReconToolHistoryIn) {
				rows := sqlmock.NewRows([]string{"InvalidColumn"}).AddRow(nil)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryReconToolHistoryCreate)).
					WillReturnRows(rows)
			},
			wantErr: true,
		},
		{
			name: "error db",
			in:   models.CreateReconToolHistoryIn{},
			doMock: func(args models.CreateReconToolHistoryIn) {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryReconToolHistoryCreate)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			tc.doMock(tc.in)

			_, err := suite.repo.Create(context.Background(), &tc.in)
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *reconToolHistoryRepoTestSuite) TestRepository_DeleteByID() {
	testCases := []struct {
		name         string
		wantErr      bool
		id           string
		rowsAffected int64
		doMock       func(id string, rowsAffected int64)
	}{
		{
			name:         "happy path",
			id:           "1",
			rowsAffected: 1,
			doMock: func(id string, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryReconToolHistoryDeleteByID)).
					WithArgs(id).
					WillReturnResult(sqlmock.NewResult(0, rowsAffected))
			},
			wantErr: false,
		},
		{
			name:         "success - no rows affected",
			id:           "1",
			rowsAffected: 0,
			doMock: func(id string, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryReconToolHistoryDeleteByID)).
					WithArgs(id).
					WillReturnResult(sqlmock.NewResult(0, rowsAffected))
			},
			wantErr: false,
		},
		{
			name: "failed - err db",
			doMock: func(id string, rowsAffected int64) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryReconToolHistoryDeleteByID)).
					WithArgs(id).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			tc.doMock(tc.id, tc.rowsAffected)

			err := suite.repo.DeleteByID(context.Background(), tc.id)
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *reconToolHistoryRepoTestSuite) TestRepository_GetList() {
	testCases := []struct {
		name    string
		args    models.ReconToolHistoryFilterOptions
		wantErr bool
		doMock  func(args models.ReconToolHistoryFilterOptions)
	}{
		{
			name: "success get list",
			doMock: func(args models.ReconToolHistoryFilterOptions) {
				query, _, _ := buildListReconToolHistoryQuery(args)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(query)).
					WillReturnRows(
						sqlmock.
							NewRows([]string{
								`"id"`,
								`"orderType"`,
								`"transactionType"`,
								`"transactionDate"`,
								`"resultFilePath"`,
								`"uploadedFilePath"`,
								`"status"`,
								`"createdAt"`,
								`"updatedAt"`,
							}).
							AddRow(1, "TOPUP", "TOPUP", time.Now(), "my_file1.txt", "my_file2.txt", "SUCCESS", time.Now(), time.Now()),
					)
			},
			wantErr: false,
		},
		{
			name: "failed scan row",
			doMock: func(args models.ReconToolHistoryFilterOptions) {
				query, _, _ := buildListReconToolHistoryQuery(args)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(query)).
					WillReturnRows(
						sqlmock.NewRows([]string{"InvalidColumn"}).AddRow(nil),
					)
			},
			wantErr: true,
		},
		{
			name: "failed from db",
			doMock: func(args models.ReconToolHistoryFilterOptions) {
				query, _, _ := buildListReconToolHistoryQuery(args)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(query)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			tc.doMock(tc.args)

			_, err := suite.repo.GetList(context.Background(), tc.args)
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *reconToolHistoryRepoTestSuite) TestRepository_CountAll() {
	testCases := []struct {
		name    string
		args    models.ReconToolHistoryFilterOptions
		wantErr bool
		doMock  func(args models.ReconToolHistoryFilterOptions)
	}{
		{
			name: "success get count",
			doMock: func(args models.ReconToolHistoryFilterOptions) {
				query, _, _ := buildCountReconToolHistoryQuery(args)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(query)).
					WillReturnRows(
						sqlmock.
							NewRows([]string{`"count"`}).
							AddRow(1),
					)
			},
			wantErr: false,
		},
		{
			name: "failed scan row",
			doMock: func(args models.ReconToolHistoryFilterOptions) {
				query, _, _ := buildCountReconToolHistoryQuery(args)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(query)).
					WillReturnRows(
						sqlmock.NewRows([]string{"InvalidColumn"}).AddRow(nil),
					)
			},
			wantErr: true,
		},
		{
			name: "failed from db",
			doMock: func(args models.ReconToolHistoryFilterOptions) {
				query, _, _ := buildCountReconToolHistoryQuery(args)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(query)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			tc.doMock(tc.args)

			_, err := suite.repo.CountAll(context.Background(), tc.args)
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *reconToolHistoryRepoTestSuite) TestRepository_GetById() {
	testCases := []struct {
		name    string
		args    uint64
		wantErr bool
		doMock  func()
	}{
		{
			name: "success get by id",
			doMock: func() {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryReconToolHistoryGetById)).
					WillReturnRows(
						sqlmock.
							NewRows([]string{
								`"id"`,
								`"orderType"`,
								`"transactionType"`,
								`"transactionDate"`,
								`"resultFilePath"`,
								`"uploadedFilePath"`,
								`"status"`,
								`"createdAt"`,
								`"updatedAt"`,
							}).
							AddRow(1, "TOPUP", "TOPUP", time.Now(), "my_file1.txt", "my_file2.txt", "SUCCESS", time.Now(), time.Now()),
					)
			},
			wantErr: false,
		},
		{
			name: "failed scan row",
			doMock: func() {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryReconToolHistoryGetById)).
					WillReturnRows(
						sqlmock.NewRows([]string{"InvalidColumn"}).AddRow(nil),
					)
			},
			wantErr: true,
		},
		{
			name: "failed from db",
			doMock: func() {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryReconToolHistoryGetById)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			tc.doMock()

			_, err := suite.repo.GetById(context.Background(), tc.args)
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *reconToolHistoryRepoTestSuite) TestRepository_Update() {
	type args struct {
		id uint64
		in *models.ReconToolHistory
	}

	ct := time.Now()

	testCases := []struct {
		name    string
		wantErr bool
		args    args
		doMock  func(args args)
	}{
		{
			name: "success update",
			args: args{
				id: 1,
				in: &models.ReconToolHistory{
					ID:               1,
					OrderType:        "TEST_1",
					TransactionType:  "TEST_2",
					TransactionDate:  &ct,
					ResultFilePath:   "TEST_3.txt",
					UploadedFilePath: "TEST_4.txt",
					Status:           "SUCCESS",
				},
			},
			doMock: func(args args) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryReconToolHistoryUpdate)).
					WithArgs(
						args.id,
						args.in.OrderType,
						args.in.TransactionType,
						args.in.TransactionDate,
						args.in.UploadedFilePath,
						args.in.ResultFilePath,
						args.in.Status,
					).
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
			wantErr: false,
		},
		{
			name: "failed - err db",
			args: args{
				id: 1,
				in: &models.ReconToolHistory{
					ID:               1,
					OrderType:        "TEST_1",
					TransactionType:  "TEST_2",
					TransactionDate:  &ct,
					ResultFilePath:   "TEST_3.txt",
					UploadedFilePath: "TEST_4.txt",
					Status:           "SUCCESS",
				},
			},
			doMock: func(args args) {
				suite.mock.
					ExpectExec(regexp.QuoteMeta(queryReconToolHistoryUpdate)).
					WithArgs(
						args.id,
						args.in.OrderType,
						args.in.TransactionType,
						args.in.TransactionDate,
						args.in.UploadedFilePath,
						args.in.ResultFilePath,
						args.in.Status,
					).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			tc.doMock(tc.args)

			_, err := suite.repo.Update(context.Background(), tc.args.id, tc.args.in)
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
