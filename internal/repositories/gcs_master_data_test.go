package repositories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/shopspring/decimal"
	"go.uber.org/mock/gomock"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/safeaccess"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/safeaccess/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
)

type masterDataHelper struct {
	mockCtrl *gomock.Controller

	defaultConfig         config.MasterDataConfig
	mockConfigsVatRevenue *mock.MockObjectStorageClient[[]models.ConfigVatRevenue]
	mockOrderTypes        *mock.MockObjectStorageClient[[]models.OrderType]

	defaultClientOpts      []option.ClientOption
	defaultValueOrderType  []models.OrderType
	defaultValueVatRevenue []models.ConfigVatRevenue
}

func newMasterDataHelper(t *testing.T) *masterDataHelper {
	t.Helper()
	t.Parallel()

	mockCtrl := gomock.NewController(t)

	server, err := fakestorage.NewServerWithOptions(fakestorage.Options{
		NoListener: true,
	})
	assert.NoError(t, err)

	// init dummy orderTypes
	mdConf := config.MasterDataConfig{
		BucketName:         "DUMMY_BUCKET",
		OrderTypeFilePath:  "DUMMY_FILE_PATH.json",
		VatRevenueFilePath: "DUMMY_FILE_PATH_2.json",
	}

	currentTime := time.Now()

	return &masterDataHelper{
		mockCtrl:              mockCtrl,
		defaultConfig:         mdConf,
		mockConfigsVatRevenue: mock.NewMockObjectStorageClient[[]models.ConfigVatRevenue](mockCtrl),
		mockOrderTypes:        mock.NewMockObjectStorageClient[[]models.OrderType](mockCtrl),
		defaultValueOrderType: []models.OrderType{
			{
				OrderTypeCode: "1001",
				OrderTypeName: "Top Up Lender (P2P)",
				TransactionTypes: []models.TransactionType{
					{
						TransactionTypeCode: "1001001",
						TransactionTypeName: "Top up Lender via Mandiri",
					},
					{
						TransactionTypeCode: "1001002",
						TransactionTypeName: "Top up Lender via Permata",
					},
				},
			},
			{
				OrderTypeCode: "1002",
				OrderTypeName: "Cashout Lender P2P",
				TransactionTypes: []models.TransactionType{
					{
						TransactionTypeCode: "1002001",
						TransactionTypeName: "Request Cashout",
					},
					{
						TransactionTypeCode: "1002002",
						TransactionTypeName: "Reject Request Cashout",
					},
				},
			},
		},
		defaultClientOpts: []option.ClientOption{
			option.WithoutAuthentication(),
			option.WithHTTPClient(server.HTTPClient()),
		},
		defaultValueVatRevenue: []models.ConfigVatRevenue{
			{
				Percentage: decimal.NewFromFloat(0.11),
				StartTime:  currentTime.Add(-24 * time.Hour),
				EndTime:    currentTime.Add(24 * time.Hour),
			},
			{
				Percentage: decimal.NewFromFloat(0.12),
				StartTime:  currentTime.Add(24 * time.Hour),
				EndTime:    currentTime.Add(2 * 24 * time.Hour),
			},
		},
	}
}

