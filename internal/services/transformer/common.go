package transformer

import (
	"fmt"
	"strings"
	"time"

	"golang.org/x/exp/slices"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/shopspring/decimal"
)

func transformWalletTransactionStatus(statusWallet models.WalletTransactionStatus) (models.TransactionStatus, error) {
	var statusAcuan models.TransactionStatus
	switch statusWallet {
	case models.WalletTransactionStatusSuccess:
		statusAcuan = models.TransactionStatusSuccess
	case models.WalletTransactionStatusPending:
		statusAcuan = models.TransactionStatusPending
	case models.WalletTransactionStatusCancel:
		statusAcuan = models.TransactionStatusCancel
	default:
		return "", fmt.Errorf("invalid wallet status: %s", statusWallet)
	}

	return statusAcuan, nil
}

func transformCurrency(currency string) string {
	if currency == "" {
		return models.IDRCurrency
	}

	return currency
}

func getOrderTime(parentWalletTransaction models.WalletTransaction) (orderTime time.Time) {
	orderTime = parentWalletTransaction.CreatedAt

	if orderTime.IsZero() {
		orderTime = time.Now()
	}

	return orderTime
}

func getEntityFromMetadata(metadata models.WalletMetadata) string {
	if entity, ok := metadata["entity"].(string); ok {
		return entity
	}

	return ""
}

func getProductTypeFromMetadata(metadata models.WalletMetadata) string {
	if productType, ok := metadata["productType"].(string); ok {
		return productType
	}

	return ""
}

func getPartnerPPOBMetadata(metadata models.WalletMetadata) string {
	if partnerPPOB, ok := metadata["partner"].(string); ok {
		return partnerPPOB
	}

	return ""
}

func getLoanAccountNumberMetadata(metadata models.WalletMetadata) string {
	if lan, ok := metadata["loanAccountNumber"].(string); ok {
		return lan
	}

	return ""
}

func getCustomerNumberMetadata(metadata models.WalletMetadata) string {
	if customerNum, ok := metadata["customerNumber"].(string); ok {
		return customerNum
	}

	return ""
}

// getAccountNumberFromConfig returns case-insensitive account number based on the key
// we use case-insensitive comparison for this because there still open issue on viper 3rd party library (go-config-loader)
// [link issue](https://github.com/spf13/viper/issues/1014), and this will not be fixed [link](https://github.com/spf13/viper?tab=readme-ov-file#does-viper-support-case-sensitive-keys)
func getAccountNumberFromConfig(configMapAccount map[string]string, key string) (string, error) {
	for k, v := range configMapAccount {
		if strings.EqualFold(k, key) {
			if v == "" {
				return "", fmt.Errorf("%w: account number for %s is empty", common.ErrConfigAccountNumberNotFound, key)
			}

			return v, nil
		}
	}

	return "", fmt.Errorf("%w: no account found for %s", common.ErrConfigAccountNumberNotFound, key)
}

type VATOpts func(opts *vatOpts)

type vatOpts struct {
	manager models.ConfigVATRevenueManager
}

func WithVATRevenueConfig(vatConfig []models.ConfigVatRevenue) VATOpts {
	return func(opts *vatOpts) {
		opts.manager = models.ConfigVATRevenueManager{
			Config: vatConfig,
		}
	}
}

// calculateVAT calculates the VAT amount from the given amount(including VAT)
func calculateVAT(amount decimal.Decimal, transactionTime time.Time, opts ...VATOpts) (decimal.Decimal, error) {
	opt := &vatOpts{}
	for _, fnOpt := range opts {
		fnOpt(opt)
	}

	// default 0.11
	calculatedPercentage := decimal.NewFromFloat(0.11).Div(decimal.NewFromFloat(1.11))

	if len(opt.manager.Config) > 0 {
		configVAT, err := opt.manager.GetActiveConfig(transactionTime)
		if err != nil {
			return decimal.Zero, err
		}

		one := decimal.NewFromInt(1)
		calculatedPercentage = configVAT.Percentage.Div(one.Add(configVAT.Percentage))
	}

	return amount.Mul(calculatedPercentage).Round(0), nil
}

func getLoanAccountNumber(metadata models.WalletMetadata) string {
	if loanAccountNumber, ok := metadata["loanAccountNumber"].(string); ok {
		return loanAccountNumber
	}

	return ""
}

func getLoanIds(metadata models.WalletMetadata) ([]string, error) {
	loanIdsInterface, ok := metadata["loanIds"].([]any)
	if !ok {
		return nil, common.ErrInvalidLoanIdsTypeMetadata
	}

	loanIds := make([]string, len(loanIdsInterface))
	for i, v := range loanIdsInterface {
		loanIds[i], ok = v.(string)
		if !ok {
			return nil, common.ErrInvalidLoanIdsTypeMetadata
		}
	}

	return loanIds, nil
}

