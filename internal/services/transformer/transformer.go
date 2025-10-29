package transformer

import (
	"context"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/flag"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/monitoring"
	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"

	"github.com/hashicorp/go-multierror"
	"github.com/shopspring/decimal"
)

// Transformer is an interface that will be used to transform wallet transaction to acuan transaction
// This interface will be used for many transaction type
// for example if you have new transaction type, you can create a new struct that implements this interface
type Transformer interface {
	// Transform will transform amount to acuan transaction request
	// amount is the amount of specified transaction type
	// parentWalletTransaction is the parent transaction that will be used as reference
	Transform(ctx context.Context, amount models.Amount, parentWalletTransaction models.WalletTransaction) (res []models.TransactionReq, err error)
}

// baseWalletTransactionTransformer is a struct that will be used for helping struct that implements WalletTransactionTransformer interface
// to transform the wallet transaction to acuan transaction
// currently it only consist of config for getting system account.
// if you need more capabilities you can add it here, for example if you need to get account number from go-accounting
// you can add go-accounting client here, so another struct that implements WalletTransactionTransformer can use it
type baseWalletTransactionTransformer struct {
	config                  config.Config
	flag                    flag.Client
	accountingClient        accounting.Client
	masterDataRepository    repositories.MasterDataRepository
	accountRepository       repositories.AccountRepository
	transaction             repositories.TransactionRepository
	walletTransaction       repositories.WalletTransactionRepository
	accountConfigRepository repositories.AccountConfigRepository
}

// MapTransformer is a map that will be used to get transformer for specified transaction type
type MapTransformer map[string]Transformer

