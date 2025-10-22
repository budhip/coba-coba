package common

import (
	nethttp "net/http"

	"github.com/labstack/echo/v4"
)

var statusMessage = map[int]string{
	nethttp.StatusOK:                  "success",
	nethttp.StatusBadRequest:          "badRequest",
	nethttp.StatusUnauthorized:        "unAuthorized",
	nethttp.StatusForbidden:           "forbidden",
	nethttp.StatusNotFound:            "notFound",
	nethttp.StatusInternalServerError: "internalServerError",
	nethttp.StatusCreated:             "created",
	nethttp.StatusConflict:            "duplicateTrx",
}

type ApiSuccessResponseModel struct {
	Code    int         `json:"code,omitempty" example:"200"`
	Status  string      `json:"status,omitempty" example:"success"`
	Message string      `json:"message,omitempty" example:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

type ApiResponseModel struct {
	Code    int         `json:"code,omitempty" example:"200"`
	Status  string      `json:"status,omitempty" example:"success"`
	Message string      `json:"message,omitempty" example:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

type ApiErrorResponseModel struct {
	Status string             `json:"status" example:"internalServerError"`
	Error  ErrorResponseModel `json:"error"`
}

type ErrorResponseModel struct {
	Code    int                    `json:"code,omitempty" example:"400"`
	Message string                 `json:"message,omitempty" example:"error"`
	Errors  map[string]interface{} `json:"errors,omitempty"`
}

type ErrorValidationResponseModel struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Errors  interface{} `json:"errors"`
}

func StatusTxt(code int) string {
	return statusMessage[code]
}

func SuccessResponse(c echo.Context, code int, message string, data interface{}) error {
	return c.JSON(code, ApiSuccessResponseModel{
		Code:    code,
		Status:  StatusTxt(code),
		Message: message,
		Data:    data,
	})
}

func SuccessResponseList(c echo.Context, code int, message string, data interface{}, meta interface{}) error {
	return c.JSON(nethttp.StatusOK, ApiResponseModel{
		Code:    code,
		Status:  StatusTxt(code),
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// ErrorResponse function for custom error code and message
func ErrorResponseRest(c echo.Context, code int, errStr string) error {
	return c.JSON(code, ApiErrorResponseModel{
		Status: StatusTxt(code),
		Error: ErrorResponseModel{
			Code:    code,
			Message: errStr,
		},
	})
}

func ErrorValidationResponse(c echo.Context, code int, errStr string, err interface{}) error {
	return c.JSON(code, ErrorValidationResponseModel{
		Code:    code,
		Message: errStr,
		Errors:  err,
	})
}
