package subcategory

import (
	"errors"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/validation"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/gofiber/fiber/v2"
)

type subCategoryHandler struct {
	subCatSvc services.SubCategoryService
}

// New transaction handler will initialize the sub-categories/ resources endpoint
func New(app fiber.Router, subCatSvc services.SubCategoryService) {
	handler := subCategoryHandler{
		subCatSvc: subCatSvc,
	}
	api := app.Group("/sub-categories")
	api.Post("/", handler.createSubCategory())
	api.Get("/", handler.getAllSubCategory())
}

// createSubCategory API create sub category
// @Summary Create data sub category
// @Description Create data sub category
// @Tags Sub Categories
// @Accept  json
// @Produce  json
// @Param body body models.CreateSubCategoryRequest true "body"
// @Success 201 {object} http.RestTotalRowResponseModel
// @Failure 400 {object} http.RestErrorResponseModel
// @Failure 404 {object} http.RestErrorResponseModel
// @Failure 409 {object} http.RestErrorResponseModel
// @Failure 422 {object} http.RestErrorValidationResponseModel
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/sub-categories [post]
func (h *subCategoryHandler) createSubCategory() fiber.Handler {
	return func(c *fiber.Ctx) error {
		req := new(models.CreateSubCategoryRequest)

		if err := c.BodyParser(req); err != nil {
			return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
		}

		if err := validation.ValidateStruct(req); err != nil {
			return http.RestErrorValidationResponse(c, err)
		}

		res, err := h.subCatSvc.Create(c.UserContext(), models.CreateSubCategory(*req))
		if err != nil {
			code := fiber.StatusInternalServerError
			if errors.Is(err, common.ErrDataNotFound) {
				code = fiber.StatusNotFound
			} else if errors.Is(err, common.ErrDataExist) {
				code = fiber.StatusConflict
			}
			return http.RestErrorResponse(c, code, err)
		}

		return http.RestSuccessResponse(c, fiber.StatusCreated, res.ToResponse())
	}
}

// getAllSubCategory API get sub category
// @Summary Get all data sub category
// @Description Get all data sub category
// @Tags Sub Categories
// @Accept  json
// @Produce  json
// @Success 200 {object} http.RestTotalRowResponseModel
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/sub-categories [get]
func (h *subCategoryHandler) getAllSubCategory() fiber.Handler {
	return func(c *fiber.Ctx) error {
		res, err := h.subCatSvc.GetAll(c.UserContext())
		if err != nil {
			return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
		}

		data := []models.SubCategoryOut{}
		for _, v := range *res {
			data = append(data, *v.ToResponse())
		}

		return http.RestSuccessResponseListWithTotalRows(c, data, len(data))
	}
}
