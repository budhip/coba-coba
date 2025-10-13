package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/cache"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/flag"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	xlog "bitbucket.org/Amartha/go-x/log"
)

type sqlRepo struct {
	r *Repository
}

type Repository struct {
	dbWrite *sql.DB
	dbRead  *sql.DB
	config  config.Config
	flag    flag.Client
	common  sqlRepo

	ar   *accountRepository
	br   *balanceRepository
	tr   *transactionRepository
	abdr *accountBalanceDailyRepository
	cr   *categoryRepository
	scr  *subCategoryRepository
	er   *entityRepository
	rsr  *reconToolHistoryRepo
	fr   *featureRepository
	wtr  *walletTrxRepo
	mfc  *moneyFlowRepository

	accountConfigFromInternal AccountConfigRepository
	accountConfigFromExternal AccountConfigRepository

	cacheAccount cache.Client[models.GetAccountOut]
}

func NewSQLRepository(
	dbWrite *sql.DB,
	dbRead *sql.DB,
	cfg config.Config,
	f flag.Client,
	accounting accounting.Client,
) *Repository {
	rtx := &Repository{
		dbWrite: dbWrite,
		dbRead:  dbRead,
		config:  cfg,
		flag:    f,
	}
	rtx.common.r = rtx
	rtx.ar = (*accountRepository)(&rtx.common)
	rtx.br = (*balanceRepository)(&rtx.common)
	rtx.tr = (*transactionRepository)(&rtx.common)
	rtx.abdr = (*accountBalanceDailyRepository)(&rtx.common)
	rtx.cr = (*categoryRepository)(&rtx.common)
	rtx.scr = (*subCategoryRepository)(&rtx.common)
	rtx.er = (*entityRepository)(&rtx.common)
	rtx.rsr = (*reconToolHistoryRepo)(&rtx.common)
	rtx.fr = (*featureRepository)(&rtx.common)
	rtx.wtr = (*walletTrxRepo)(&rtx.common)
	rtx.mfc = (*moneyFlowRepository)(&rtx.common)

	rtx.accountConfigFromInternal = (*accountConfigRepository)(&rtx.common)
	rtx.accountConfigFromExternal = &accountConfigFromExternal{accountingClient: accounting}

	rtx.cacheAccount = cache.NewInMemoryClient[models.GetAccountOut]()

	return rtx
}

type SQLRepository interface {
	Atomic(ctx context.Context, steps func(ctx context.Context, r SQLRepository) error) error
	GetAccountRepository() AccountRepository
	GetTransactionRepository() TransactionRepository
	GetAccountBalanceDailyRepository() AccountBalanceDailyRepository
	GetCategoryRepository() CategoryRepository
	GetSubCategoryRepository() SubCategoryRepository
	GetEntityRepository() EntityRepository
	GetReconToolHistoryRepository() ReconToolHistoryRepository
	GetFeatureRepository() FeatureRepository
	GetWalletTransactionRepository() WalletTransactionRepository
	GetBalanceRepository() BalanceRepository

	GetAccountConfigInternalRepository() AccountConfigRepository
	GetAccountConfigExternalRepository() AccountConfigRepository

	// DisableIndexScan is a temporary solution to disable index scan
	// it will make sure query planner will use another plan beside index scan (sequential scan, bitmap scan, etc)
	// note: make sure only use this inside Atomic function, so it only affect the current transaction
	DisableIndexScan(ctx context.Context) (err error)

	GetMoneyFlowCalcRepository() MoneyFlowRepository
}

var _ SQLRepository = (*Repository)(nil)

func (r *Repository) Atomic(ctx context.Context, steps func(ctx context.Context, r SQLRepository) error) (err error) {
	tx, err := r.dbWrite.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	xlog.Info(ctx, "[DATABASE.TRANSACTION.BEGIN]")
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			err = fmt.Errorf("panic happened because: %v", p)
			xlog.Panic(ctx, "[DATABASE.TRANSACTION.PANIC]", xlog.Err(err))
		} else if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
			}
			xlog.Warn(ctx, "[DATABASE.TRANSACTION.ROLLBACK]", xlog.Err(err))
		} else {
			if err = tx.Commit(); err != nil {
				if errors.Is(err, sql.ErrTxDone) {
					xlog.Warn(ctx, "[DATABASE.TRANSACTION.ALREADY_COMMITTED_OR_ROLLEDBACK]", xlog.Err(err))
					err = nil
				}
			}

			xlog.Info(ctx, "[DATABASE.TRANSACTION.COMMIT]")
		}
	}()
	ctx = injectTx(ctx, tx)
	err = steps(ctx, r)
	return
}

func (r *Repository) GetAccountRepository() AccountRepository {
	return r.ar
}

func (r *Repository) GetBalanceRepository() BalanceRepository {
	return r.br
}

func (r *Repository) GetTransactionRepository() TransactionRepository {
	return r.tr
}

func (r *Repository) GetAccountBalanceDailyRepository() AccountBalanceDailyRepository {
	return r.abdr
}

func (r *Repository) GetCategoryRepository() CategoryRepository {
	return r.cr
}

func (r *Repository) GetSubCategoryRepository() SubCategoryRepository {
	return r.scr
}

func (r *Repository) GetEntityRepository() EntityRepository {
	return r.er
}

func (r *Repository) GetReconToolHistoryRepository() ReconToolHistoryRepository {
	return r.rsr
}

func (r *Repository) GetFeatureRepository() FeatureRepository {
	return r.fr
}

func (r *Repository) GetWalletTransactionRepository() WalletTransactionRepository {
	return r.wtr
}

func (r *Repository) GetAccountConfigInternalRepository() AccountConfigRepository {
	return r.accountConfigFromInternal
}

func (r *Repository) GetAccountConfigExternalRepository() AccountConfigRepository {
	return r.accountConfigFromExternal
}

func (r *Repository) DisableIndexScan(ctx context.Context) (err error) {
	db := r.extractTxWrite(ctx)
	_, err = db.ExecContext(ctx, "SET enable_indexscan = OFF;")
	return
}

func (r *Repository) SubstitutePlaceholder(data string, startInt int) (res string) {
	placeholderCount := strings.Count(data, "?")
	res = data
	for i := startInt; i < startInt+placeholderCount; i++ {
		res = strings.Replace(res, "?", "$"+strconv.Itoa(i), 1)
	}
	return res
}

func (r *Repository) GetMoneyFlowCalcRepository() MoneyFlowRepository {
	return r.mfc
}
