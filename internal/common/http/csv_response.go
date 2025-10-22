package http

import (
	"fmt"

	"github.com/labstack/echo/v4"
)

func CSVSuccessResponse(c echo.Context, fileName string) error {
	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s", fileName))

	return nil
}
