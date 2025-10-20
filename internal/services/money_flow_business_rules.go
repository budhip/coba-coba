package services

import (
	"fmt"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

// BusinessRulesHelper handles business rules operations
type BusinessRulesHelper struct {
	configs *models.BusinessRulesConfigs
}

// NewBusinessRulesHelper creates a new BusinessRulesHelper instance
func NewBusinessRulesHelper(configs *models.BusinessRulesConfigs) *BusinessRulesHelper {
	return &BusinessRulesHelper{configs: configs}
}

// GetByPaymentType retrieves business rule config by payment type
func (h *BusinessRulesHelper) GetByPaymentType(paymentType string) (*models.BusinessRuleConfig, error) {
	config, exists := h.configs.BusinessRulesConfigs[paymentType]
	if !exists {
		return nil, fmt.Errorf("payment type not found: %s", paymentType)
	}
	return &config, nil
}

// GetByTransactionType retrieves business rule config by transaction type
func (h *BusinessRulesHelper) GetByTransactionType(transactionType string) (*models.BusinessRuleConfig, string, error) {
	paymentType, exists := h.configs.TransactionToPaymentMap[transactionType]
	if !exists {
		return nil, "", fmt.Errorf("transaction type not found: %s", transactionType)
	}

	config, err := h.GetByPaymentType(paymentType)
	if err != nil {
		return nil, "", err
	}

	return config, paymentType, nil
}
