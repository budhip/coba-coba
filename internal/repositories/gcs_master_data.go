package repositories

import (
	"context"
	"fmt"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/safeaccess"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"

	xlog "bitbucket.org/Amartha/go-x/log"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
)

type MasterDataRepository interface {
	GetListOrderType(ctx context.Context, filter models.FilterMasterData) ([]models.OrderType, error)
	GetOrderType(ctx context.Context, orderTypeCode string) (*models.OrderType, error)
	GetListTransactionType(ctx context.Context, filter models.FilterMasterData) ([]models.TransactionType, error)
	GetTransactionType(ctx context.Context, transactionTypeCode string) (*models.TransactionType, error)
	UpsertOrderType(ctx context.Context, orderType models.OrderType) error
	GetListOrderTypeCode(ctx context.Context) ([]string, error)
	GetListTransactionTypeCode(ctx context.Context) ([]string, error)
	RefreshDataPeriodically(ctx context.Context, interval time.Duration)

	// GetConfigVATRevenue is get list PPN Amartha revenue
	GetConfigVATRevenue(ctx context.Context) ([]models.ConfigVatRevenue, error)
	UpsertConfigVATRevenue(ctx context.Context, vatRevenue []models.ConfigVatRevenue) error
}

type gcsMasterDataRepository struct {
	client *storage.Client

	configsVatRevenue safeaccess.ObjectStorageClient[[]models.ConfigVatRevenue]
	orderTypes        safeaccess.ObjectStorageClient[[]models.OrderType]

	orderTypeCodes       []string
	transactionTypeCodes []string
}

func NewGCSMasterDataRepository(cfg *config.Config, opts ...option.ClientOption) (MasterDataRepository, error) {
	if cfg.MasterData.BucketName == "" {
		return nil, fmt.Errorf("failed to init master orderTypes, bucket name not set")
	}

	if cfg.MasterData.OrderTypeFilePath == "" {
		return nil, fmt.Errorf("failed to init master orderTypes, file path not set")
	}

	if cfg.MasterData.VatRevenueFilePath == "" {
		return nil, fmt.Errorf("failed to init master vat revenue, file path not set")
	}

	client, err := storage.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	return &gcsMasterDataRepository{
		configsVatRevenue: safeaccess.NewGCSJson[[]models.ConfigVatRevenue](
			client.Bucket(cfg.MasterData.BucketName).Object(cfg.MasterData.VatRevenueFilePath),
		),
		orderTypes: safeaccess.NewGCSJson[[]models.OrderType](
			client.Bucket(cfg.MasterData.BucketName).Object(cfg.MasterData.OrderTypeFilePath),
		),
		client: client,
	}, nil
}

func (g *gcsMasterDataRepository) updateTransactionCodes(data []models.OrderType) {
	var orderTypeCodes []string
	for _, datum := range data {
		orderTypeCodes = append(orderTypeCodes, datum.OrderTypeCode)
	}

	var transactionTypeCodes []string
	for _, datum := range data {
		for _, transactionType := range datum.TransactionTypes {
			transactionTypeCodes = append(transactionTypeCodes, transactionType.TransactionTypeCode)
		}
	}

	g.orderTypeCodes = orderTypeCodes
	g.transactionTypeCodes = transactionTypeCodes
}

func (g *gcsMasterDataRepository) repopulate(ctx context.Context) error {
	err := g.orderTypes.LoadFile(ctx)
	if err != nil {
		return fmt.Errorf("failed to read master data: %w", err)
	}

	err = g.configsVatRevenue.LoadFile(ctx)
	if err != nil {
		return fmt.Errorf("failed to read vat revenue: %w", err)
	}

	g.updateTransactionCodes(g.orderTypes.Value().Load())

	return nil
}

func (g *gcsMasterDataRepository) RefreshDataPeriodically(ctx context.Context, interval time.Duration) {
	err := g.repopulate(ctx)
	if err != nil {
		xlog.Warn(ctx, "failed to repopulate master data", xlog.Err(err))
	}

	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				err = g.repopulate(ctx)
				if err != nil {
					xlog.Warn(ctx, "failed to repopulate master data", xlog.Err(err))
				}
			}
		}
	}()
}

