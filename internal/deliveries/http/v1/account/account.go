package account

import (
	"errors"
	nethttp "net/http"
	"strings"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http/middleware"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/validation"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/labstack/echo/v4"
)

type accountHandler struct {
	accountService       services.AccountService
	balanceService       services.BalanceService
	walletAccountService services.WalletAccountService
}

// New account handler will initialize the account/ resources endpoint
func New(app *echo.Group,
	accountSrv services.AccountService,
	walletAccSrv services.WalletAccountService,
	balanceSrv services.BalanceService,
	m middleware.AppMiddleware) {
	ah := accountHandler{
		accountService:       accountSrv,
		balanceService:       balanceSrv,
		walletAccountService: walletAccSrv,
	}
	account := app.Group("/accounts")
	account.GET("/balances", ah.getTotalBalance)
	account.POST("", ah.createAccount, m.CheckRetryDLQ())
	account.GET("", ah.getAllAccount)
	account.GET("/:accountNumber", ah.getOneAccount)
	account.PATCH("/:accountNumber", ah.updateOneAccount)
	account.DELETE("/:accountNumber", ah.deleteAccount)
	account.GET("/:accountNumber/balances", ah.getAccountBalance)
	account.PATCH("/sub-category/:subCategoryCode", ah.updateAccountBySubCategory)

	// wallet feature
	account.POST("/:accountNumber/features", ah.createAccountFeature)
}

// @Summary 	Get All account
// @Description Get All account
// @Tags 		Accounts
// @Accept		json
// @Produce		json
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Success 200 {object} http.RestPaginationResponseModel[[]models.GetAccountResponse] "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while delete account"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while delete account"
// @Router /v1/accounts [get]
func (ah accountHandler) getAllAccount(c echo.Context) error {
	var queryFilter models.DoGetListAccountRequest

	err := c.Bind(&queryFilter)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err = validation.ValidateStruct(queryFilter); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	opts, err := queryFilter.ToFilterOpts()
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	accounts, total, err := ah.accountService.GetList(c.Request().Context(), *opts)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponseCursorPagination[models.GetAccountResponse](c, accounts, opts.Limit, total)
}

// @Summary 	Create Account
// @Description Create New Account
// @Tags 		Accounts
// @Accept		json
// @Produce		json
// @Param 	payload body models.DoCreateAccountRequest true "A JSON object containing create account payload"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Success 201 {object} models.DoCreateAccountResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while create account"
// @Failure 404 {object} http.RestErrorResponseModel "Data not found. This can happen if there is an data not found while create account"
// @Failure 422 {object} http.RestErrorValidationResponseModel{errors=[]validation.ErrorValidateResponse} "Validation error. This can happen if there is an error validation while create account"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while create account"
// @Router /v1/accounts/ [post]
func (ah accountHandler) createAccount(c echo.Context) error {
	req := new(models.DoCreateAccountRequest)

	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err := validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	res, err := ah.accountService.Create(c.Request().Context(), models.CreateAccount{
		AccountNumber:   req.AccountNumber,
		Name:            req.Name,
		OwnerID:         req.OwnerID,
		CategoryCode:    req.CategoryCode,
		SubCategoryCode: req.SubCategoryCode,
		ProductTypeName: req.ProductTypeName,
		EntityCode:      req.EntityCode,
		Currency:        req.Currency,
		AltId:           req.AltId,
		LegacyId:        req.LegacyId,
		Status:          req.Status,
		Metadata:        req.Metadata,
	})
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusCreated, res.ToCreateAccountResponse())
}

// @Summary 	Get one account by account number
// @Description Get one account detail by account number
// @Tags 		Accounts
// @Accept		json
// @Produce		json
// @Param 	accountNumber path string true "account identifier"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param   params query models.DoGetAccountRequest true "Get all account query parameters"
// @Success 200 {object} models.GetAccountResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while get account"
// @Failure 404 {object} http.RestErrorResponseModel "Data not found. This can happen if data not found while get account"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while get account"
// @Router /v1/accounts/{accountNumber} [get]
func (ah accountHandler) getOneAccount(c echo.Context) error {
	req := new(models.DoGetAccountRequest)

	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	result, err := ah.accountService.GetOneByAccountNumberOrLegacyId(c.Request().Context(), req.AccountNumber)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return http.RestErrorResponse(c, nethttp.StatusNotFound, err)
		}
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, result.ToModelResponse())
}

// @Summary 	Get total balance of all account
// @Description Get total balance of all account
// @Tags 		Accounts
// @Accept		json
// @Produce		json
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Success 200 {object} models.AccountsTotalBalanceResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while delete account"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while delete account"
// @Router /v1/accounts/balances [get]
func (ah accountHandler) getTotalBalance(c echo.Context) error {
	// Query params
	var queryFilter models.DoGetListAccountRequest
	err := c.Bind(&queryFilter)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}
	opts, err := queryFilter.ToFilterOpts()
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	// Service
	totalBalance, err := ah.accountService.GetTotalBalance(c.Request().Context(), *opts)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, models.NewAccountsTotalBalanceResponse(totalBalance))
}

