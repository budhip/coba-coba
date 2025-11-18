package finsnapshot

import (
	nethttp "net/http"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/labstack/echo/v4"
)

type finSnapshotHandler struct {
	transactionSrv services.TransactionService
}

// New finSnapshot handler will initialize the transaction/ resources endpoint
func New(app *echo.Group, transactionSrv services.TransactionService) {
	handler := finSnapshotHandler{transactionSrv}
	finSnapshot := app.Group("/fin-snapshot")
	finSnapshot.GET("/collect", handler.finSnapshotCollectRepayment)
}

// @Summary 	Get repayment report (last 7 days)
// @Description Get aggregated repayment report from yesterday - 6 days before
// @Tags 		Transactions
// @Accept		json
// @Produce		json
// @Param		X-Secret-Key header string true "X-Secret-Key"
// @Success 200 {object} models.DoGetReportRepaymentResponse "Response indicates that the request succeeded and the resources have been fetched and transmitted in the message body"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while getting repayment report"
// @Router /v1/fin-snapshot/collect [get]
func (fs *finSnapshotHandler) finSnapshotCollectRepayment(c echo.Context) error {
	summary, err := fs.transactionSrv.CollectRepayment(c.Request().Context())
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, summary.MapToFinSnapshot())
}
