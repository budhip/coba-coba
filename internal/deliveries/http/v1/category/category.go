package category

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

type categoryHandler struct {
	categorySvc services.CategoryService
}

// New transaction handler will initialize the categories/ resources endpoint
func New(app *echo.Group, categorySvc services.CategoryService) {
	handler := categoryHandler{
		categorySvc: categorySvc,
	}
	api := app.Group("/categories")
	api.POST("", handler.createCategory)
	api.GET("", handler.getAllCategory)
}

// createCategory API create category
// @Summary Create data category
// @Description Create data category
// @Tags Categories
// @Accept  json
// @Produce  json
// @Param body body models.CreateCategoryRequest true "body"
// @Success 201 {object} http.RestTotalRowResponseModel
// @Failure 400 {object} http.RestErrorResponseModel
// @Failure 422 {object} http.RestErrorValidationResponseModel
// @Failure 409 {object} http.RestErrorResponseModel
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/categories [post]
func (h *categoryHandler) createCategory(c echo.Context) error {
	req := new(models.CreateCategoryRequest)

	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err := validation.ValidateStruct(req); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	res, err := h.categorySvc.Create(c.Request().Context(), models.CreateCategoryIn(*req))
	if err != nil {
		if errors.Is(err, common.ErrDataExist) {
			return http.RestErrorResponse(c, nethttp.StatusConflict, err)
		}
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusCreated, res.ConvertToCategoryOut())
}

// getAllCategory API get all category
// @Summary Get all data category
// @Description Get all data category
// @Tags Categories
// @Accept  json
// @Produce  json
// @Success 200 {object} http.RestTotalRowResponseModel
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/categories [get]
func (h *categoryHandler) getAllCategory(c echo.Context) error {
	res, err := h.categorySvc.GetAll(c.Request().Context())
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	var data []models.CategoryOut
	for _, v := range *res {
		data = append(data, *v.ConvertToCategoryOut())
	}

	return http.RestSuccessResponseListWithTotalRows(c, data, len(data))
}
