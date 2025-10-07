package accounting

const SERVICE_NAME string = "go-accounting"

type ResponseGetLenderAccount struct {
	Kind                    string `json:"kind"`
	CihAccountNumber        string `json:"cihAccountNumber"`
	InvestedAccountNumber   string `json:"investedAccountNumber"`
	ReceivableAccountNumber string `json:"receivablesAccountNumber"`
}

type ResponseGetLoanAccount struct {
	Kind                            string `json:"kind"`
	LoanAccountNumber               string `json:"loanAccountNumber"`
	LoanAdvancePaymentAccountNumber string `json:"loanAdvancePaymentAccountNumber"`
}

type ResponseGetListAccountNumber struct {
	Kind     string                `json:"kind"`
	Contents []DetailAccountNumber `json:"contents"`
}

type DetailAccountNumber struct {
	Kind                string `json:"kind"`
	PartnerId           string `json:"partnerId"`
	LoanLKind           string `json:"loanLKind"`
	AccountNumber       string `json:"accountNumber"`
	AccountType         string `json:"accountType"`
	EntityCode          string `json:"entityCode"`
	LoanSubCategoryCode string `json:"loanSubCategoryCode"`
}
