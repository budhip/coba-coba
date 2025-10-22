package masterdata

import (
	"errors"
	nethttp "net/http"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/validation"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/labstack/echo/v4"
)

type masterDataHandler struct {
	masterDataSvc services.MasterDataService
}

// New transaction handler will initialize the order-types/ and transaction-types/ resources endpoint
func New(app *echo.Group, masterDataSvc services.MasterDataService) {
	handler := masterDataHandler{
		masterDataSvc: masterDataSvc,
	}

	apiOrderTypes := app.Group("/order-types")
	apiOrderTypes.POST("", handler.createOrderType)
	apiOrderTypes.PATCH("", handler.updateOrderType)
	apiOrderTypes.GET("", handler.getAllOrderType)
	apiOrderTypes.GET("/:orderTypeCode", handler.getOrderType)

	apiTransactionTypes := app.Group("/transaction-types")
	apiTransactionTypes.GET("", handler.getAllTransactionType)
	apiTransactionTypes.GET("/:transactionTypeCode", handler.getTransactionType)

	apiVATConfigs := app.Group("/vat-configs")
	apiVATConfigs.GET("", handler.getAllVatConfig)
	apiVATConfigs.PATCH("", handler.upsertVatConfig)
}

// getAllOrderType API get all order type
// @Summary Get all data order type
// @Description Get all data order type
// @Tags OrderType
// @Accept  json
// @Produce  json
// @Success 200 {object} http.RestTotalRowResponseModel
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/order-types [get]
func (h *masterDataHandler) getAllOrderType(c echo.Context) error {
	var queryFilter models.FilterMasterData

	err := c.Bind(&queryFilter)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	ots, err := h.masterDataSvc.GetAllOrderType(c.Request().Context(), queryFilter)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	var data []models.OrderTypeOut
	for _, v := range ots {
		data = append(data, v.ToResponse())
	}

	return http.RestSuccessResponseListWithTotalRows(c, data, len(data))
}

// getAllTransactionType API get all transaction type
// @Summary Get all data transaction type
// @Description Get all data transaction type
// @Tags TransactionType
// @Accept  json
// @Produce  json
// @Success 200 {object} http.RestTotalRowResponseModel
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/transaction-types [get]
func (h *masterDataHandler) getAllTransactionType(c echo.Context) error {
	var queryFilter models.FilterMasterData
	err := c.Bind(&queryFilter)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	ots, err := h.masterDataSvc.GetAllTransactionType(c.Request().Context(), queryFilter)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	var data []models.TransactionTypeOut
	for _, v := range ots {
		data = append(data, v.ToResponse())
	}

	return http.RestSuccessResponseListWithTotalRows(c, data, len(data))
}

