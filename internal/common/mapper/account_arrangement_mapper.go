package mapper

import (
	"fmt"
)

type AccountArrangementMapper interface {
	CreateAccountArrangementMapping(arrangementId, loanAccountNumber string) error
	GetLoanAccountNumberByArrangementId(arrangementId string) (string, error)
}

type accountArrangementMapper struct{}

func NewMapperArrangement() AccountArrangementMapper {
	return &accountArrangementMapper{}
}

func (am accountArrangementMapper) CreateAccountArrangementMapping(_, _ string) error {
	return nil
}

func (am accountArrangementMapper) GetLoanAccountNumberByArrangementId(arrangementId string) (string, error) {
	return "", fmt.Errorf("not implemented")
}
