package common

import (
	"github.com/gofiber/fiber/v2"
)

var statusMessage = map[int]string{
	fiber.StatusOK:                  "success",
	fiber.StatusBadRequest:          "badRequest",
	fiber.StatusUnauthorized:        "unAuthorized",
	fiber.StatusForbidden:           "forbidden",
	fiber.StatusNotFound:            "notFound",
	fiber.StatusInternalServerError: "internalServerError",
	fiber.StatusCreated:             "created",
	fiber.StatusConflict:            "duplicateTrx",
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

func SuccessResponse(c *fiber.Ctx, code int, message string, data interface{}) error {
	return c.Status(code).JSON(ApiSuccessResponseModel{
		Code:    code,
		Status:  StatusTxt(code),
		Message: message,
		Data:    data,
	})
}

func SuccessResponseList(c *fiber.Ctx, code int, message string, data interface{}, meta interface{}) error {
	return c.Status(fiber.StatusOK).JSON(ApiResponseModel{
		Code:    code,
		Status:  StatusTxt(code),
		Message: message,
		Data:    data,
		Meta:    meta,
	})
}

// ErrorResponse function for custom error code and message
func ErrorResponseRest(c *fiber.Ctx, code int, errStr string) error {
	return c.Status(code).JSON(ApiErrorResponseModel{
		Status: StatusTxt(code),
		Error: ErrorResponseModel{
			Code:    code,
			Message: errStr,
		},
	})
}

func ErrorValidationResponse(c *fiber.Ctx, code int, errStr string, err interface{}) error {
	return c.Status(code).JSON(ErrorValidationResponseModel{
		Code:    code,
		Message: errStr,
		Errors:  err,
	})
}