// createOrderType API create order type
// @Summary Create data order type
// @Description Create data order type
// @Tags OrderType
// @Accept  json
// @Produce  json
// @Param body body models.OrderType true "body"
// @Success 201 {object} http.RestTotalRowResponseModel
// @Failure 400 {object} http.RestErrorResponseModel
// @Failure 422 {object} http.RestErrorValidationResponseModel
// @Failure 409 {object} http.RestErrorResponseModel
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/order-types [post]
func (h *masterDataHandler) createOrderType(c echo.Context) error {
	var req models.OrderType

	if err := c.Bind(&req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err := validation.ValidateStruct(&req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	err := h.masterDataSvc.CreateOrderType(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, common.ErrDataExist) {
			return http.RestErrorResponse(c, nethttp.StatusConflict, err)
		}
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusCreated, req.ToResponse())
}

// updateOrderType API update order type
// @Summary Create data order type
// @Description Create data order type
// @Tags OrderType
// @Accept  json
// @Produce  json
// @Param body body models.OrderType true "body"
// @Success 200 {object} http.RestTotalRowResponseModel
// @Failure 400 {object} http.RestErrorResponseModel
// @Failure 422 {object} http.RestErrorValidationResponseModel
// @Failure 409 {object} http.RestErrorResponseModel
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/order-types [post]
func (h *masterDataHandler) updateOrderType(c echo.Context) error {
	var req models.OrderType

	if err := c.Bind(&req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err := validation.ValidateStruct(&req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	err := h.masterDataSvc.UpdateOrderType(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, common.ErrDataNotFound) {
			return http.RestErrorResponse(c, nethttp.StatusNotFound, err)
		}
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusCreated, req.ToResponse())
}

// getOrderType API get order type detail
// @Summary Get order type detail
// @Description Get order type detail
// @Tags OrderType
// @Accept  json
// @Produce  json
// @Param 	orderTypeCode path string true "order type code"
// @Success 200 {object} models.OrderTypeOut
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/order-types/:orderTypeCode [get]
func (h *masterDataHandler) getOrderType(c echo.Context) error {
	orderTypeCode := c.Param("orderTypeCode")
	if orderTypeCode == "" {
		err := models.GetErrMap(models.ErrKeyOrderTypeCodeRequired, "orderTypeCode is missing")
		return http.RestErrorResponse(c, nethttp.StatusUnprocessableEntity, err)
	}

	orderType, err := h.masterDataSvc.GetOneOrderType(c.Request().Context(), orderTypeCode)
	if err != nil {
		if errors.Is(err, common.ErrDataNotFound) {
			return http.RestErrorResponse(c, nethttp.StatusNotFound, err)
		}
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, orderType.ToResponse())
}

// getTransactionType API get transaction type detail
// @Summary Get transaction type detail
// @Description Get transaction type detail
// @Tags TransactionType
// @Accept  json
// @Produce  json
// @Param 	transactionTypeCode path string true "transaction type code"
// @Success 200 {object} models.TransactionTypeOut
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/transaction-types/:transactionTypeCode [get]
func (h *masterDataHandler) getTransactionType(c echo.Context) error {
	transactionTypeCode := c.Param("transactionTypeCode")
	if transactionTypeCode == "" {
		err := models.GetErrMap(models.ErrKeyTransactionTypeCodeRequired, "transactionTypeCode is missing")
		return http.RestErrorResponse(c, nethttp.StatusUnprocessableEntity, err)
	}

	transactionType, err := h.masterDataSvc.GetOneTransactionType(c.Request().Context(), transactionTypeCode)
	if err != nil {
		if errors.Is(err, common.ErrDataNotFound) {
			return http.RestErrorResponse(c, nethttp.StatusNotFound, err)
		}
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, transactionType.ToResponse())
}

// getAllVatConfig API get all data vat config
// @Summary Get all data vat config
// @Description Get all data vat config
// @Tags VATConfig
// @Accept  json
// @Produce  json
// @Success 200 {object} http.RestTotalRowResponseModel
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/vat-configs [get]
func (h *masterDataHandler) getAllVatConfig(c echo.Context) error {
	vatConf, err := h.masterDataSvc.GetAllVATConfig(c.Request().Context())
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	var data []models.ConfigVatRevenueOut
	for _, v := range vatConf {
		data = append(data, v.ToResponse())
	}

	return http.RestSuccessResponseListWithTotalRows(c, data, len(data))
}

// upsertVatConfig API update vat config
// @Summary Update data vat config
// @Description Update data vat config
// @Tags VATConfig
// @Accept  json
// @Produce  json
// @Param payload body []models.ConfigVatRevenue true "body"
// @Success 200 {object} http.RestTotalRowResponseModel
// @Failure 400 {object} http.RestErrorResponseModel
// @Failure 422 {object} http.RestErrorValidationResponseModel
// @Failure 409 {object} http.RestErrorResponseModel
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/vat-configs [patch]
func (h *masterDataHandler) upsertVatConfig(c echo.Context) error {
	var req []models.ConfigVatRevenue

	if err := c.Bind(&req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	err := h.masterDataSvc.UpsertVATConfig(c.Request().Context(), req)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	var data []models.ConfigVatRevenueOut
	for _, v := range req {
		data = append(data, v.ToResponse())
	}

	return http.RestSuccessResponseListWithTotalRows(c, data, len(data))
}
