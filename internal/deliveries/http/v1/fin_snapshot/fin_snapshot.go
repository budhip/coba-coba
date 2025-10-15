package finsnapshot

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
	"github.com/gofiber/fiber/v2"
)

type finSnapshotHandler struct {
	transactionSrv services.TransactionService
}

// New finSnapshot handler will initialize the transaction/ resources endpoint
func New(app fiber.Router, transactionSrv services.TransactionService) {
	handler := finSnapshotHandler{transactionSrv}
	finSnapshot := app.Group("/fin-snapshot")
	finSnapshot.Get("/collect", handler.finSnapshotCollectRepayment())
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
func (fs *finSnapshotHandler) finSnapshotCollectRepayment() fiber.Handler {
	return func(c *fiber.Ctx) error {
		summary, err := fs.transactionSrv.CollectRepayment(c.UserContext())
		if err != nil {
			return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
		}

		return http.RestSuccessResponse(c, fiber.StatusOK, models.CollectRepayment(*summary).MapToFinSnapshot())
	}
}