func TestNewGCSMasterDataRepository(t *testing.T) {
	helper := newMasterDataHelper(t)
	defer helper.mockCtrl.Finish()

	type args struct {
		cfg  *config.Config
		opts []option.ClientOption
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success new gcs master orderTypes repository",
			args: args{
				cfg: &config.Config{
					MasterData: helper.defaultConfig,
				},
				opts: helper.defaultClientOpts,
			},
			wantErr: false,
		},
		{
			name: "failed new gcs master orderTypes repository, bucket name not set",
			args: args{
				cfg: &config.Config{
					MasterData: config.MasterDataConfig{
						BucketName:         "",
						OrderTypeFilePath:  helper.defaultConfig.OrderTypeFilePath,
						VatRevenueFilePath: helper.defaultConfig.VatRevenueFilePath,
					},
				},
				opts: helper.defaultClientOpts,
			},
			wantErr: true,
		},
		{
			name: "failed new gcs master orderTypes repository, file path order not set",
			args: args{
				cfg: &config.Config{
					MasterData: config.MasterDataConfig{
						BucketName:         helper.defaultConfig.BucketName,
						OrderTypeFilePath:  "",
						VatRevenueFilePath: helper.defaultConfig.VatRevenueFilePath,
					},
				},
				opts: helper.defaultClientOpts,
			},
			wantErr: true,
		},
		{
			name: "failed new gcs master orderTypes repository, file path vat config not set",
			args: args{
				cfg: &config.Config{
					MasterData: config.MasterDataConfig{
						BucketName:         helper.defaultConfig.BucketName,
						OrderTypeFilePath:  helper.defaultConfig.OrderTypeFilePath,
						VatRevenueFilePath: "",
					},
				},
				opts: helper.defaultClientOpts,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewGCSMasterDataRepository(tt.args.cfg, tt.args.opts...)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_gcsMasterDataRepository_GetListOrderTypeCode(t *testing.T) {
	type fields struct {
		client               *storage.Client
		configsVatRevenue    safeaccess.ObjectStorageClient[[]models.ConfigVatRevenue]
		orderTypes           safeaccess.ObjectStorageClient[[]models.OrderType]
		orderTypeCodes       []string
		transactionTypeCodes []string
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success get list order type codes",
			fields: fields{
				orderTypeCodes: []string{"TUP", "INV"},
			},
			args: args{
				ctx: context.TODO(),
			},
			want:    []string{"TUP", "INV"},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &gcsMasterDataRepository{
				client:               tt.fields.client,
				configsVatRevenue:    tt.fields.configsVatRevenue,
				orderTypes:           tt.fields.orderTypes,
				orderTypeCodes:       tt.fields.orderTypeCodes,
				transactionTypeCodes: tt.fields.transactionTypeCodes,
			}
			got, err := g.GetListOrderTypeCode(tt.args.ctx)
			if !tt.wantErr(t, err, fmt.Sprintf("GetListOrderTypeCode(%v)", tt.args.ctx)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetListOrderTypeCode(%v)", tt.args.ctx)
		})
	}
}

func Test_gcsMasterDataRepository_GetListTransactionTypeCode(t *testing.T) {
	type fields struct {
		client               *storage.Client
		configsVatRevenue    safeaccess.ObjectStorageClient[[]models.ConfigVatRevenue]
		orderTypes           safeaccess.ObjectStorageClient[[]models.OrderType]
		orderTypeCodes       []string
		transactionTypeCodes []string
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success get list transaction type codes",
			fields: fields{
				transactionTypeCodes: []string{"TUPVA", "INVMT"},
			},
			args: args{
				ctx: context.TODO(),
			},
			want:    []string{"TUPVA", "INVMT"},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &gcsMasterDataRepository{
				client:               tt.fields.client,
				configsVatRevenue:    tt.fields.configsVatRevenue,
				orderTypes:           tt.fields.orderTypes,
				orderTypeCodes:       tt.fields.orderTypeCodes,
				transactionTypeCodes: tt.fields.transactionTypeCodes,
			}
			got, err := g.GetListTransactionTypeCode(tt.args.ctx)
			if !tt.wantErr(t, err, fmt.Sprintf("GetListTransactionTypeCode(%v)", tt.args.ctx)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetListTransactionTypeCode(%v)", tt.args.ctx)
		})
	}
}

func Test_gcsMasterDataRepository_UpsertOrderType(t *testing.T) {
	helper := newMasterDataHelper(t)
	defer helper.mockCtrl.Finish()

	type fields struct {
		client               *storage.Client
		configsVatRevenue    safeaccess.ObjectStorageClient[[]models.ConfigVatRevenue]
		orderTypes           safeaccess.ObjectStorageClient[[]models.OrderType]
		orderTypeCodes       []string
		transactionTypeCodes []string
	}
	type args struct {
		ctx       context.Context
		orderType models.OrderType
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		doMocks func(m *masterDataHelper)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success upsert",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx:       context.TODO(),
				orderType: models.OrderType{},
			},
			doMocks: func(m *masterDataHelper) {
				m.mockOrderTypes.
					EXPECT().
					Value().
					Return(safeaccess.New([]models.OrderType{
						{
							OrderTypeCode: "TUP",
						},
					})).
					Times(2)
				m.mockOrderTypes.
					EXPECT().
					UpdateFile(gomock.Any()).
					Return(nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "failed update file",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx:       context.TODO(),
				orderType: models.OrderType{},
			},
			doMocks: func(m *masterDataHelper) {
				m.mockOrderTypes.
					EXPECT().
					Value().
					Return(safeaccess.New([]models.OrderType{
						{
							OrderTypeCode: "TUP",
						},
					})).
					Times(2)
				m.mockOrderTypes.
					EXPECT().
					UpdateFile(gomock.Any()).
					Return(assert.AnError)
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMocks != nil {
				tt.doMocks(helper)
			}

			g := &gcsMasterDataRepository{
				client:               tt.fields.client,
				configsVatRevenue:    tt.fields.configsVatRevenue,
				orderTypes:           tt.fields.orderTypes,
				orderTypeCodes:       tt.fields.orderTypeCodes,
				transactionTypeCodes: tt.fields.transactionTypeCodes,
			}
			tt.wantErr(t, g.UpsertOrderType(tt.args.ctx, tt.args.orderType), fmt.Sprintf("UpsertOrderType(%v, %v)", tt.args.ctx, tt.args.orderType))
		})
	}
}

