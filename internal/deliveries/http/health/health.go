package health

import (
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"github.com/gofiber/fiber/v2"
)

type healthHandler struct{}

// New health handler will initialize the health/ resources endpoint
func New(app fiber.Router) {
	hh := healthHandler{}
	health := app.Group("/health")
	health.Get("/", hh.healthCheck())
}

type (
	DoHealthCheckLivenessResponse struct {
		Kind   string `json:"kind" example:"health"`
		Status string `json:"status" example:"server is up and running"`
	}
)

// healthCheck godoc
// @Summary 	Get the status of server
// @Description	Get the status of server
// @Accept		json
// @Produce		json
// @Success 200 {object} DoHealthCheckLivenessResponse "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Router /health [get]
func (th healthHandler) healthCheck() fiber.Handler {
	return func(c *fiber.Ctx) error {
		return http.RestSuccessResponse(c, fiber.StatusOK, DoHealthCheckLivenessResponse{
			Kind:   "health",
			Status: "server is up and running",
		})
	}
}
