package repositories

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting/mock"
	mockFlag "bitbucket.org/Amartha/go-fp-transaction/internal/common/flag/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

func TestSubCategoryRepositoryTestSuite(t *testing.T) {
	t.Helper()
	suite.Run(t, new(subCategoryTestSuite))
}

type subCategoryTestSuite struct {
	suite.Suite
	t        *testing.T
	writeDB  *sql.DB
	readDB   *sql.DB
	mock     sqlmock.Sqlmock
	mockFlag *mockFlag.MockClient
	repo     SubCategoryRepository
}

func (suite *subCategoryTestSuite) SetupTest() {
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

	suite.repo = NewSQLRepository(suite.writeDB, suite.readDB, cfg, suite.mockFlag, mockAccounting).GetSubCategoryRepository()
}

func (suite *subCategoryTestSuite) TearDownTest() {
	defer suite.writeDB.Close()
	defer suite.readDB.Close()

}

func (suite *subCategoryTestSuite) TestRepository_CheckCategoryByCode() {
	type args struct {
		ctx                   context.Context
		code, subCategoryCode string
		setupMocks            func()
	}

	testCases := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "test success",
			args: args{
				ctx:             context.Background(),
				code:            "211",
				subCategoryCode: "100",
				setupMocks: func() {
					rows := sqlmock.NewRows(
						[]string{"code"}).AddRow("100")
					suite.mock.ExpectQuery(regexp.QuoteMeta(querySubCategoryIsExistByCode)).WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
		{
			name: "test data not found",
			args: args{
				ctx: context.TODO(),
				setupMocks: func() {
					suite.mock.ExpectQuery(regexp.QuoteMeta(querySubCategoryIsExistByCode)).WillReturnError(sql.ErrNoRows)
				},
			},
			wantErr: true,
		},
		{
			name: "test error result",
			args: args{
				ctx: context.TODO(),
				setupMocks: func() {
					suite.mock.ExpectQuery(regexp.QuoteMeta(querySubCategoryIsExistByCode)).WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			err := suite.repo.CheckSubCategoryByCodeAndCategoryCode(tt.args.ctx, tt.args.code, tt.args.subCategoryCode)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *subCategoryTestSuite) TestRepository_GetByCode() {
	testCases := []struct {
		name           string
		code           string
		dbExpectations func(mock sqlmock.Sqlmock)
		expectedResult *models.SubCategory
		expectedError  error
	}{
		{
			name: "Valid Code",
			code: "valid_code",
			dbExpectations: func(mock sqlmock.Sqlmock) {
				// Expect a SELECT query and return a single row
				mock.ExpectQuery("SELECT").
					WithArgs("valid_code").
					WillReturnRows(sqlmock.NewRows([]string{"id", "categoryCode", "description", "code", "name", "createdAt", "updatedAt"}).
						AddRow(1, "cat1", "desc1", "valid_code", "name1", nil, nil))
			},
			expectedResult: &models.SubCategory{
				ID:           1,
				CategoryCode: "cat1",
				Description:  "desc1",
				Code:         "valid_code",
				Name:         "name1",
			},
			expectedError: nil,
		},
		{
			name: "No Rows Found",
			code: "nonexistent_code",
			dbExpectations: func(mock sqlmock.Sqlmock) {
				// Expect a SELECT query with no rows returned
				mock.ExpectQuery("SELECT").
					WithArgs("nonexistent_code").
					WillReturnError(sql.ErrNoRows)
			},
			expectedResult: nil,
			expectedError:  nil,
		},
		{
			name: "Database Error",
			code: "db_error_code",
			dbExpectations: func(mock sqlmock.Sqlmock) {
				// Expect a SELECT query and return a database error
				mock.ExpectQuery("SELECT").
					WithArgs("db_error_code").
					WillReturnError(assert.AnError)
			},
			expectedResult: nil,
			expectedError:  assert.AnError,
		},
	}
	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			tc.dbExpectations(suite.mock)

			subCat, err := suite.repo.GetByCode(context.Background(), tc.code)
			assert.Equal(t, tc.expectedError, err)
			assert.Equal(t, tc.expectedResult, subCat)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *subCategoryTestSuite) TestRepository_Create() {
	testCases := []struct {
		name    string
		in      models.CreateSubCategory
		wantErr bool
		doMock  func(args models.CreateSubCategory)
	}{
		{
			name: "happy path",
			in: models.CreateSubCategory{
				Code:        "test",
				Name:        "test",
				Description: "test",
			},
			doMock: func(in models.CreateSubCategory) {
				rows := sqlmock.
					NewRows([]string{"id", "categoryCode", "code", "name", "description", "createdAt", "updatedAt"}).
					AddRow(1, in.Code, in.CategoryCode, in.Name, in.Description, time.Now(), time.Now())

				suite.mock.
					ExpectQuery(regexp.QuoteMeta(querySubCategoryCreate)).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "error scan row",
			in:   models.CreateSubCategory{},
			doMock: func(args models.CreateSubCategory) {
				rows := sqlmock.NewRows([]string{"InvalidColumn"}).AddRow(nil)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(querySubCategoryCreate)).
					WillReturnRows(rows)
			},
			wantErr: true,
		},
		{
			name: "error db",
			in:   models.CreateSubCategory{},
			doMock: func(args models.CreateSubCategory) {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(querySubCategoryCreate)).
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

func (suite *subCategoryTestSuite) TestRepository_GetAll() {
	testCases := []struct {
		name    string
		wantErr bool
		doMock  func()
	}{
		{
			name: "success get all",
			doMock: func() {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryGetAllSubCategory)).
					WillReturnRows(
						sqlmock.
							NewRows([]string{"id", "categoryCode", "code", "name", "description", "createdAt", "updatedAt"}).
							AddRow(1, "221", "100000", "ENT", "this is description", time.Now(), time.Now()),
					)
			},
			wantErr: false,
		},
		{
			name: "failed scan row",
			doMock: func() {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryGetAllSubCategory)).
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
					ExpectQuery(regexp.QuoteMeta(queryGetAllSubCategory)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			tc.doMock()
			_, err := suite.repo.GetAll(context.Background())
			assert.Equal(t, tc.wantErr, err != nil)
			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
