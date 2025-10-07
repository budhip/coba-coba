package models

import (
	"encoding/json"
)

// BalanceLogsPayload is the payload kafka message for balance-logs topic
type BalanceLogsPayload struct {
	Before Balance `json:"before"`
	After  Balance `json:"after"`
}

func (blp BalanceLogsPayload) MarshalJSON() ([]byte, error) {
	res := struct {
		Before balanceJSONV2 `json:"before"`
		After  balanceJSONV2 `json:"after"`
	}{
		Before: newBalanceJSONV2(blp.Before),
		After:  newBalanceJSONV2(blp.After),
	}

	return json.Marshal(res)
}
