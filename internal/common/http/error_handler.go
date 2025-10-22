package http

import (
	nethttp "net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// HandleRepositoryError handles common repository errors
func HandleRepositoryError(c echo.Context, err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "not found") {
		return RestErrorResponse(c, nethttp.StatusNotFound, err)
	}

	return RestErrorResponse(c, nethttp.StatusInternalServerError, err)
}
