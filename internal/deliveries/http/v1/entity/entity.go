package entity

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

type entityHandler struct {
	entitySvc services.EntityService
}

// New transaction handler will initialize the entities/ resources endpoint
func New(app *echo.Group, entitySvc services.EntityService) {
	handler := entityHandler{
		entitySvc: entitySvc,
	}
	api := app.Group("/entities")
	api.POST("", handler.createEntity)
	api.GET("", handler.getAllEntity)
}

// createEntity API create entity
// @Summary Create data entity
// @Description Create data entity
// @Tags Entities
// @Accept  json
// @Produce  json
// @Param body body models.CreateEntityRequest true "body"
// @Success 201 {object} http.RestTotalRowResponseModel
// @Failure 400 {object} http.RestErrorResponseModel
// @Failure 422 {object} http.RestErrorValidationResponseModel
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/entities [post]
func (h *entityHandler) createEntity(c echo.Context) error {
	req := new(models.CreateEntityRequest)

	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err := validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	res, err := h.entitySvc.Create(c.Request().Context(), models.CreateEntityIn(*req))
	if err != nil {
		if errors.Is(err, common.ErrDataExist) {
			return http.RestErrorResponse(c, nethttp.StatusConflict, err)
		}
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusCreated, res.ToResponse())
}

// getAllEntity API get all entity
// @Summary Get all data entity
// @Description Get all data entity
// @Tags Entities
// @Accept  json
// @Produce  json
// @Success 200 {object} http.RestTotalRowResponseModel{contents=[]models.EntityOut}
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/entities [get]
func (h *entityHandler) getAllEntity(c echo.Context) error {
	res, err := h.entitySvc.GetAll(c.Request().Context())
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	var data []models.EntityOut
	for _, v := range *res {
		data = append(data, *v.ToResponse())
	}

	return http.RestSuccessResponseListWithTotalRows(c, data, len(data))
}