func Test_gcsMasterDataRepository_RefreshDataPeriodically(t *testing.T) {
	helper := newMasterDataHelper(t)
	defer helper.mockCtrl.Finish()

	type fields struct {
		client            *storage.Client
		configsVatRevenue safeaccess.ObjectStorageClient[[]models.ConfigVatRevenue]
		orderTypes        safeaccess.ObjectStorageClient[[]models.OrderType]
	}

	tests := []struct {
		name   string
		fields fields
		doMock func(fields fields)
	}{
		{
			name: "success refresh orderTypes periodically",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			doMock: func(fields fields) {
				// mock should match with expectation
				helper.mockConfigsVatRevenue.
					EXPECT().
					LoadFile(gomock.Any()).
					Return(nil).
					AnyTimes()
				helper.mockOrderTypes.
					EXPECT().
					LoadFile(gomock.Any()).
					Return(nil).
					AnyTimes()
				helper.mockConfigsVatRevenue.
					EXPECT().
					Value().
					Return(safeaccess.New(helper.defaultValueVatRevenue)).
					AnyTimes()
				helper.mockOrderTypes.
					EXPECT().
					Value().
					Return(safeaccess.New(helper.defaultValueOrderType)).
					AnyTimes()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.fields)
			}

			g := &gcsMasterDataRepository{
				client:            tt.fields.client,
				configsVatRevenue: tt.fields.configsVatRevenue,
				orderTypes:        tt.fields.orderTypes,
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
			g.RefreshDataPeriodically(ctx, 100*time.Millisecond)

			time.Sleep(time.Second)
			cancel()
		})
	}
}

func Test_gcsMasterDataRepository_GetListOrderType(t *testing.T) {
	helper := newMasterDataHelper(t)
	defer helper.mockCtrl.Finish()

	type fields struct {
		client               *storage.Client
		configsVatRevenue    safeaccess.ObjectStorageClient[[]models.ConfigVatRevenue]
		orderTypes           safeaccess.ObjectStorageClient[[]models.OrderType]
		orderTypeCodes       []string
		transactionTypeCodes []string
	}
	type args struct {
		ctx    context.Context
		filter models.FilterMasterData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		doMocks func(m *masterDataHelper)
		want    []models.OrderType
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success get order types",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx:    context.TODO(),
				filter: models.FilterMasterData{},
			},
			doMocks: func(m *masterDataHelper) {
				m.mockOrderTypes.
					EXPECT().
					Value().
					Return(safeaccess.New(helper.defaultValueOrderType))
			},
			want:    helper.defaultValueOrderType,
			wantErr: assert.NoError,
		},
		{
			name: "filter based on code",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx: context.TODO(),
				filter: models.FilterMasterData{
					Code: "1001",
				},
			},
			doMocks: func(m *masterDataHelper) {
				m.mockOrderTypes.
					EXPECT().
					Value().
					Return(safeaccess.New(helper.defaultValueOrderType))
			},
			want: []models.OrderType{
				{
					OrderTypeCode: "1001",
					OrderTypeName: "Top Up Lender (P2P)",
					TransactionTypes: []models.TransactionType{
						{
							TransactionTypeCode: "1001001",
							TransactionTypeName: "Top up Lender via Mandiri",
						},
						{
							TransactionTypeCode: "1001002",
							TransactionTypeName: "Top up Lender via Permata",
						},
					},
				},
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMocks != nil {
				tt.doMocks(helper)
			}

			g := &gcsMasterDataRepository{
				client:               tt.fields.client,
				configsVatRevenue:    tt.fields.configsVatRevenue,
				orderTypes:           tt.fields.orderTypes,
				orderTypeCodes:       tt.fields.orderTypeCodes,
				transactionTypeCodes: tt.fields.transactionTypeCodes,
			}
			got, err := g.GetListOrderType(tt.args.ctx, tt.args.filter)
			if !tt.wantErr(t, err, fmt.Sprintf("GetListOrderType(%v, %v)", tt.args.ctx, tt.args.filter)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetListOrderType(%v, %v)", tt.args.ctx, tt.args.filter)
		})
	}
}

