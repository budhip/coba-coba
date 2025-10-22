package order

import (
	nethttp "net/http"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http/middleware"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/validation"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/labstack/echo/v4"
)

type orderHandler struct {
	trxService services.TransactionService
}

// New order handler will initialize the order/ resources endpoint
func New(app *echo.Group, trxService services.TransactionService, m middleware.AppMiddleware) {
	handler := orderHandler{trxService}
	orders := app.Group("/orders")
	orders.POST("", handler.createOrder, m.CheckRetryDLQ())
}

// createOrder API create order that have multiple transactions
// @Summary Create order that have multiple transactions
// @Description Create order that have multiple transactions
// @Tags Order
// @Accept  json
// @Produce  json
// @Param 	payload body models.CreateOrderRequest true "A JSON object containing create transaction payload"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Success 201 {object} models.CreateOrderResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while create transaction"
// @Failure 404 {object} http.RestErrorResponseModel "Data not found. This can happen if there is a data not found while create transaction"
// @Failure 422 {object} http.RestErrorValidationResponseModel{errors=[]validation.ErrorValidateResponse} "Validation error. This can happen if there is an error validation while create transaction"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while create transaction"
// @Router /orders [post]
func (h *orderHandler) createOrder(c echo.Context) error {
	req := new(models.CreateOrderRequest)
	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err := validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	err := h.trxService.NewStoreBulkTransaction(c.Request().Context(), req.ToTransactionReqs())
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusCreated, req.ToCreateOrderResponse())
}
