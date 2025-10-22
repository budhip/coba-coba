package health

import (
	nethttp "net/http"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"

	"github.com/labstack/echo/v4"
)

type healthHandler struct{}

// New health handler will initialize the health/ resources endpoint
func New(app *echo.Group) {
	hh := healthHandler{}
	health := app.Group("/health")
	health.GET("", hh.healthCheck)
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
func (th healthHandler) healthCheck(c echo.Context) error {
	return http.RestSuccessResponse(c, nethttp.StatusOK, DoHealthCheckLivenessResponse{
		Kind:   "health",
		Status: "server is up and running",
	})
}