func Test_gcsMasterDataRepository_GetOrderType(t *testing.T) {
	helper := newMasterDataHelper(t)
	defer helper.mockCtrl.Finish()

	type fields struct {
		client               *storage.Client
		configsVatRevenue    safeaccess.ObjectStorageClient[[]models.ConfigVatRevenue]
		orderTypes           safeaccess.ObjectStorageClient[[]models.OrderType]
		orderTypeCodes       []string
		transactionTypeCodes []string
	}
	type args struct {
		ctx           context.Context
		orderTypeCode string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		doMocks func(m *masterDataHelper)
		want    *models.OrderType
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success get order type",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx:           context.TODO(),
				orderTypeCode: "1001",
			},
			doMocks: func(m *masterDataHelper) {
				m.mockOrderTypes.
					EXPECT().
					Value().
					Return(safeaccess.New(helper.defaultValueOrderType))
			},
			want:    &helper.defaultValueOrderType[0],
			wantErr: assert.NoError,
		},
		{
			name: "order type not found",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx:           context.TODO(),
				orderTypeCode: "CODE_THAT_SHOULD_NOT_BE_EXISTS",
			},
			doMocks: func(m *masterDataHelper) {
				m.mockOrderTypes.
					EXPECT().
					Value().
					Return(safeaccess.New(helper.defaultValueOrderType))
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMocks != nil {
				tt.doMocks(helper)
			}

			g := &gcsMasterDataRepository{
				client:               tt.fields.client,
				configsVatRevenue:    tt.fields.configsVatRevenue,
				orderTypes:           tt.fields.orderTypes,
				orderTypeCodes:       tt.fields.orderTypeCodes,
				transactionTypeCodes: tt.fields.transactionTypeCodes,
			}
			got, err := g.GetOrderType(tt.args.ctx, tt.args.orderTypeCode)
			if !tt.wantErr(t, err, fmt.Sprintf("GetOrderType(%v, %v)", tt.args.ctx, tt.args.orderTypeCode)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetOrderType(%v, %v)", tt.args.ctx, tt.args.orderTypeCode)
		})
	}
}

func Test_gcsMasterDataRepository_GetListTransactionType(t *testing.T) {
	helper := newMasterDataHelper(t)
	defer helper.mockCtrl.Finish()

	type fields struct {
		client               *storage.Client
		configsVatRevenue    safeaccess.ObjectStorageClient[[]models.ConfigVatRevenue]
		orderTypes           safeaccess.ObjectStorageClient[[]models.OrderType]
		orderTypeCodes       []string
		transactionTypeCodes []string
	}
	type args struct {
		ctx    context.Context
		filter models.FilterMasterData
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		doMocks func(m *masterDataHelper)
		want    []models.TransactionType
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success get list transaction type",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx: context.TODO(),
				filter: models.FilterMasterData{
					Code: helper.defaultValueOrderType[0].TransactionTypes[0].TransactionTypeCode,
				},
			},
			doMocks: func(m *masterDataHelper) {
				m.mockOrderTypes.
					EXPECT().
					Value().
					Return(safeaccess.New(helper.defaultValueOrderType))
			},
			want: []models.TransactionType{
				helper.defaultValueOrderType[0].TransactionTypes[0],
			},
			wantErr: assert.NoError,
		},
		{
			name: "empty transaction type",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx: context.TODO(),
				filter: models.FilterMasterData{
					Code: "CODE_THAT_SHOULD_NOT_BE_EXISTS",
				},
			},
			doMocks: func(m *masterDataHelper) {
				m.mockOrderTypes.
					EXPECT().
					Value().
					Return(safeaccess.New(helper.defaultValueOrderType))
			},
			want:    []models.TransactionType(nil),
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMocks != nil {
				tt.doMocks(helper)
			}

			g := &gcsMasterDataRepository{
				client:               tt.fields.client,
				configsVatRevenue:    tt.fields.configsVatRevenue,
				orderTypes:           tt.fields.orderTypes,
				orderTypeCodes:       tt.fields.orderTypeCodes,
				transactionTypeCodes: tt.fields.transactionTypeCodes,
			}
			got, err := g.GetListTransactionType(tt.args.ctx, tt.args.filter)
			if !tt.wantErr(t, err, fmt.Sprintf("GetListTransactionType(%v, %v)", tt.args.ctx, tt.args.filter)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetListTransactionType(%v, %v)", tt.args.ctx, tt.args.filter)
		})
	}
}

