package transaction

import (
	"errors"
	"fmt"
	nethttp "net/http"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	"github.com/labstack/echo/v4"
)

type transactionV2Handler struct {
	transactionSrv services.TransactionService
}

// New transaction handler will initialize the transaction/ resources endpoint
func New(app *echo.Group, transactionSrv services.TransactionService) {
	handler := transactionV2Handler{transactionSrv}
	transaction := app.Group("/transaction")
	transaction.GET("/download", handler.downloadTransaction)
}

// @Summary 	Get All transaction
// @Description Get All transaction
// @Tags 		Transactions
// @Accept		json
// @Produce		text/csv
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param   params query models.DoGetListTransactionRequest true "Get all transaction query parameters"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while get all transaction"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while get all transaction"
// @Router /v2/transaction/download [get]
func (th *transactionV2Handler) downloadTransaction(c echo.Context) error {
	queryFilter := new(models.DoGetListTransactionRequest)

	if err := c.Bind(queryFilter); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	opts, err := queryFilter.ToDownloadFilterOpts()
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	err = th.transactionSrv.DownloadV2TransactionFileCSV(c.Request().Context(), models.DownloadTransactionRequest{
		Options: *opts,
		Writer:  nil,
	})
	if err != nil {
		if errors.Is(err, common.ErrRowLimitDownloadExceed) {
			return http.RestErrorResponse(c, nethttp.StatusUnprocessableEntity, err)
		}

		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	fileName := fmt.Sprintf("transaction-%s.csv", opts.StartDate.Format(common.DateFormatYYYYMMDD))
	return http.CSVSuccessResponse(c, fileName)
}
