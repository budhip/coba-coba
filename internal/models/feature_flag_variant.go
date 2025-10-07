package models

type AllowedTransactionTypesVariant struct {
	AllowedTransactionTypeIbuDB []string `json:"allowedTransactionTypeIbuDB"`
	AllowedTransactionTypeAcuan []string `json:"allowedTransactionTypeAcuan"`
}

type ExcludeConsumeTransactionVariant struct {
	SubCategories []string `json:"subCategories"`
}