func NewMapTransformer(
	config config.Config,
	masterDataRepository repositories.MasterDataRepository,
	accountingClient accounting.Client,
	accountRepository repositories.AccountRepository,
	transaction repositories.TransactionRepository,
	accountConfigRepository repositories.AccountConfigRepository,
	walletTransaction repositories.WalletTransactionRepository,
	featureFlag flag.Client,
) MapTransformer {
	baseTransformer := baseWalletTransactionTransformer{
		config:                  config,
		masterDataRepository:    masterDataRepository,
		accountingClient:        accountingClient,
		accountRepository:       accountRepository,
		transaction:             transaction,
		accountConfigRepository: accountConfigRepository,
		walletTransaction:       walletTransaction,
		flag:                    featureFlag,
	}

	// register all transformer here
	return MapTransformer{
		"ADJAF": &adjafTransformer{baseTransformer},
		"ADMBI": &admbiTransformer{baseTransformer},
		"ADMBT": &admbtTransformer{baseTransformer},
		"ADMCB": &admcbTransformer{baseTransformer},
		"ADMCE": &admceTransformer{baseTransformer},
		"ADMDA": &admdaTransformer{baseTransformer},
		"ADMDD": &admddTransformer{baseTransformer},
		"ADMDE": &admdeTransformer{baseTransformer},
		"ADMDU": &admduTransformer{baseTransformer},
		"ADMDV": &admdvTransformer{baseTransformer},
		"ADMFE": &admfeTransformer{baseTransformer},
		"ADMFM": &admfmTransformer{baseTransformer},
		"ADMMD": &admmdTransformer{baseTransformer},
		"ADMME": &admmeTransformer{baseTransformer},
		"ADMMV": &admmvTransformer{baseTransformer},
		"ADMPB": &admpbTransformer{baseTransformer},
		"ADMPE": &admpeTransformer{baseTransformer},
		"ADMOB": &admobTransformer{baseTransformer},
		"ADMOC": &admocTransformer{baseTransformer},
		"ADMRA": &admraTransformer{baseTransformer},
		"ADMRD": &admrdTransformer{baseTransformer},
		"ADMTP": &admtpTransformer{baseTransformer},
		"ADMTD": &admtdTransformer{baseTransformer},
		"ADMTV": &admtvTransformer{baseTransformer},
		"ADMVI": &admviTransformer{baseTransformer},
		"ADMWP": &admwpTransformer{baseTransformer},
		"ADMPF": &admpfTransformer{baseTransformer},
		"ADMPN": &admpnTransformer{baseTransformer},
		"ADMPP": &admppTransformer{baseTransformer},
		"ADMPT": &admptTransformer{baseTransformer},
		"ADMRF": &admrfTransformer{baseTransformer},
		"ADMRP": &admrpTransformer{baseTransformer},
		"ADMRV": &admrvTransformer{baseTransformer},
		"BBLDN": &bbldnTransformer{baseTransformer},
		"BBLEN": &bblenTransformer{baseTransformer},
		"COTLR": &cotlrTransformer{baseTransformer},
		"COTMF": &cotmfTransformer{baseTransformer},
		"COTPB": &cotpbTransformer{baseTransformer},
		"COTPR": &cotprTransformer{baseTransformer},
		"COTRC": &cotrcTransformer{baseTransformer},
		"COTRJ": &cotrjTransformer{baseTransformer},
		"COTRT": &cotrtTransformer{baseTransformer},
		"COTWC": &cotwcTransformer{baseTransformer},
		"COTRQ": &cotrqTransformer{baseTransformer},
		"COTAI": &cotaiTransformer{baseTransformer},
		"COTAM": &cotamTransformer{baseTransformer},
		"COTBM": &cotbmTransformer{baseTransformer},
		"COTDA": &cotdaTransformer{baseTransformer},
		"COTGC": &cotgcTransformer{baseTransformer},
		"DBFAA": &dbfaaTransformer{baseTransformer},
		"DBFAB": &dbfabTransformer{baseTransformer},
		"DBFAC": &dbfacTransformer{baseTransformer},
		"DBFEA": &dbfeaTransformer{baseTransformer},
		"DSBAA": &dsbaaTransformer{baseTransformer},
		"DSBAB": &dsbabTransformer{baseTransformer},
		"DSBAO": &dsbaoTransformer{baseTransformer},
		"DSBBA": &dsbbaTransformer{baseTransformer},
		"DSBBB": &dsbbbTransformer{baseTransformer},
		"DSBBC": &dsbbcTransformer{baseTransformer},
		"DSBBD": &dsbbdTransformer{baseTransformer},
		"DSBBE": &dsbbeTransformer{baseTransformer},
		"DSBED": &dsbedTransformer{baseTransformer},
		"DSBLD": &dsbldTransformer{baseTransformer},
		"DSBMR": &dsbmrTransformer{baseTransformer},
		"DSBPD": &dsbpdTransformer{baseTransformer},
		"DSBPO": &dsbpoTransformer{baseTransformer},
		"DSBRD": &dsbrdTransformer{baseTransformer},
		"DSBRP": &dsbrpTransformer{baseTransformer},
		"DSBTI": &dsbtiTransformer{baseTransformer},
		"DSBAP": &dsbapTransformer{baseTransformer},
		"DSBFD": &dsbfdTransformer{baseTransformer},
		"DSBLB": &dsblbTransformer{baseTransformer},
		"FPEPT": &fpeptTransformer{baseTransformer},
		"FPEPD": &fpepdTransformer{baseTransformer},
		"INSCA": &inscaTransformer{baseTransformer},
		"INSCL": &insclTransformer{baseTransformer},
		"INSDL": &insdlTransformer{baseTransformer},
		"INSHN": &inshnTransformer{baseTransformer},
		"INSLL": &insllTransformer{baseTransformer},
		"INSLR": &inslrTransformer{baseTransformer},
		"INSPA": &inspaTransformer{baseTransformer},
		"INSPI": &inspiTransformer{baseTransformer},
		"INSPL": &insplTransformer{baseTransformer},
		"INSPN": &inspnTransformer{baseTransformer},
		"INVMT": &invmtTransformer{baseTransformer},
		"INVVO": &invvoTransformer{baseTransformer},
		"ITDED": &itdedTransformer{baseTransformer},
		"ITDEP": &itdepTransformer{baseTransformer},
		"ITDPH": &itdphTransformer{baseTransformer},
		"ITRTF": &itrtfTransformer{baseTransformer},
		"ITRTP": &itrtpTransformer{baseTransformer},
		"MFAAJ": &mfaajTransformer{baseTransformer},
		"MFAQR": &mfaqrTransformer{baseTransformer},
		"MFFMD": &mffmdTransformer{baseTransformer},
		"MFFWC": &mffwcTransformer{baseTransformer},
		"MFFEP": &mffepTransformer{baseTransformer},
		"MFMRP": &mfmrpTransformer{baseTransformer},
		"MFMWC": &mfmwcTransformer{baseTransformer},
		"MFNPC": &mfnpcTransformer{baseTransformer},
		"MFNPR": &mfnprTransformer{baseTransformer},
		"MFMEP": &mfmepTransformer{baseTransformer},
		"MFMPP": &mfmppTransformer{baseTransformer},
		"MFFGL": &mffglTransformer{baseTransformer},
		"MFWIT": &mfwitTransformer{baseTransformer},
		"MFWAF": &mfwafTransformer{baseTransformer},
		"MFMMP": &mfmmpTransformer{baseTransformer},
		"MFWLF": &mfwlfTransformer{baseTransformer},
		"MFWPH": &mfwphTransformer{baseTransformer},
		"MMWPD": &mmwpdTransformer{baseTransformer},
		"MFWPN": &mfwpnTransformer{baseTransformer},
		"MFWRP": &mfwrpTransformer{baseTransformer},
		"MFWRQ": &mfwrqTransformer{baseTransformer},
		"MFMTP": &mfmtpTransformer{baseTransformer},
		"MWMPD": &mwmpdTransformer{baseTransformer},
		"PAYDL": &paydlTransformer{baseTransformer},
		"PAYDP": &paydpTransformer{baseTransformer},
		"PAYFL": &payflTransformer{baseTransformer},
		"PAYFP": &payfpTransformer{baseTransformer},
		"PAYGL": &payglTransformer{baseTransformer},
		"PAYMD": &paymdTransformer{baseTransformer},
		"PAYPC": &paypcTransformer{baseTransformer},
		"PAYPD": &paypdTransformer{baseTransformer},
		"PAYPM": &paypmTransformer{baseTransformer},
		"PAYPR": &payprTransformer{baseTransformer},
		"PAYPV": &paypvTransformer{baseTransformer},
		"PAYQR": &payqrTransformer{baseTransformer},
		"PAYVP": &payvpTransformer{baseTransformer},
		"PAYWM": &paywmTransformer{baseTransformer},
		"PAYWC": &paywcTransformer{baseTransformer},
		"PRMCB": &prmcbTransformer{baseTransformer},
		"RFDCB": &rfdcbTransformer{baseTransformer},
		"RFDDL": &rfddlTransformer{baseTransformer},
		"RFDDP": &rfddpTransformer{baseTransformer},
		"RFDFL": &rfdflTransformer{baseTransformer},
		"RFDFP": &rfdfpTransformer{baseTransformer},
		"RFDMD": &rfdmdTransformer{baseTransformer},
		"RFDMP": &rfdmpTransformer{baseTransformer},
		"RFDPD": &rfdpdTransformer{baseTransformer},
		"RFDPC": &rfdpcTransformer{baseTransformer},
		"RFDPP": &rfdppTransformer{baseTransformer},
		"RFDPV": &rfdpvTransformer{baseTransformer},
		"RFDQR": &rfdqrTransformer{baseTransformer},
		"RFDMT": &rfdmtTransformer{baseTransformer},
		"RFDPR": &rfdprTransformer{baseTransformer},
		"RFDPY": &rfdpyTransformer{baseTransformer},
		"RFDTX": &rfdtxTransformer{baseTransformer},
		"RFDVP": &rfdvpTransformer{baseTransformer},
		"RPYAA": &rpyaaTransformer{baseTransformer},
		"RPYAB": &rpyabTransformer{baseTransformer},
		"RPYAC": &rpyacTransformer{baseTransformer},
		"RPYAD": &rpyadTransformer{baseTransformer},
		"RPYAE": &rpyaeTransformer{baseTransformer},
		"RPYAF": &rpyafTransformer{baseTransformer},
		"RPYAH": &rpyahTransformer{baseTransformer},
		"RPYAI": &rpyaiTransformer{baseTransformer},
		"RPYAJ": &rpyajTransformer{baseTransformer},
		"RPYAK": &rpyakTransformer{baseTransformer},
		"RPYAO": &rpyaoTransformer{baseTransformer},
		"RPYBV": &rpybvTransformer{baseTransformer},
		"RPYMC": &rpymcTransformer{baseTransformer},
		"RPYPD": &rpypdTransformer{baseTransformer},
		"RPYTR": &rpytrTransformer{baseTransformer},
		"RPYRD": &rpyrdTransformer{baseTransformer},
		"RPYVA": &rpyvaTransformer{baseTransformer},
		"RPYPO": &rpypoTransformer{baseTransformer},
		"RPYRO": &rpyroTransformer{baseTransformer},
		"RPYTD": &rpytdTransformer{baseTransformer},
		"RPYCO": &rpycoTransformer{baseTransformer},
		"RPYEN": &rpyenTransformer{baseTransformer},
		"RPYIO": &rpyioTransformer{baseTransformer},
		"RVRSL": &rvrslTransformer{baseTransformer},
		"SIVEA": &siveaTransformer{baseTransformer},
		"SIVED": &sivedTransformer{baseTransformer},
		"SIVEP": &sivepTransformer{baseTransformer},
		"TUPCB": &tupcbTransformer{baseTransformer},
		"TUPEP": &tupepTransformer{baseTransformer},
		"TUPGC": &tupgcTransformer{baseTransformer},
		"TUPFE": &tupfeTransformer{baseTransformer},
		"TUPGE": &tupgeTransformer{baseTransformer},
		"TUPIK": &tupikTransformer{baseTransformer},
		"TUPIL": &tupilTransformer{baseTransformer},
		"TUPLF": &tuplfTransformer{baseTransformer},
		"TUPLR": &tuplrTransformer{baseTransformer},
		"TUPPY": &tuppyTransformer{baseTransformer},
		"TUPQR": &tupqrTransformer{baseTransformer},
		"TUPTI": &tuptiTransformer{baseTransformer},
		"TUPVA": &tupvaTransformer{baseTransformer},
		"TUPVB": &tupvbTransformer{baseTransformer},
		"TUPVI": &tupviTransformer{baseTransformer},
		"TUPVM": &tupvmTransformer{baseTransformer},
		"TUPVP": &tupvpTransformer{baseTransformer},
		"TUPWC": &tupwcTransformer{baseTransformer},
		"TUPEN": &tupenTransformer{baseTransformer},
		"TUPLW": &tuplwTransformer{baseTransformer},
		"TUPWD": &tupwdTransformer{baseTransformer},
		"TUPWM": &tupwmTransformer{baseTransformer},
		"TUPWX": &tupwxTransformer{baseTransformer},
		"TUPBA": &tupbaTransformer{baseTransformer},
		"TUPPB": &tuppbTransformer{baseTransformer},
		"TUPPO": &tuppoTransformer{baseTransformer},
		"TUPIN": &tupinTransformer{baseTransformer},
		"TUPIP": &tupipTransformer{baseTransformer},
		"TUPBH": &tupbhTransformer{baseTransformer},
		"TUPBM": &tupbmTransformer{baseTransformer},
		"TUPDN": &tupdnTransformer{baseTransformer},
		"TUPGP": &tupgpTransformer{baseTransformer},
		"TUPED": &tupedTransformer{baseTransformer},
		"WOLPB": &wolpbTransformer{baseTransformer},
		"WOLAR": &wolarTransformer{baseTransformer},
		"WOLIL": &wolilTransformer{baseTransformer},
		"WOLLR": &wollrTransformer{baseTransformer},
		"WOLLC": &wollcTransformer{baseTransformer},
		"DSBTF": &dsbtfTransformer{baseTransformer},
		"ADMMA": &admmaTransformer{baseTransformer},
		"TUPPP": &tupppTransformer{baseTransformer},
		"SIVTF": &sivtfTransformer{baseTransformer},
		"BBLTF": &bbltfTransformer{baseTransformer},
		"RVRTF": &rvrtfTransformer{baseTransformer},
	}
}

