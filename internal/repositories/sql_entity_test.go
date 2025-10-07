package repositories

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting/mock"
	mockFlag "bitbucket.org/Amartha/go-fp-transaction/internal/common/flag/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func TestEntityRepositoryTestSuite(t *testing.T) {
	t.Helper()
	suite.Run(t, new(entityTestSuite))
}

type entityTestSuite struct {
	suite.Suite
	t        *testing.T
	writeDB  *sql.DB
	readDB   *sql.DB
	mock     sqlmock.Sqlmock
	mockFlag *mockFlag.MockClient
	repo     EntityRepository
}

func (suite *entityTestSuite) SetupTest() {
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

	suite.repo = NewSQLRepository(suite.writeDB, suite.readDB, cfg, suite.mockFlag, mockAccounting).GetEntityRepository()
}

func (suite *entityTestSuite) TearDownTest() {
	defer suite.writeDB.Close()
	defer suite.readDB.Close()

}

func (suite *entityTestSuite) TestRepository_CheckEntityByCode() {
	type args struct {
		ctx        context.Context
		code       string
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
				ctx:  context.Background(),
				code: "211",
				setupMocks: func() {
					rows := sqlmock.NewRows(
						[]string{"code"}).AddRow("211")
					suite.mock.ExpectQuery(regexp.QuoteMeta(queryEntityIsExistByCode)).WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
		{
			name: "test data not found",
			args: args{
				ctx: context.TODO(),
				setupMocks: func() {
					suite.mock.ExpectQuery(regexp.QuoteMeta(queryEntityIsExistByCode)).WillReturnError(sql.ErrNoRows)
				},
			},
			wantErr: true,
		},
		{
			name: "test error result",
			args: args{
				ctx: context.TODO(),
				setupMocks: func() {
					suite.mock.ExpectQuery(regexp.QuoteMeta(queryEntityIsExistByCode)).WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			err := suite.repo.CheckEntityByCode(tt.args.ctx, tt.args.code)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *entityTestSuite) TestRepository_Create() {
	type args struct {
		ctx context.Context
		req models.CreateEntityIn
	}

	testCases := []struct {
		name    string
		args    args
		wantErr bool
		doMock  func(args args)
	}{
		{
			name: "test success",
			args: args{
				ctx: context.TODO(),
				req: models.CreateEntityIn{
					Code:        "test",
					Name:        "test",
					Description: "test",
				},
			},
			doMock: func(args args) {
				rows := sqlmock.
					NewRows([]string{"id", "code", "name", "description", "createdAt", "updatedAt"}).
					AddRow(1, args.req.Code, args.req.Name, args.req.Description, time.Now(), time.Now())

				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryEntityCreate)).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "error scan row",
			args: args{
				ctx: context.TODO(),
				req: models.CreateEntityIn{},
			},
			doMock: func(args args) {
				rows := sqlmock.NewRows([]string{"InvalidColumn"}).AddRow(nil)
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryEntityCreate)).
					WillReturnRows(rows)
			},
			wantErr: true,
		},
		{
			name: "test error db",
			args: args{
				ctx: context.TODO(),
				req: models.CreateEntityIn{},
			},
			doMock: func(args args) {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryEntityCreate)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.doMock(tt.args)

			_, err := suite.repo.Create(tt.args.ctx, &tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *entityTestSuite) TestRepository_List() {
	testCases := []struct {
		name    string
		wantErr bool
		doMock  func()
	}{
		{
			name: "success get list",
			doMock: func() {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryEntityList)).
					WillReturnRows(
						sqlmock.
							NewRows([]string{"id", "description", "code", "name", "createdAt", "updatedAt"}).
							AddRow(1, "this is description", "666", "ENT", time.Now(), time.Now()),
					)
			},
			wantErr: false,
		},
		{
			name: "failed scan row",
			doMock: func() {
				suite.mock.
					ExpectQuery(regexp.QuoteMeta(queryEntityList)).
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
					ExpectQuery(regexp.QuoteMeta(queryEntityList)).
					WillReturnError(assert.AnError)
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		suite.t.Run(tc.name, func(t *testing.T) {
			tc.doMock()

			_, err := suite.repo.List(context.Background())
			assert.Equal(t, tc.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
