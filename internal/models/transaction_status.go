package models

type TransactionStatus string

func (m TransactionStatus) String() string {
	return string(m)
}

const (
	TransactionStatusPending TransactionStatus = "0"
	TransactionStatusSuccess TransactionStatus = "1"
	TransactionStatusCancel  TransactionStatus = "2"
)

var (
	// MapTransactionStatus is a map of transaction status with its title for display purpose
	MapTransactionStatus = map[TransactionStatus]string{
		TransactionStatusPending: "PENDING",
		TransactionStatusSuccess: "SUCCESS",
		TransactionStatusCancel:  "CANCEL",
	}
)