func (m MapTransformer) GetTransformer(transactionType string) (Transformer, error) {
	transformer, ok := m[transactionType]
	if !ok {
		return nil, fmt.Errorf("%w: %s not found", common.ErrUnableGetTransformer, transactionType)
	}

	return transformer, nil
}

// Transform will transform wallet transaction to acuan transaction
// since there are many transaction type in wallet transaction, we need to get the transformer for specified transaction type
// then we will use the transformer to transform the wallet transaction to acuan transaction
func (m MapTransformer) Transform(ctx context.Context, in models.WalletTransaction) (res []models.TransactionReq, err error) {
	monitor := monitoring.New(ctx)
	defer monitor.Finish(monitoring.WithFinishCheckError(err))

	transformer, err := m.GetTransformer(in.TransactionType)
	if err != nil {
		return nil, err
	}

	var transformed []models.TransactionReq
	if in.NetAmount.ValueDecimal.GreaterThan(decimal.Zero) {
		transformed, err = transformer.Transform(ctx, in.NetAmount, in)
		if err != nil {
			return nil, err
		}

		res = append(res, transformed...)
	}

	var errs *multierror.Error
	for _, amount := range in.Amounts {
		transformer, err = m.GetTransformer(amount.Type)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		if amount.Amount.ValueDecimal.GreaterThan(decimal.Zero) {
			transformed, err = transformer.Transform(ctx, *amount.Amount, in)
			if err != nil {
				errs = multierror.Append(errs, err)
				continue
			}

			res = append(res, transformed...)
		}
	}

	return res, errs.ErrorOrNil()
}
