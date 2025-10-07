package http

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

func CSVSuccessResponse(c *fiber.Ctx, fileName string) error {
	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment;filename=%s", fileName))

	return nil
}
