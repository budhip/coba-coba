package validation

import "fmt"

type (
	ErrorValidateResponse struct {
		Code    string `json:"code,omitempty" example:"accountNumber_required"`
		Field   string `json:"field,omitempty" example:"MISSING_FIELD"`
		Message string `json:"message,omitempty" example:"field is missing"`
	}
)

func (e ErrorValidateResponse) Error() string {
	return fmt.Sprintf("code: %s, field: %s, message: %s", e.Code, e.Field, e.Message)
}
