package transaction

import (
	"errors"
	nethttp "net/http"
	"net/url"
	"strings"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/validation"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/labstack/echo/v4"
)

// publishTransaction API publish transaction
// @Summary Publish transaction to kafka
// @Description Publish transaction to kafka to create data transaction
// @Tags Transactions
// @Accept  json
// @Produce  json
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param 	payload body models.DoPublishTransactionRequest true "A JSON object containing publish transaction payload"
// @Success 201 {object} models.DoPublishTransactionResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while publish transaction"
// @Failure 422 {object} http.RestErrorValidationResponseModel{errors=[]validation.ErrorValidateResponse} "Validation error. This can happen if there is an error validation while publish transaction"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while publish transaction"
// @Router /v1/transaction/publish [post]
func (th *transactionHandler) publishTransaction(c echo.Context) error {
	req := new(models.DoPublishTransactionRequest)

	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err := validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	res, err := th.transactionSrv.PublishTransaction(c.Request().Context(), *req)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusCreated, res)
}

// @Summary 	Get All transaction
// @Description Get All transaction
// @Tags 		Transactions
// @Accept		json
// @Produce		json
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param   params query models.DoGetListTransactionRequest true "Get all transaction query parameters"
// @Success 200 {object} http.RestPaginationResponseModel[[]models.DoGetTransactionResponse] "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while get all transaction"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while get all transaction"
// @Router /v1/transaction [get]
func (th *transactionHandler) getAllTransaction(c echo.Context) error {
	queryFilter := new(models.DoGetListTransactionRequest)

	if err := c.Bind(queryFilter); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	opts, err := queryFilter.ToFilterOpts()
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	transactions, total, err := th.transactionSrv.GetAllTransaction(c.Request().Context(), *opts)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponseCursorPagination[models.DoGetTransactionResponse](c, transactions, opts.Limit, total)
}

// @Summary 	Get single transaction with criteria
// @Description Get single transaction with criteria
// @Tags 		Transactions
// @Accept		json
// @Produce		json
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param   transactionType path string true "transactionType"
// @Param   refNumber path string true "refNumber"
// @Success 200 {object} models.DoGetTransactionResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while get all transaction"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while get all transaction"
// @Router /v1/transaction/{transactionType}/{refNumber} [get]
func (th *transactionHandler) getByTypeAndRefNumber(c echo.Context) error {
	req := new(models.TransactionGetByTypeAndRefNumberRequest)
	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err := common.ValidateStruct(req); err != nil {
		return common.ErrorValidationResponse(c, nethttp.StatusUnprocessableEntity, "Validation Failed", err)
	}

	trxType, err := url.QueryUnescape(req.TransactionType)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}
	req.TransactionType = trxType

	resp, err := th.transactionSrv.GetByTransactionTypeAndRefNumber(c.Request().Context(), req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return http.RestErrorResponse(c, nethttp.StatusNotFound, err)
		}
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, resp.ToModelResponse())
}

// updateStatusReservedTransaction API to update status reserved transaction
// @Summary Update status reserved transaction
// @Description Update status reserved transaction
// @Tags Transaction
// @Accept  json
// @Produce  json
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param   transactionId path string true "transactionId"
// @Param 	payload body models.UpdateStatusReservedTransactionRequest true "A JSON object containing payload"
// @Success 200 {object} models.UpdateStatusReservedTransactionResponse "Response indicates that the request succeeded"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error"
// @Router /v1/transaction/{transactionId} [patch]
func (th *transactionHandler) updateStatusReservedTransaction(c echo.Context) error {
	reqBody := new(models.UpdateStatusReservedTransactionRequest)
	reqBody.TransactionId = c.Param("transactionId")
	if err := c.Bind(&reqBody); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}
	if err := validation.ValidateStruct(reqBody); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	var trx *models.Transaction
	var err error
	if reqBody.Status == models.TransactionRequestCommitStatus {
		clientId := c.Request().Header.Get(models.ClientIdHeader)
		trx, err = th.transactionSrv.CommitReservedTransaction(c.Request().Context(), reqBody.TransactionId, clientId)
	} else {
		trx, err = th.transactionSrv.CancelReservedTransaction(c.Request().Context(), reqBody.TransactionId)
	}

	if err != nil {
		statusCode := nethttp.StatusInternalServerError
		if errors.Is(models.GetErrMap(models.ErrKeyDataNotFound), err) {
			statusCode = nethttp.StatusNotFound
		} else if errors.Is(common.ErrTransactionNotReserved, err) {
			statusCode = nethttp.StatusConflict
		}
		return http.RestErrorResponse(c, statusCode, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, trx.ToUpdateStatusReservedTransactionResponse())
}

// @Summary 	Get transaction status count
// @Description Get transaction status count
// @Tags 		Transactions
// @Accept		json
// @Produce		json
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param   params query models.DoGetStatusCountTransactionRequest true "Get status count transaction query parameters"
// @Success 200 {object} models.DoGetStatusCountTransactionResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while get status count transaction"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while get status count transaction"
// @Router /v1/transaction/status-count [get]
func (th *transactionHandler) getTransactionStatusCount(c echo.Context) error {
	queryFilter := new(models.DoGetStatusCountTransactionRequest)

	if err := c.Bind(queryFilter); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	opts, threshold, err := queryFilter.ToFilterOpts()
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	statusCount, err := th.transactionSrv.GetStatusCount(c.Request().Context(), threshold, *opts)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, statusCount.ToResponse())
}

// @Summary 	Get repayment report (last 7 days)
// @Description Get aggregated repayment report from yesterday - 6 days before
// @Tags 		Transactions
// @Accept		json
// @Produce		json
// @Param		X-Secret-Key header string true "X-Secret-Key"
// @Success 200 {object} models.DoGetReportRepaymentResponse "Response indicates that the request succeeded and the resources have been fetched and transmitted in the message body"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while getting repayment report"
// @Router /v1/report/repayment [get]
func (th *transactionHandler) getReportRepaymentSummary(c echo.Context) error {
	summary, err := th.transactionSrv.GetReportRepayment(c.Request().Context())
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, models.ReportRepayments(summary).ToResponse())
}
