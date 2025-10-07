package common

type AccountStatus int32

const (
	ACCOUNT_STATUS_ACTIVE AccountStatus = iota
	ACCOUNT_STATUS_INACTIVE
	ACCOUNT_STATUS_FROZEN
	ACCOUNT_STATUS_CLOSED
)

const (
	AccountStatusActive   = "active"
	AccountStatusInActive = "inactive"
	AccountStatusFrozen   = "frozen"
	AccountStatusClosed   = "closed"
)

var (
	MapAccountStatus = map[AccountStatus]string{
		ACCOUNT_STATUS_ACTIVE:   AccountStatusActive,
		ACCOUNT_STATUS_INACTIVE: AccountStatusInActive,
		ACCOUNT_STATUS_FROZEN:   AccountStatusFrozen,
		ACCOUNT_STATUS_CLOSED:   AccountStatusClosed,
	}
	MapAccountStatusReverse = map[string]AccountStatus{
		AccountStatusActive:   ACCOUNT_STATUS_ACTIVE,
		AccountStatusInActive: ACCOUNT_STATUS_INACTIVE,
		AccountStatusFrozen:   ACCOUNT_STATUS_FROZEN,
		AccountStatusClosed:   ACCOUNT_STATUS_CLOSED,
	}
)
