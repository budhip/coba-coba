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

func (fs *finSnapshotHandler) finSnapshotCollectRepayment(c echo.Context) error {
	summary, err := fs.transactionSrv.CollectRepayment(c.Request().Context())
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, summary.MapToFinSnapshot())
}
