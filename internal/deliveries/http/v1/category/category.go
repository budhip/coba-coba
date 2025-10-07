package category

import (
	"errors"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/validation"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/gofiber/fiber/v2"
)

type categoryHandler struct {
	categorySvc services.CategoryService
}

// New transaction handler will initialize the categories/ resources endpoint
func New(app fiber.Router, categorySvc services.CategoryService) {
	handler := categoryHandler{
		categorySvc: categorySvc,
	}
	api := app.Group("/categories")
	api.Post("/", handler.createCategory())
	api.Get("/", handler.getAllCategory())
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
func (h *categoryHandler) createCategory() fiber.Handler {
	return func(c *fiber.Ctx) error {
		req := new(models.CreateCategoryRequest)

		if err := c.BodyParser(req); err != nil {
			return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
		}

		if err := validation.ValidateStruct(req); err != nil {
			return http.RestErrorValidationResponse(c, err)
		}

		res, err := h.categorySvc.Create(c.UserContext(), models.CreateCategoryIn(*req))
		if err != nil {
			if errors.Is(err, common.ErrDataExist) {
				return http.RestErrorResponse(c, fiber.StatusConflict, err)
			}
			return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
		}

		return http.RestSuccessResponse(c, fiber.StatusCreated, res.ConvertToCategoryOut())
	}
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
func (h *categoryHandler) getAllCategory() fiber.Handler {
	return func(c *fiber.Ctx) error {
		res, err := h.categorySvc.GetAll(c.UserContext())
		if err != nil {
			return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
		}

		data := []models.CategoryOut{}
		for _, v := range *res {
			data = append(data, *v.ConvertToCategoryOut())
		}

		return http.RestSuccessResponseListWithTotalRows(c, data, len(data))
	}
}
