package services

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
)

const FileWalletRowLength = 11

func (s *file) parseErrorWalletTransaction(record []string, lineNumb int, err string) (walletTrx models.ErrWalletTransaction) {
	walletTrx.LineNumb = fmt.Sprintf("[LINE %d]", lineNumb)
	walletTrx.RefNumber = record[1]
	walletTrx.TransactionType = record[3]
	walletTrx.AccountNumber = record[4]
	walletTrx.Error = err

	return
}

func (s *file) parseWalletTransactionData(defaultRefNumber string, record []string) (walletTrx models.NewWalletTransaction, err error) {
	combinedCols := strings.ReplaceAll(strings.Join(record, ""), " ", "")
	if combinedCols == "" {
		err = common.ErrCSVRowIsEmpty
		return
	}

	if len(record) < FileWalletRowLength {
		err = fmt.Errorf("column length mismatch with model type. value: %s", record)
		return
	}

	walletTrx.ID = uuid.NewString()
	walletTrx.Status = models.WalletTransactionStatusSuccess

	// Row 1 - Transaction Date
	walletTrx.TransactionTime, err = common.ParseStringToDatetime(common.DateFormatDDMMYYYYWithoutDash, record[0])
	if err != nil {
		err = fmt.Errorf("unable to parse date %s: %w", record[0], err)
		return
	}

	if common.IsTodayAfterDate(walletTrx.TransactionTime) {
		err = fmt.Errorf("transaction date can't be the next day. value: %s", record[0])
		return
	}

	if s.srv.conf.TransactionConfig.TransactionTimeUploadMaxWindowDays > 0 {
		limitDate := time.Now().AddDate(0, 0, -s.srv.conf.TransactionConfig.TransactionTimeUploadMaxWindowDays)
		if walletTrx.TransactionTime.Before(limitDate) {
			err = fmt.Errorf("transaction date must be within %d days from today. value: %s", s.srv.conf.TransactionConfig.TransactionTimeUploadMaxWindowDays, record[0])
			return
		}
	}

	// Row 2 - Reference Number
	walletTrx.RefNumber = record[1]
	if walletTrx.RefNumber == "" {
		walletTrx.RefNumber = defaultRefNumber
	}

	// Row 3 - Transaction Flow
	walletTrx.TransactionFlow = models.TransactionFlow(record[2])

	// Row 4 - Transaction Type
	walletTrx.TransactionType = record[3]

	// Row 5 - Account Number
	walletTrx.AccountNumber = record[4]

	// Row 6 - Amount
	amountDecimal, err := models.NewDecimal(record[5])
	if err != nil {
		err = fmt.Errorf("unable to parse amount %s: %w", record[5], err)
		return
	}
	walletTrx.NetAmount = models.Amount{
		ValueDecimal: amountDecimal,
		Currency:     models.IDRCurrency,
	}

	// Row 7 - Destination Account Number
	walletTrx.DestinationAccountNumber = record[6]

	// Row 8 - Description
	walletTrx.Description = record[7]

	// Row 9 - Meta
	walletTrx.Metadata = models.WalletMetadata{}
	if record[8] != "" {
		metadata := make(models.WalletMetadata)
		err = json.Unmarshal([]byte(record[8]), &metadata)
		if err != nil {
			err = fmt.Errorf("unable to parse metadata %s: %w", record[8], err)
			return
		}
		walletTrx.Metadata = metadata
	}

	// Row 10 - Child Transaction Type
	childTransactionTypes := strings.Split(record[9], ";")

	// Row 11 - Child Amount
	childTransactionAmounts := strings.Split(record[10], ";")

	// Process child transaction
	walletTrx.Amounts = models.Amounts{}
	if len(childTransactionTypes) != len(childTransactionAmounts) {
		err = fmt.Errorf("child transaction did not match: %d != %d", len(childTransactionTypes), len(childTransactionAmounts))
		return
	}

	for i, trxType := range childTransactionTypes {
		if trxType == "" {
			continue
		}

		childAmount, nestedErr := models.NewDecimal(childTransactionAmounts[i])
		if nestedErr != nil {
			err = fmt.Errorf("unable to parse amount (%s) (%s): %w", childTransactionAmounts[i], trxType, nestedErr)
			return
		}

		walletTrx.Amounts = append(walletTrx.Amounts, models.AmountDetail{
			Type: trxType,
			Amount: &models.Amount{
				ValueDecimal: childAmount,
				Currency:     models.IDRCurrency,
			},
		})
	}

	return
}

// isCSVHeader checks if the provided string `txt` starts with "Transaction Date".
func (s *file) isWalletTransactionCSVHeader(txt string) bool {
	return strings.HasPrefix(txt, "Transaction Date")
}
