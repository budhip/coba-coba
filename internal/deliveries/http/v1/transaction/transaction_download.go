package transaction

import (
	"errors"
	"fmt"
	nethttp "net/http"
	"os"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/labstack/echo/v4"
)

// @Summary 	Get All transaction
// @Description Get All transaction
// @Tags 		Transactions
// @Accept		json
// @Produce		text/csv
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param   params query models.DoGetListTransactionRequest true "Get all transaction query parameters"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while get all transaction"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while get all transaction"
// @Router /v1/transaction/download [get]
func (th *transactionHandler) downloadTransaction(c echo.Context) error {
	queryFilter := new(models.DoGetListTransactionRequest)

	if err := c.Bind(queryFilter); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	opts, err := queryFilter.ToDownloadFilterOpts()
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	file, err := os.CreateTemp("", "")
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	err = th.transactionSrv.DownloadTransactionFileCSV(c.Request().Context(), models.DownloadTransactionRequest{
		Options: *opts,
		Writer:  file,
	})
	if err != nil {
		if errors.Is(err, common.ErrRowLimitDownloadExceed) {
			return http.RestErrorResponse(c, nethttp.StatusUnprocessableEntity, err)
		}

		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	err = file.Close()
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	err = c.File(file.Name())
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	fileName := fmt.Sprintf("transaction-%s.csv", opts.StartDate.Format(common.DateFormatYYYYMMDD))
	return http.CSVSuccessResponse(c, fileName)
}
