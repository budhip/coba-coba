package http

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

// HandleRepositoryError handles common repository errors
func HandleRepositoryError(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	if strings.Contains(err.Error(), "not found") {
		return RestErrorResponse(c, fiber.StatusNotFound, err)
	}

	return RestErrorResponse(c, fiber.StatusInternalServerError, err)
}
