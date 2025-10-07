package wallettrx

import (
	"errors"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http/middleware"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/validation"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/timeout"
)

type walletTrxHandler struct {
	walletTrxService services.WalletTrxService
	accountService   services.AccountService
	cfg              config.Config
}

const (
	defaultTimeoutHandler = 15 * time.Second
)

// New wallet transaction handler will initialize the /wallet-transactions resources endpoint
func New(
	cfg config.Config,
	app fiber.Router,
	walletTrxService services.WalletTrxService,
	accountService services.AccountService,
	m middleware.AppMiddleware) {

	handler := walletTrxHandler{
		cfg:              cfg,
		walletTrxService: walletTrxService,
		accountService:   accountService,
	}

	timeoutHandler := defaultTimeoutHandler
	if cfg.TransactionConfig.HandlerTimeoutWalletTransaction > 0 {
		timeoutHandler = cfg.TransactionConfig.HandlerTimeoutWalletTransaction
	}

	transaction := app.Group("/wallet-transactions")
	transaction.Post("",
		timeout.NewWithContext(handler.createWalletTransaction, timeoutHandler))
	transaction.Patch("/:transactionId",
		timeout.NewWithContext(handler.updateStatusWalletTransaction, timeoutHandler))
}

// createWalletTransaction API create wallet transaction
// @Summary Create wallet transaction
// @Description Create wallet transaction
// @Tags WalletTransaction
// @Accept  json
// @Produce  json
// @Param 	payload body models.CreateWalletTransactionRequest true "A JSON object containing create transaction payload"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Success 201 {object} models.WalletTransactionResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while create transaction"
// @Failure 404 {object} http.RestErrorResponseModel "Data not found. This can happen if there is a data not found while create transaction"
// @Failure 422 {object} http.RestErrorValidationResponseModel{errors=[]validation.ErrorValidateResponse} "Validation error. This can happen if there is an error validation while create transaction"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while create transaction"
// @Router /wallet-transactions [post]
func (h *walletTrxHandler) createWalletTransaction(c *fiber.Ctx) (err error) {
	req := new(models.CreateWalletTransactionRequest)
	if err = c.BodyParser(req); err != nil {
		return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
	}

	if err = validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	headers := c.GetReqHeaders()
	req.ClientId = getClientId(headers)
	req.IdempotencyKey = getIdempotencyKey(headers)

	created, err := h.walletTrxService.CreateTransaction(c.UserContext(), *req)
	if err != nil {
		return http.RestErrorResponse(c, getHttpErrorStatusCode(err), err)
	}

	return http.RestSuccessResponse(c, fiber.StatusCreated, req.ToResponse(*created))
}

func getClientId(headers map[string][]string) string {
	clientId := headers[models.ClientIdHeader]
	if len(clientId) > 0 {
		return clientId[0]
	}
	return ""
}

func getIdempotencyKey(headers map[string][]string) string {
	idempotencyKey := headers[models.IdempotencyKeyHeader]
	if len(idempotencyKey) > 0 {
		return idempotencyKey[0]
	}

	return ""
}

// updateStatusWalletTransaction API to update status wallet transaction
// @Summary Update status wallet transaction
// @Description Update status wallet transaction
// @Tags WalletTransaction
// @Accept  json
// @Produce  json
// @Param 	payload body models.UpdateStatusWalletTransactionRequest true "A JSON object containing create transaction payload"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Success 200 {object} models.UpdateStatusWalletTransactionResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while update status transaction"
// @Failure 404 {object} http.RestErrorResponseModel "Data not found. This can happen if there is a data not found while update status transaction"
// @Failure 422 {object} http.RestErrorValidationResponseModel{errors=[]validation.ErrorValidateResponse} "Validation error. This can happen if there is an error validation while update status transaction"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while update status transaction"
// @Router /wallet-transactions/{id} [patch]
func (h *walletTrxHandler) updateStatusWalletTransaction(c *fiber.Ctx) (err error) {
	req := models.UpdateStatusWalletTransactionRequest{
		TransactionId: c.Params("transactionId"),
	}

	if err = c.BodyParser(&req); err != nil {
		return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
	}

	if err = validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	if err = req.TransformTransactionTime(); err != nil {
		return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
	}

	headers := c.GetReqHeaders()
	req.ClientId = getClientId(headers)

	walletTransaction, err := h.walletTrxService.ProcessReservedTransaction(c.UserContext(), req)
	if err != nil {
		var code = fiber.StatusInternalServerError
		if errors.Is(err, common.ErrTransactionNotReserved) {
			code = fiber.StatusConflict
		}
		return http.RestErrorResponse(c, code, err)
	}

	return http.RestSuccessResponse(c, fiber.StatusOK, req.ToResponse(*walletTransaction))
}
