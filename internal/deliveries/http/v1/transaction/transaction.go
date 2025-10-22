package transaction

import (
	nethttp "net/http"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http/middleware"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/validation"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/labstack/echo/v4"
)

type transactionHandler struct {
	transactionSrv services.TransactionService
}

// New transaction handler will initialize the transaction/ resources endpoint
func New(app *echo.Group, transactionSrv services.TransactionService, m middleware.AppMiddleware) {
	handler := transactionHandler{transactionSrv}
	transaction := app.Group("/transaction")
	transaction.GET("", handler.getAllTransaction)
	transaction.GET("/status-count", handler.getTransactionStatusCount)
	transaction.GET("/download", handler.downloadTransaction)
	transaction.POST("/publish", handler.publishTransaction)
	transaction.POST("/report", handler.generateTransactionReport)
	transaction.GET("/:transactionType/:refNumber", handler.getByTypeAndRefNumber)
	transaction.PATCH("/:transactionId", handler.updateStatusReservedTransaction)

	transaction.POST("", handler.createTransaction)
	transaction.POST("/bulk", handler.createBulkTransaction)

	report := app.Group("/report")
	report.GET("/repayment", handler.getReportRepaymentSummary)

	transactions := app.Group("/transactions")
	transactions.POST("", handler.createTransaction)
	transactions.PATCH("/:transactionId", handler.updateStatusReservedTransaction)

	orders := transactions.Group("/orders")
	orders.POST("", handler.createOrderTransaction)
}

// createTransaction API create transaction
// @Summary Create data transaction
// @Description Create data from any transaction
// @Tags Transaction
// @Accept  json
// @Produce  json
// @Param 	payload body models.DoCreateTransactionRequest true "A JSON object containing create transaction payload"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Success 201 {object} models.DoGetTransactionResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while create transaction"
// @Failure 404 {object} http.RestErrorResponseModel "Data not found. This can happen if there is a data not found while create transaction"
// @Failure 422 {object} http.RestErrorValidationResponseModel{errors=[]validation.ErrorValidateResponse} "Validation error. This can happen if there is an error validation while create transaction"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while create transaction"
// @Router /transaction [post]
func (th *transactionHandler) createTransaction(c echo.Context) error {
	var err error

	req := new(models.DoCreateTransactionRequest)
	if err = c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err = validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	storeType := models.TransactionStoreProcessNormal
	if req.IsReserved {
		storeType = models.TransactionStoreProcessReserved
	}

	clientID := c.Request().Header.Get(models.ClientIdHeader)

	res, err := th.transactionSrv.StoreTransaction(c.Request().Context(), req.ToTransactionReq(), storeType, clientID)
	if err != nil {
		httpStatusCode := getHTTPStatusCode(err)
		return http.RestErrorResponse(c, httpStatusCode, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusCreated, res.ToModelResponse())
}

// createTransaction API create bulk transaction
// @Summary Create data transaction
// @Description Create data from any transaction
// @Tags Transaction
// @Accept  json
// @Produce  json
// @Success 201 {object} common.ApiSuccessResponseModel
// @Failure 400 {object} common.ApiErrorResponseModel
// @Failure 422 {object} common.ApiErrorResponseModel
// @Failure 500 {object} common.ApiErrorResponseModel
// @Router /transaction/bulk [post]
func (th *transactionHandler) createBulkTransaction(c echo.Context) error {
	var req []models.TransactionReq
	errParse := c.Bind(&req)
	if errParse != nil {
		return common.ErrorResponseRest(c, nethttp.StatusBadRequest, errParse.Error())
	}

	err := th.transactionSrv.StoreBulkTransaction(c.Request().Context(), req)
	if err != nil {
		return common.ErrorResponseRest(c, nethttp.StatusBadRequest, err.Error())
	}
	return common.SuccessResponse(c, nethttp.StatusCreated, "", nil)
}

// generateTransactionReport API will generate transaction report
// @Summary Generate transaction report
// @Description Generate transaction data as CSV and upload to GCS
// @Tags Transaction
// @Accept json
// @Produce json
// @Success 200 {object} common.ApiSuccessResponseModel
// @Failure 400 {object} common.ApiErrorResponseModel
// @Failure 500 {object} common.ApiErrorResponseModel
// @Router /transaction/report [post]
func (th *transactionHandler) generateTransactionReport(c echo.Context) error {
	url, err := th.transactionSrv.GenerateTransactionReport(c.Request().Context())
	if err != nil {
		return common.ErrorResponseRest(c, nethttp.StatusBadRequest, err.Error())
	}

	return common.SuccessResponseList(c, nethttp.StatusOK, "Successfully generate transaction report", url, nil)
}

// createOrderTransaction API create order transaction
// @Summary Create order transaction
// @Description Create order transaction
// @Tags Transaction
// @Accept  json
// @Produce  json
// @Param 	payload body models.CreateOrderRequest true "A JSON object containing create transaction payload"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Success 201 {object} models.DoGetTransactionResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while create transaction"
// @Failure 404 {object} http.RestErrorResponseModel "Data not found. This can happen if there is a data not found while create transaction"
// @Failure 422 {object} http.RestErrorValidationResponseModel{errors=[]validation.ErrorValidateResponse} "Validation error. This can happen if there is an error validation while create transaction"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while create transaction"
// @Router /transactions/orders [post]
func (th *transactionHandler) createOrderTransaction(c echo.Context) error {
	req := new(models.CreateOrderRequest)
	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err := validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	err := th.transactionSrv.NewStoreBulkTransaction(c.Request().Context(), req.ToTransactionReqs())
	if err != nil {
		httpStatusCode := getHTTPStatusCode(err)
		return http.RestErrorResponse(c, httpStatusCode, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusCreated, req.ToCreateOrderResponse())
}