func (g *gcsMasterDataRepository) GetListOrderTypeCode(ctx context.Context) ([]string, error) {
	return g.orderTypeCodes, nil
}

func (g *gcsMasterDataRepository) GetListTransactionTypeCode(ctx context.Context) ([]string, error) {
	return g.transactionTypeCodes, nil
}

func (g *gcsMasterDataRepository) GetListTransactionType(ctx context.Context, filter models.FilterMasterData) ([]models.TransactionType, error) {
	var result []models.TransactionType
	for _, v := range g.orderTypes.Value().Load() {
		for _, transactionType := range v.TransactionTypes {
			isMatchCode := filter.Code != "" && transactionType.TransactionTypeCode == filter.Code
			isMatchName := filter.Name != "" && transactionType.TransactionTypeName == filter.Name

			if (filter.Code == "" && filter.Name == "") || isMatchCode || isMatchName {
				result = append(result, transactionType)
			}
		}
	}

	return result, nil
}

func (g *gcsMasterDataRepository) GetListOrderType(ctx context.Context, filter models.FilterMasterData) ([]models.OrderType, error) {
	var result []models.OrderType
	for _, orderType := range g.orderTypes.Value().Load() {
		isMatchCode := filter.Code != "" && orderType.OrderTypeCode == filter.Code
		isMatchName := filter.Name != "" && orderType.OrderTypeName == filter.Name

		if (filter.Code == "" && filter.Name == "") || isMatchCode || isMatchName {
			result = append(result, orderType)
		}
	}

	return result, nil
}

func (g *gcsMasterDataRepository) UpsertOrderType(ctx context.Context, orderType models.OrderType) (err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	orderTypes := g.orderTypes.Value().Load()

	isNeedAppend := true
	for i, datum := range orderTypes {
		if datum.OrderTypeCode == orderType.OrderTypeCode {
			orderTypes[i] = orderType
			isNeedAppend = false
			break
		}
	}

	if isNeedAppend {
		orderTypes = append(orderTypes, orderType)
	}

	g.orderTypes.Value().Store(orderTypes)
	g.updateTransactionCodes(orderTypes)

	err = g.orderTypes.UpdateFile(ctx)
	if err != nil {
		return fmt.Errorf("failed to update master orderTypes: %w", err)
	}

	return nil
}

func (g *gcsMasterDataRepository) GetOrderType(ctx context.Context, orderTypeCode string) (*models.OrderType, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	var result *models.OrderType
	for _, orderType := range g.orderTypes.Value().Load() {
		if orderType.OrderTypeCode == orderTypeCode {
			result = &orderType
			break
		}
	}

	if result == nil {
		err = common.ErrDataNotFound
		return nil, err
	}

	return result, nil
}

func (g *gcsMasterDataRepository) GetTransactionType(ctx context.Context, transactionTypeCode string) (*models.TransactionType, error) {
	var err error

	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	var result *models.TransactionType
	for _, orderType := range g.orderTypes.Value().Load() {
		for _, transactionType := range orderType.TransactionTypes {
			if transactionType.TransactionTypeCode == transactionTypeCode {
				result = &transactionType
				break
			}
		}
	}

	if result == nil {
		err = common.ErrDataNotFound
		return nil, err
	}

	return result, nil
}

func (g *gcsMasterDataRepository) GetConfigVATRevenue(_ context.Context) ([]models.ConfigVatRevenue, error) {
	return g.configsVatRevenue.Value().Load(), nil
}

func (g *gcsMasterDataRepository) UpsertConfigVATRevenue(ctx context.Context, vatRevenue []models.ConfigVatRevenue) error {
	g.configsVatRevenue.Value().Store(vatRevenue)

	err := g.configsVatRevenue.UpdateFile(ctx)
	if err != nil {
		return fmt.Errorf("failed to update master configVATRevenue: %w", err)
	}

	return nil
}
