package files

import (
	"strings"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/gofiber/fiber/v2"
)

type filesHandler struct {
	fileSvc services.FileService
}

// New will initialize the files/ resources endpoint
func New(app fiber.Router, fileSvc services.FileService) {
	handler := filesHandler{
		fileSvc: fileSvc,
	}
	files := app.Group("/files")
	files.Post("/upload", handler.uploadFile())
}

// uploadFile API to upload transaction file
// @Summary Upload transaction file
// @Description Upload transaction from CSV file
// @Tags Files
// @Accept json
// @Produce json
// @Success 200 {object} models.FileOut "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while create account"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while create account"
// @Router /v1/files/upload [post]
func (h *filesHandler) uploadFile() fiber.Handler {
	return func(c *fiber.Ctx) error {
		file, err := c.FormFile("files")
		if err != nil {
			err = models.GetErrMap(models.ErrKeyFilesRequired, "files can not empty")
			return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
		}

		// Check if the uploaded file is a CSV
		if !strings.HasSuffix(strings.ToLower(file.Filename), ".csv") {
			err = models.GetErrMap(models.ErrKeyFilesMustCsv, "files must be .csv")
			return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
		}

		go h.fileSvc.Upload(c.UserContext(), file)

		return http.RestSuccessResponse(c, fiber.StatusOK, models.NewFileOut(file.Filename, "processing"))
	}
}