func Test_gcsMasterDataRepository_repopulate(t *testing.T) {
	helper := newMasterDataHelper(t)
	defer helper.mockCtrl.Finish()

	type fields struct {
		client               *storage.Client
		configsVatRevenue    safeaccess.ObjectStorageClient[[]models.ConfigVatRevenue]
		orderTypes           safeaccess.ObjectStorageClient[[]models.OrderType]
		orderTypeCodes       []string
		transactionTypeCodes []string
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		doMocks func(m *masterDataHelper)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success repopulate",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx: context.TODO(),
			},
			doMocks: func(m *masterDataHelper) {
				m.mockOrderTypes.
					EXPECT().
					LoadFile(gomock.Any()).
					Return(nil)
				m.mockConfigsVatRevenue.
					EXPECT().
					LoadFile(gomock.Any()).
					Return(nil)
				m.mockOrderTypes.
					EXPECT().
					Value().
					Return(safeaccess.New(m.defaultValueOrderType))
			},
			wantErr: assert.NoError,
		},
		{
			name: "failed load order types",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx: context.TODO(),
			},
			doMocks: func(m *masterDataHelper) {
				m.mockOrderTypes.
					EXPECT().
					LoadFile(gomock.Any()).
					Return(assert.AnError)
			},
			wantErr: assert.Error,
		},
		{
			name: "failed load config vat",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx: context.TODO(),
			},
			doMocks: func(m *masterDataHelper) {
				m.mockOrderTypes.
					EXPECT().
					LoadFile(gomock.Any()).
					Return(nil)
				m.mockConfigsVatRevenue.
					EXPECT().
					LoadFile(gomock.Any()).
					Return(assert.AnError)
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMocks != nil {
				tt.doMocks(helper)
			}

			g := &gcsMasterDataRepository{
				client:               tt.fields.client,
				configsVatRevenue:    tt.fields.configsVatRevenue,
				orderTypes:           tt.fields.orderTypes,
				orderTypeCodes:       tt.fields.orderTypeCodes,
				transactionTypeCodes: tt.fields.transactionTypeCodes,
			}
			tt.wantErr(t, g.repopulate(tt.args.ctx), fmt.Sprintf("repopulate(%v)", tt.args.ctx))
		})
	}
}

func Test_gcsMasterDataRepository_GetTransactionType(t *testing.T) {
	helper := newMasterDataHelper(t)
	defer helper.mockCtrl.Finish()

	type fields struct {
		client               *storage.Client
		configsVatRevenue    safeaccess.ObjectStorageClient[[]models.ConfigVatRevenue]
		orderTypes           safeaccess.ObjectStorageClient[[]models.OrderType]
		orderTypeCodes       []string
		transactionTypeCodes []string
	}
	type args struct {
		ctx                 context.Context
		transactionTypeCode string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		doMocks func(m *masterDataHelper)
		want    *models.TransactionType
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success get transaction type",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx:                 context.TODO(),
				transactionTypeCode: helper.defaultValueOrderType[0].TransactionTypes[0].TransactionTypeCode,
			},
			doMocks: func(m *masterDataHelper) {
				m.mockOrderTypes.
					EXPECT().
					Value().
					Return(safeaccess.New(m.defaultValueOrderType))
			},
			want:    &helper.defaultValueOrderType[0].TransactionTypes[0],
			wantErr: assert.NoError,
		},
		{
			name: "failed get transaction type",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx:                 context.TODO(),
				transactionTypeCode: "INVALID_TRANSACTION_TYPE_CODE",
			},
			doMocks: func(m *masterDataHelper) {
				m.mockOrderTypes.
					EXPECT().
					Value().
					Return(safeaccess.New(m.defaultValueOrderType))
			},
			want:    nil,
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMocks != nil {
				tt.doMocks(helper)
			}

			g := &gcsMasterDataRepository{
				client:               tt.fields.client,
				configsVatRevenue:    tt.fields.configsVatRevenue,
				orderTypes:           tt.fields.orderTypes,
				orderTypeCodes:       tt.fields.orderTypeCodes,
				transactionTypeCodes: tt.fields.transactionTypeCodes,
			}
			got, err := g.GetTransactionType(tt.args.ctx, tt.args.transactionTypeCode)
			if !tt.wantErr(t, err, fmt.Sprintf("GetTransactionType(%v, %v)", tt.args.ctx, tt.args.transactionTypeCode)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetTransactionType(%v, %v)", tt.args.ctx, tt.args.transactionTypeCode)
		})
	}
}

