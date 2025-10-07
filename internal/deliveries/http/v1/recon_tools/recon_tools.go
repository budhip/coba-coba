package recontools

import (
	"errors"
	"strconv"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/validation"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/gofiber/fiber/v2"
)

type reconToolsHandler struct {
	reconSvc services.ReconService
}

// New will initialize the recon-tools/ resources endpoint
func New(app fiber.Router, reconSvc services.ReconService) {
	handler := reconToolsHandler{
		reconSvc: reconSvc,
	}
	recon := app.Group("/recon-tools")
	recon.Get("/", handler.getAllReconHistory())
	recon.Get("/:id/download", handler.getResultURLReconHistory())
	recon.Post("/upload", handler.reconToolsUpload())
}

// @Summary 	Get All Recon History
// @Description Get All Recon History
// @Tags 		ReconTools
// @Accept		json
// @Produce		json
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Success 200 {object} http.RestPaginationResponseModel[[]models.ReconToolHistory] "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while delete account"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while delete account"
// @Router /v1/recon-tools [get]
func (h *reconToolsHandler) getAllReconHistory() fiber.Handler {
	return func(c *fiber.Ctx) error {
		var queryFilter models.DoGetListReconToolHistoryRequest

		err := c.QueryParser(&queryFilter)
		if err != nil {
			return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
		}

		opts, err := queryFilter.ToFilterOpts()
		if err != nil {
			return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
		}

		reconHistories, total, err := h.reconSvc.GetListReconHistory(c.UserContext(), *opts)
		if err != nil {
			return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
		}

		return http.RestSuccessResponseCursorPagination[models.DoGetReconToolHistoryResponse](c, reconHistories, opts.Limit, total)
	}
}

// @Summary 	Get result file URL Recon History
// @Description Get result file URL Recon History
// @Tags 		ReconTools
// @Accept		json
// @Produce		json
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Success 200 {object} models.GetURLReconFileResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while delete account"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while delete account"
// @Router /v1/recon-tools/:id/download [get]
func (h *reconToolsHandler) getResultURLReconHistory() fiber.Handler {
	return func(c *fiber.Ctx) error {
		idParam := c.Params("id")
		id, err := strconv.ParseUint(idParam, 10, 64)
		if err != nil {
			return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
		}

		url, err := h.reconSvc.GetResultFileURL(c.UserContext(), id)
		if err != nil {
			if errors.Is(err, common.ErrFilePathEmpty) {
				return http.RestErrorResponse(c, fiber.StatusUnprocessableEntity, err)
			}
			return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
		}

		return http.RestSuccessResponse(c, fiber.StatusOK, models.NewGetURLReconFileResponse(url))
	}
}

// reconToolsUpload API to upload reconciliation template file
// @Summary Upload reconciliation template file
// @Description Upload reconciliation template file
// @Tags ReconTools
// @Accept json
// @Produce json
// @Success 200 {object} models.FileOut "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while create account"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while create account"
// @Router /v1/recon-tools/upload [post]
func (h *reconToolsHandler) reconToolsUpload() fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Request validation
		req := new(models.UploadReconFileRequest)
		if err := c.BodyParser(req); err != nil {
			return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
		}

		reconFile, _ := c.FormFile("reconFile")
		req.ReconFile = reconFile

		if err := validation.ValidateStruct(req); err != nil {
			return http.RestErrorValidationResponse(c, err)
		}

		if err := h.reconSvc.UploadReconTemplate(c.UserContext(), req); err != nil {
			return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
		}

		return http.RestSuccessResponse(c, fiber.StatusAccepted, models.NewUploadReconFileResponse())
	}
}
