package mapper

import (
	"context"
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/repositories"
)

type AccountMapper interface {
	CreateAccountMapping(t24AccountNumber, pasAccountNumber string) error
	GetPASAccountNumber(t24AccountNumber string) (string, error)
}

type accountMapper struct {
	repo repositories.AccountRepository
}

func NewMapper(ar repositories.AccountRepository) AccountMapper {
	return &accountMapper{
		repo: ar,
	}
}

func (am accountMapper) CreateAccountMapping(_, _ string) error {
	return nil
}

// GetPASAccountNumber returns the PAS account number for the given T24 account number.
// TODO: improvement use caching to reduce the number of calls to the database.
func (am accountMapper) GetPASAccountNumber(t24AccountNumber string) (string, error) {
	account, err := am.repo.GetOneByLegacyId(context.TODO(), t24AccountNumber)
	if err != nil {
		return "", fmt.Errorf("error getting account: %w", err)
	}

	return account.AccountNumber, nil
}