// @Summary 	Get one account by account number
// @Description Get one account detail by account number
// @Tags 		Accounts
// @Accept		json
// @Produce		json
// @Param 	accountNumber path string true "account identifier"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Success 200 {object} models.DoGetAccountBalanceResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while get account"
// @Failure 404 {object} http.RestErrorResponseModel "Data not found. This can happen if data not found while get account"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while get account"
// @Router /v1/accounts/{accountNumber}/balances [get]
func (ah accountHandler) getAccountBalance(c echo.Context) error {
	req := new(models.DoGetAccountRequest)

	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	result, err := ah.balanceService.Get(c.Request().Context(), req.AccountNumber)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return http.RestErrorResponse(c, nethttp.StatusNotFound, err)
		}
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, result.ToModelResponse())
}

// @Summary 	Update account's data
// @Description Update account's data by account number
// @Tags 		Accounts
// @Accept		json
// @Produce		json
// @Param 	accountNumber path string true "account identifier"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param 	payload body models.UpdateAccountRequest true "A JSON object containing payload"
// @Success 200 {object} models.GetAccountResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while get account"
// @Failure 404 {object} http.RestErrorResponseModel "Data not found. This can happen if data not found while get account"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while get account"
// @Router /v1/accounts/{accountNumber} [patch]
func (ah accountHandler) updateOneAccount(c echo.Context) error {
	req := new(models.UpdateAccountRequest)
	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err := validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	in, err := req.TransformAndValidate()
	if err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	result, err := ah.accountService.Update(c.Request().Context(), in)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return http.RestErrorResponse(c, nethttp.StatusNotFound, err)
		}
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}
	return http.RestSuccessResponse(c, nethttp.StatusOK, result.ToModelResponse())
}

// @Summary 	Create account feature
// @Description create account features related to internal wallet
// @Tags 		Accounts
// @Accept		json
// @Produce		json
// @Param 	accountNumber path string true "account identifier"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param 	payload body models.CreateWalletReq true "A JSON object containing payload"
// @Success 200 {object} models.WalletResponse "Response indicates that the request succeeded and the resources has been created in system"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This happens due to incorrect format payload"
// @Failure 422 {object} http.RestErrorResponseModel "Unprocessable entity. This happens due to missing mandatory fields in payload"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This happens if there is an unexpected error while creating account feature"
// @Router /v1/accounts/{accountNumber}/features [post]
func (ah accountHandler) createAccountFeature(c echo.Context) error {
	req := new(models.CreateWalletReq)
	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err := validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	payload, err := req.TransformAndValidate()
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	res, err := ah.walletAccountService.CreateAccountFeature(c.Request().Context(), payload)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusCreated, res.ToModelResponse())
}

// @Summary 	Update account's data by sub category
// @Description Update account's data by sub category
// @Tags 		Accounts
// @Accept		json
// @Produce		json
// @Param 	accountNumber path string true "account identifier"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param 	payload body models.UpdateAccountBySubCategoryRequest true "A JSON object containing payload"
// @Success 200 {object} nil "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while update account"
// @Failure 422 {object} http.RestErrorResponseModel "Validation error. This can happen if payload failed to be validated"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while update account"
// @Router /v1/accounts/sub-category/:subCategoryCode [patch]
func (ah accountHandler) updateAccountBySubCategory(c echo.Context) error {
	req := new(models.UpdateAccountBySubCategoryRequest)

	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err := validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	in := req.TransformAndValidate()

	err := ah.accountService.UpdateBySubCategory(c.Request().Context(), in)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, nil)
}

// @Summary 	Delete account's data
// @Description Delete account's data by account number
// @Tags 		Accounts
// @Accept		json
// @Produce		json
// @Param 	accountNumber path string true "account identifier"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param   params query models.DoGetAccountRequest true "Get all account query parameters"
// @Success 204 "Empty response"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while get account"
// @Failure 404 {object} http.RestErrorResponseModel "Data not found. This can happen if data not found while get account"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while get account"
// @Router /v1/accounts/{accountNumber} [delete]
func (ah accountHandler) deleteAccount(c echo.Context) error {
	req := new(models.DoGetAccountRequest)

	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	err := ah.accountService.Delete(c.Request().Context(), req.AccountNumber)
	if err != nil {
		if errors.Is(err, common.ErrNoRowsAffected) {
			return http.RestErrorResponse(c, nethttp.StatusNotFound, err)
		}

		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusNoContent, nil)
}