func getMapFromConfig(configMapAccount map[string]map[string]string, key string) map[string]string {
	for k, v := range configMapAccount {
		if strings.EqualFold(k, key) {
			return v
		}
	}

	return map[string]string{}
}

func (b baseWalletTransactionTransformer) GetSourceAccountForDSBAA(parentWalletTransaction models.WalletTransaction) string {
	return parentWalletTransaction.AccountNumber
}

func (b baseWalletTransactionTransformer) GetDestinationAccountForRPYVA() string {
	return b.config.AccountConfig.SplitEscrow
}

func (b baseWalletTransactionTransformer) getBankAccountNumberForTUPVI(parentWalletTransaction models.WalletTransaction) string {
	entity := getEntityFromMetadata(parentWalletTransaction.Metadata)
	accountNumberBank, _ := getAccountNumberFromConfig(
		b.config.AccountConfig.AccountNumberBankTUPVIForADMFE,
		entity,
	)
	return accountNumberBank
}

func (b baseWalletTransactionTransformer) getBankAccountNumberForCOTLR(parentWalletTransaction models.WalletTransaction) string {
	entity := getEntityFromMetadata(parentWalletTransaction.Metadata)

	accountNumberBank, _ := getAccountNumberFromConfig(
		b.config.AccountConfig.AccountNumberBankCOTLRForADMFEByEntity,
		entity,
	)

	return accountNumberBank
}

func (b baseWalletTransactionTransformer) GetDestinationAccountForRPYAA(parentWalletTransaction models.WalletTransaction) string {
	return parentWalletTransaction.AccountNumber
}

func (b baseWalletTransactionTransformer) GenerateAccountNumberBankForMetadataADMFE(parentWalletTransaction models.WalletTransaction) string {
	switch parentWalletTransaction.TransactionType {
	case "DSBAA":
		return b.GetSourceAccountForDSBAA(parentWalletTransaction)
	case "RPYVA":
		return b.GetDestinationAccountForRPYVA()
	case "TUPVI":
		return b.getBankAccountNumberForTUPVI(parentWalletTransaction)
	case "COTLR":
		return b.getBankAccountNumberForCOTLR(parentWalletTransaction)
	case "RPYAA":
		return b.GetDestinationAccountForRPYAA(parentWalletTransaction)
	case "TUPBH":
		return b.config.AccountConfig.BRIEscrowAFAAccountNumber
	case "COTPR":
		return b.config.AccountConfig.BRIEscrowAFAAccountNumber
	}

	return parentWalletTransaction.AccountNumber
}

func (b baseWalletTransactionTransformer) MutateMetadataByAccountEntity(entityCode string, meta models.WalletMetadata) models.WalletMetadata {
	meta["entity"] = b.config.AccountConfig.MapAccountEntity[entityCode]
	return meta
}

func getTotalAmount(wallet models.WalletTransaction, transactionType string) decimal.Decimal {
	var totalAmount decimal.Decimal

	// get from parent
	if wallet.TransactionType == transactionType {
		totalAmount = totalAmount.Add(wallet.NetAmount.ValueDecimal.Decimal)
	}

	// get all child
	for _, amount := range wallet.Amounts {
		if amount.Type == transactionType {
			totalAmount = totalAmount.Add(amount.Amount.ValueDecimal.Decimal)
		}
	}

	return totalAmount
}

func getTotalAmountTransactions(transactions []models.Transaction, transactionType string) decimal.Decimal {
	var totalAmount decimal.Decimal
	for _, transaction := range transactions {
		if transaction.TypeTransaction == transactionType {
			totalAmount = totalAmount.Add(transaction.Amount.Decimal)
		}
	}

	return totalAmount
}

func isTransactionContains(transactions []models.Transaction, transactionType string) bool {
	var transactionTypes []string
	var reversedAmount decimal.Decimal

	for _, transaction := range transactions {
		reversedAmount = reversedAmount.Add(transaction.Amount.Decimal)
		transactionTypes = append(transactionTypes, transaction.TypeTransaction)
	}

	return slices.Contains(transactionTypes, transactionType)
}

type optsCalculateTotalAmount struct {
	transactionType string
	status          []models.WalletTransactionStatus
}

func getTotalAmountFromWallets(wallets []models.WalletTransaction, opts optsCalculateTotalAmount) decimal.Decimal {
	totalAmount := decimal.Zero
	for _, wt := range wallets {
		if wt.TransactionType == opts.transactionType && slices.Contains(opts.status, wt.Status) {
			totalAmount = totalAmount.Add(getTotalAmount(wt, opts.transactionType))
		}
	}

	return totalAmount
}
