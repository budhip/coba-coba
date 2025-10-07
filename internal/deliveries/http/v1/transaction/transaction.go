package transaction

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http/middleware"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/validation"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/gofiber/fiber/v2"
)

type transactionHandler struct {
	transactionSrv services.TransactionService
}

// New transaction handler will initialize the transaction/ resources endpoint
func New(app fiber.Router, transactionSrv services.TransactionService, m middleware.AppMiddleware) {
	handler := transactionHandler{transactionSrv}
	transaction := app.Group("/transaction")
	transaction.Get("/", handler.getAllTransaction())
	transaction.Get("/status-count", handler.getTransactionStatusCount())
	transaction.Get("/download", handler.downloadTransaction())
	transaction.Post("/publish", handler.publishTransaction())
	transaction.Post("/report", handler.generateTransactionReport())
	transaction.Get("/:transactionType/:refNumber", handler.getByTypeAndRefNumber())
	transaction.Patch("/:transactionId", handler.updateStatusReservedTransaction())

	transaction.Post("/", m.CheckIdempotentRequest(), handler.createTransaction())
	transaction.Post("/bulk", handler.createBulkTransaction())

	report := app.Group("/report")
	report.Get("/repayment", handler.getReportRepaymentSummary())

	transactions := app.Group("/transactions")
	transactions.Post("/", m.CheckIdempotentRequest(), handler.createTransaction())
	transactions.Patch("/:transactionId", handler.updateStatusReservedTransaction())

	orders := transactions.Group("/orders")
	orders.Post("/", handler.createOrderTransaction)
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
func (th *transactionHandler) createTransaction() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var err error

		req := new(models.DoCreateTransactionRequest)
		if err = c.BodyParser(req); err != nil {
			return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
		}

		if err = validation.ValidateStruct(req); err != nil {
			return http.RestErrorValidationResponse(c, err)
		}

		storeType := models.TransactionStoreProcessNormal
		if req.IsReserved {
			storeType = models.TransactionStoreProcessReserved
		}

		clientID := c.Get(models.ClientIdHeader)

		res, err := th.transactionSrv.StoreTransaction(c.UserContext(), req.ToTransactionReq(), storeType, clientID)
		if err != nil {
			httpStatusCode := getHTTPStatusCode(err)
			return http.RestErrorResponse(c, httpStatusCode, err)
		}

		return http.RestSuccessResponse(c, fiber.StatusCreated, res.ToModelResponse())
	}
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
func (th *transactionHandler) createBulkTransaction() fiber.Handler {
	return func(c *fiber.Ctx) error {
		req := []models.TransactionReq{}
		errParse := c.BodyParser(&req)
		if errParse != nil {
			return common.ErrorResponseRest(c, fiber.StatusBadRequest, errParse.Error())
		}

		err := th.transactionSrv.StoreBulkTransaction(c.UserContext(), req)
		if err != nil {
			return common.ErrorResponseRest(c, fiber.StatusBadRequest, err.Error())
		}
		return common.SuccessResponse(c, fiber.StatusCreated, "", nil)
	}
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
func (th *transactionHandler) generateTransactionReport() fiber.Handler {
	return func(c *fiber.Ctx) error {
		url, err := th.transactionSrv.GenerateTransactionReport(c.UserContext())
		if err != nil {
			return common.ErrorResponseRest(c, fiber.StatusBadRequest, err.Error())
		}

		return common.SuccessResponseList(c, fiber.StatusOK, "Successfully generate transaction report", url, nil)
	}
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
func (th *transactionHandler) createOrderTransaction(c *fiber.Ctx) (err error) {
	req := new(models.CreateOrderRequest)
	if err = c.BodyParser(req); err != nil {
		return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
	}

	if err = validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	err = th.transactionSrv.NewStoreBulkTransaction(c.UserContext(), req.ToTransactionReqs())
	if err != nil {
		httpStatusCode := getHTTPStatusCode(err)
		return http.RestErrorResponse(c, httpStatusCode, err)
	}

	return http.RestSuccessResponse(c, fiber.StatusCreated, req.ToCreateOrderResponse())
}
