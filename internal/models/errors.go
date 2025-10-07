package models

import (
	"errors"
	"fmt"
)

type (
	MapErrs     map[string]ErrorDetail
	ErrorDetail struct {
		Code         string `json:"code,omitempty"`
		ErrorMessage error  `json:"message,omitempty"`
	}
)

func (e ErrorDetail) Error() string {
	return fmt.Sprintf("code: %s, message: %v", e.Code, e.ErrorMessage)
}

func GetErrMap(code string, args ...string) ErrorDetail {
	v, ok := MapErrors[code]
	if !ok {
		return ErrorDetail{
			Code:         code,
			ErrorMessage: errors.New("unknown error mapping"),
		}
	}
	if len(args) > 0 {
		v.ErrorMessage = fmt.Errorf("%s caused by %s", v.ErrorMessage, args[0])
	}

	return v
}
