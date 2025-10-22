package files

import (
	"context"
	"errors"
	nethttp "net/http"
	"strings"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	xlog "bitbucket.org/Amartha/go-x/log"
	"github.com/labstack/echo/v4"
)

type filesHandler struct {
	fileSvc services.FileService
}

// New will initialize the files/ resources endpoint
func New(app *echo.Group, fileSvc services.FileService) {
	handler := filesHandler{
		fileSvc: fileSvc,
	}
	files := app.Group("/files")
	files.POST("/upload", handler.uploadFile)
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
// @Router /v2/files/upload [post]
func (h *filesHandler) uploadFile(c echo.Context) error {
	ctx := context.Background()
	file, err := c.FormFile("files")
	if err != nil {
		err = models.GetErrMap(models.ErrKeyFilesRequired, "files can not empty")
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	// Check if the uploaded file is a CSV
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".csv") {
		err = models.GetErrMap(models.ErrKeyFilesMustCsv, "files must be .csv")
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	username := c.Request().Header.Get(models.CtxKeyNgmisHeader)
	if username == "" {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, errors.New("ngmis username is required"))
	}

	clientID := c.Request().Header.Get(models.ClientIdHeader)

	go func() {
		errUpload := h.fileSvc.UploadWalletTransaction(ctx, file, username, clientID)
		if errUpload != nil {
			xlog.Errorf(ctx, "failed to process wallet transaction: %v", errUpload)
		}
	}()

	return http.RestSuccessResponse(c, nethttp.StatusOK, models.NewFileOut(file.Filename, "processing"))
}
