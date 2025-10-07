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
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	mockFlag "bitbucket.org/Amartha/go-fp-transaction/internal/common/flag/mock"
)

func TestABDRepositoryTestSuite(t *testing.T) {
	t.Helper()
	suite.Run(t, new(abdTestSuite))
}

type abdTestSuite struct {
	suite.Suite
	t        *testing.T
	writeDB  *sql.DB
	readDB   *sql.DB
	mock     sqlmock.Sqlmock
	mockFlag *mockFlag.MockClient
	repo     AccountBalanceDailyRepository
}

func (suite *abdTestSuite) SetupTest() {
	var err error
	var cfg config.Config

	suite.writeDB, suite.mock, err = sqlmock.New()
	require.NoError(suite.T(), err)

	suite.readDB = suite.writeDB

	suite.t = suite.T()

	mockCtrl := gomock.NewController(suite.t)
	suite.mockFlag = mockFlag.NewMockClient(mockCtrl)

	mockAccounting := mock.NewMockClient(mockCtrl)

	suite.repo = NewSQLRepository(suite.writeDB, suite.readDB, cfg, suite.mockFlag, mockAccounting).GetAccountBalanceDailyRepository()
}

func (suite *abdTestSuite) TearDownTest() {
	defer suite.writeDB.Close()
	defer suite.readDB.Close()

}

func (suite *abdTestSuite) TestRepository_ListByDate() {
	date, _ := time.Parse("2006-01-02", "2023-01-31")
	dec := decimal.Decimal{}
	amountVal := float64(10)

	type args struct {
		ctx        context.Context
		date       time.Time
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
				date: common.Now(),
				setupMocks: func() {
					rows := sqlmock.NewRows(
						[]string{"accountNumber", "date", "balance"}).
						AddRow("1", date, amountVal)
					dec.Scan(amountVal)
					suite.mock.ExpectQuery(regexp.QuoteMeta(queryListByDate)).WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
		{
			name: "test error result",
			args: args{
				ctx:  context.TODO(),
				date: common.Now(),
				setupMocks: func() {
					suite.mock.ExpectQuery(regexp.QuoteMeta(queryListByDate)).WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			_, err := suite.repo.ListByDate(tt.args.ctx, tt.args.date)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}

func (suite *abdTestSuite) TestRepository_Create() {
	// TODO
}

func (suite *abdTestSuite) TestRepository_GetLast() {
	date, _ := time.Parse("2006-01-02", "2023-01-31")
	dec := decimal.Decimal{}
	amountVal := float64(10)

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
			name: "test success",
			args: args{
				ctx: context.Background(),
				setupMocks: func() {
					rows := sqlmock.NewRows(
						[]string{"accountNumber", "date", "balance"}).
						AddRow("1", date, amountVal)
					dec.Scan(amountVal)
					suite.mock.ExpectQuery(regexp.QuoteMeta(queryABDGetLast)).WillReturnRows(rows)
				},
			},
			wantErr: false,
		},
		{
			name: "test error result",
			args: args{
				ctx: context.TODO(),
				setupMocks: func() {
					suite.mock.ExpectQuery(regexp.QuoteMeta(queryABDGetLast)).WillReturnError(assert.AnError)
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range testCases {
		tt := tt
		suite.t.Run(tt.name, func(t *testing.T) {
			tt.args.setupMocks()

			_, err := suite.repo.GetLast(tt.args.ctx)
			assert.Equal(t, tt.wantErr, err != nil)

			if err = suite.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("there were unfulfilled expectations: %s", err)
			}
		})
	}
}
