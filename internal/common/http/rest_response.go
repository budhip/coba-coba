package http

import (
	"errors"
	"net/http"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/hashicorp/go-multierror"
	"github.com/labstack/echo/v4"
)

type (
	RestErrorResponseModel struct {
		Status  string      `json:"status" example:"error"`
		Code    interface{} `json:"code"`
		Message string      `json:"message" example:"error"`
	}

	RestTotalRowResponseModel struct {
		Kind      string      `json:"kind" example:"collection"`
		Contents  interface{} `json:"contents"`
		TotalRows int         `json:"total_rows" example:"100"`
	}

	RestPaginationResponseModel[T any] struct {
		Kind       string           `json:"kind" example:"collection"`
		Contents   T                `json:"contents"`
		Pagination CursorPagination `json:"pagination"`
	}

	RestErrorValidationResponseModel struct {
		Status  string      `json:"status" example:"error"`
		Message string      `json:"message" example:"validation error"`
		Errors  interface{} `json:"errors"`
	}
)

func RestSuccessResponse(c echo.Context, code int, in interface{}) error {
	return c.JSON(code, in)
}

func RestSuccessResponseListWithTotalRows(c echo.Context, data interface{}, totalRows int) error {
	return c.JSON(http.StatusOK, RestTotalRowResponseModel{
		Kind:      "collection",
		Contents:  data,
		TotalRows: totalRows,
	})
}

func RestSuccessResponseCursorPagination[ModelResponse any, S ~[]E, E PaginateableContent[ModelResponse]](c echo.Context, data S, requestLimit, totalRows int) error {
	// we use over-fetch to make sure nextPage exists or not
	hasMorePages := len(data) > (requestLimit - 1)

	if len(data) > 0 {
		if hasMorePages {
			data = data[:len(data)-1]
		}

		if isBackward(c) {
			// reverse data
			for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
				data[i], data[j] = data[j], data[i]
			}
		}
	}

	contents := make([]ModelResponse, 0)
	for _, datum := range data {
		res := datum.ToModelResponse()
		if &res != nil {
			contents = append(contents, res)
		}
	}

	pagination := NewCursorPagination[ModelResponse](c, data, hasMorePages, totalRows)

	return c.JSON(http.StatusOK, RestPaginationResponseModel[[]ModelResponse]{
		Kind:       "collection",
		Contents:   contents,
		Pagination: pagination,
	})
}

func RestErrorResponse(c echo.Context, statusCode int, err error) error {
	res := RestErrorResponseModel{
		Status:  "error",
		Code:    statusCode,
		Message: err.Error(),
	}

	var echoErr *echo.HTTPError
	if errors.As(err, &echoErr) {
		res.Code = echoErr.Code
		res.Message = echoErr.Message.(string)
	}

	var data models.ErrorDetail
	if errors.As(err, &data) {
		res.Code = data.Code
		res.Message = data.ErrorMessage.Error()
	}
	return c.JSON(statusCode, res)
}

func RestErrorValidationResponse(c echo.Context, errors interface{}) error {
	res := RestErrorValidationResponseModel{
		Status:  "error",
		Message: common.ErrValidation.Error(),
	}
	if data, ok := errors.(*multierror.Error); ok {
		res.Errors = data.Errors
	}

	return c.JSON(http.StatusUnprocessableEntity, res)
}
