package services

import (
	"fmt"
)

func generateAccountNumber(categoryCode, entityCode string, padWidth, lastSequence int64) (string, error) {
	accountPrefix := fmt.Sprintf("%s%s", categoryCode, entityCode)
	pad := leftZeroPad(lastSequence, padWidth)
	if len(pad) != int(padWidth) {
		return "", fmt.Errorf("lastSequence %v exceed padding width %v", padWidth, lastSequence)
	}
	accountNumber := fmt.Sprintf("%s%s", accountPrefix, pad)
	return accountNumber, nil
}

func leftZeroPad(input, padWidth int64) string {
	return fmt.Sprintf(fmt.Sprintf("%%0%dd", padWidth), input)
}
