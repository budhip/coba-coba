package internalwallet

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/validation"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/gofiber/fiber/v2"
)

type internalWalletHandler struct {
	walletTrxService services.WalletTrxService
}

// New internal wallet handler will initialize the /internal-wallets resources endpoint
func New(app fiber.Router, walletTrxService services.WalletTrxService) {
	handler := internalWalletHandler{walletTrxService}

	transaction := app.Group("/internal-wallets")
	transaction.Get("/accounts/:accountNumber/transactions", handler.listTransactionByAccountNumber)
	transaction.Get("/accounts/transactions", handler.listTransaction)
}

// getAllTransaction will get all wallet transaction by accountNumber
// @Summary 	Get all wallet transaction by accountNumber
// @Description Get all wallet transaction by accountNumber
// @Tags 		Internal Wallet
// @Accept		json
// @Produce		json
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param   params query models.ListWalletTrxByAccountNumberRequest true "Get all internal wallet query parameters"
// @Success 200 {object} http.RestPaginationResponseModel[[]models.ListWalletTrxByAccountNumberResponse] "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while get all transaction"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while get all transaction"
// @Router /v1/internal-wallets/accounts/{accountNumber}/transactions [get]
func (h *internalWalletHandler) listTransactionByAccountNumber(c *fiber.Ctx) (err error) {
	req := new(models.ListWalletTrxByAccountNumberRequest)
	req.AccountNumber = c.Params("accountNumber")

	err = c.QueryParser(req)
	if err != nil {
		return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
	}

	if err = validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	opts, err := req.ToFilterOpts()
	if err != nil {
		return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
	}

	transactions, total, err := h.walletTrxService.List(c.UserContext(), *opts)
	if err != nil {
		return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
	}

	return http.RestSuccessResponseCursorPagination[models.ListWalletTrxByAccountNumberResponse](c, transactions, opts.Limit, total)
}

// getAllTransaction will get all wallet transaction
// @Summary 	Get all wallet transaction
// @Description Get all wallet transaction
// @Tags 		Internal Wallet
// @Accept		json
// @Produce		json
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param   params query models.ListWalletTrxRequest true "Get all internal wallet query parameters"
// @Success 200 {object} http.RestPaginationResponseModel[[]models.ListWalletTrxByAccountNumberResponse] "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while get all transaction"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while get all transaction"
// @Router /v1/internal-wallets/accounts/transactions [get]
func (h *internalWalletHandler) listTransaction(c *fiber.Ctx) (err error) {
	req := new(models.ListWalletTrxRequest)

	err = c.QueryParser(req)
	if err != nil {
		return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
	}

	if err = validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	opts, err := req.ToFilterOpts()
	if err != nil {
		return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
	}

	transactions, total, err := h.walletTrxService.List(c.UserContext(), *opts)
	if err != nil {
		return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
	}

	return http.RestSuccessResponseCursorPagination[models.ListWalletTrxByAccountNumberResponse](c, transactions, opts.Limit, total)
}
