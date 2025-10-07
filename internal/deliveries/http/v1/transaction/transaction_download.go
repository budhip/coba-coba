package transaction

import (
	"errors"
	"fmt"
	"os"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/gofiber/fiber/v2"
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
func (th *transactionHandler) downloadTransaction() fiber.Handler {
	return func(c *fiber.Ctx) error {
		queryFilter := new(models.DoGetListTransactionRequest)

		if err := c.QueryParser(queryFilter); err != nil {
			return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
		}

		opts, err := queryFilter.ToDownloadFilterOpts()
		if err != nil {
			return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
		}

		file, err := os.CreateTemp("", "")
		if err != nil {
			return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
		}

		err = th.transactionSrv.DownloadTransactionFileCSV(c.UserContext(), models.DownloadTransactionRequest{
			Options: *opts,
			Writer:  file,
		})
		if err != nil {
			if errors.Is(err, common.ErrRowLimitDownloadExceed) {
				return http.RestErrorResponse(c, fiber.StatusUnprocessableEntity, err)
			}

			return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
		}

		err = file.Close()
		if err != nil {
			return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
		}

		err = c.SendFile(file.Name())
		if err != nil {
			return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
		}

		fileName := fmt.Sprintf("transaction-%s.csv", opts.StartDate.Format(common.DateFormatYYYYMMDD))
		return http.CSVSuccessResponse(c, fileName)
	}
}