func Test_gcsMasterDataRepository_GetConfigVATRevenue(t *testing.T) {
	helper := newMasterDataHelper(t)
	defer helper.mockCtrl.Finish()

	type fields struct {
		client               *storage.Client
		configsVatRevenue    safeaccess.ObjectStorageClient[[]models.ConfigVatRevenue]
		orderTypes           safeaccess.ObjectStorageClient[[]models.OrderType]
		orderTypeCodes       []string
		transactionTypeCodes []string
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		doMocks func(m *masterDataHelper)
		want    []models.ConfigVatRevenue
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success get data config",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx: context.TODO(),
			},
			doMocks: func(m *masterDataHelper) {
				m.mockConfigsVatRevenue.
					EXPECT().
					Value().
					Return(safeaccess.New(m.defaultValueVatRevenue))
			},
			want:    helper.defaultValueVatRevenue,
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMocks != nil {
				tt.doMocks(helper)
			}

			g := &gcsMasterDataRepository{
				client:               tt.fields.client,
				configsVatRevenue:    tt.fields.configsVatRevenue,
				orderTypes:           tt.fields.orderTypes,
				orderTypeCodes:       tt.fields.orderTypeCodes,
				transactionTypeCodes: tt.fields.transactionTypeCodes,
			}
			got, err := g.GetConfigVATRevenue(tt.args.ctx)
			if !tt.wantErr(t, err, fmt.Sprintf("GetConfigVATRevenue(%v)", tt.args.ctx)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetConfigVATRevenue(%v)", tt.args.ctx)
		})
	}
}

func Test_gcsMasterDataRepository_UpsertConfigVATRevenue(t *testing.T) {
	helper := newMasterDataHelper(t)
	defer helper.mockCtrl.Finish()

	type fields struct {
		client               *storage.Client
		configsVatRevenue    safeaccess.ObjectStorageClient[[]models.ConfigVatRevenue]
		orderTypes           safeaccess.ObjectStorageClient[[]models.OrderType]
		orderTypeCodes       []string
		transactionTypeCodes []string
	}
	type args struct {
		ctx        context.Context
		vatRevenue []models.ConfigVatRevenue
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		doMocks func(m *masterDataHelper)
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "success update data",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx:        context.TODO(),
				vatRevenue: helper.defaultValueVatRevenue,
			},
			doMocks: func(m *masterDataHelper) {
				m.mockConfigsVatRevenue.
					EXPECT().
					Value().
					Return(safeaccess.New(m.defaultValueVatRevenue))
				m.mockConfigsVatRevenue.
					EXPECT().
					UpdateFile(gomock.Any()).
					Return(nil)
			},
			wantErr: assert.NoError,
		},
		{
			name: "failed update data",
			fields: fields{
				configsVatRevenue: helper.mockConfigsVatRevenue,
				orderTypes:        helper.mockOrderTypes,
			},
			args: args{
				ctx:        context.TODO(),
				vatRevenue: helper.defaultValueVatRevenue,
			},
			doMocks: func(m *masterDataHelper) {
				m.mockConfigsVatRevenue.
					EXPECT().
					Value().
					Return(safeaccess.New(m.defaultValueVatRevenue))
				m.mockConfigsVatRevenue.
					EXPECT().
					UpdateFile(gomock.Any()).
					Return(assert.AnError)
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMocks != nil {
				tt.doMocks(helper)
			}

			g := &gcsMasterDataRepository{
				client:               tt.fields.client,
				configsVatRevenue:    tt.fields.configsVatRevenue,
				orderTypes:           tt.fields.orderTypes,
				orderTypeCodes:       tt.fields.orderTypeCodes,
				transactionTypeCodes: tt.fields.transactionTypeCodes,
			}
			tt.wantErr(t, g.UpsertConfigVATRevenue(tt.args.ctx, tt.args.vatRevenue), fmt.Sprintf("UpsertConfigVATRevenue(%v, %v)", tt.args.ctx, tt.args.vatRevenue))
		})
	}
}
