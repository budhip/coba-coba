package repositories

import (
	"context"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/accounting"
)

type accountConfigFromExternal struct {
	accountingClient accounting.Client
}

func (a *accountConfigFromExternal) findByKey(accountDetails []accounting.DetailAccountNumber, key string) (string, error) {
	for _, accountDetail := range accountDetails {
		if accountDetail.AccountType == key {
			return accountDetail.AccountNumber, nil
		}
	}

	return "", fmt.Errorf("config account number not found from external system go-accounting. missing: %s", key)
}

func (a *accountConfigFromExternal) GetWht2326(ctx context.Context, loanAccountNumber string, loanKind string) (string, error) {
	res, err := a.accountingClient.GetLoanPartnerAccounts(ctx, loanAccountNumber, "")
	if err != nil {
		return "", err
	}

	return a.findByKey(res.Contents, "INTERNAL_ACCOUNTS_PPH_AMARTHA")
}

func (a *accountConfigFromExternal) GetVatOut(ctx context.Context, loanAccountNumber string, loanKind string) (string, error) {
	res, err := a.accountingClient.GetLoanPartnerAccounts(ctx, loanAccountNumber, "")
	if err != nil {
		return "", err
	}

	return a.findByKey(res.Contents, "INTERNAL_ACCOUNTS_PPN_AMARTHA")
}

func (a *accountConfigFromExternal) GetRevenue(ctx context.Context, loanAccountNumber string, loanKind string) (string, error) {
	res, err := a.accountingClient.GetLoanPartnerAccounts(ctx, loanAccountNumber, "")
	if err != nil {
		return "", err
	}

	return a.findByKey(res.Contents, "INTERNAL_ACCOUNTS_REVENUE_AMARTHA")
}

func (a *accountConfigFromExternal) GetAdminFee(ctx context.Context, loanAccountNumber string, loanKind string) (string, error) {
	res, err := a.accountingClient.GetLoanPartnerAccounts(ctx, loanAccountNumber, "")
	if err != nil {
		return "", err
	}

	return a.findByKey(res.Contents, "INTERNAL_ACCOUNTS_ADMIN_FEE_AMARTHA")
}
