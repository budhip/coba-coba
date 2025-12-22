package health

import (
	"errors"
	nethttp "net/http"
	"sync/atomic"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"github.com/labstack/echo/v4"
)

type HealthCheck struct {
	isShutdown atomic.Bool
}

func NewHealthCheck() *HealthCheck {
	return &HealthCheck{
		isShutdown: atomic.Bool{},
	}
}

func (d *HealthCheck) Route(g *echo.Group) {
	g.GET("", d.healthCheck)
	g.GET("/liveness", d.liveness)
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
func (h *HealthCheck) healthCheck(c echo.Context) error {
	if h.isShutdown.Load() {
		return http.RestErrorResponse(c, nethttp.StatusServiceUnavailable, errors.New("server is shutting down"))
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, DoHealthCheckLivenessResponse{
		Kind:   "health",
		Status: "server is up and running",
	})
}

// @Summary Liveness Check
// @Description Checking http service health and db connection
// @Tags Health
// @Produce json
// @Success 200 {object} response.SuccessModel "Success"
// @Failure 503 {object} response.ErrorModel "Service Unavailable"
// @Router /health [get]
func (h *HealthCheck) liveness(c echo.Context) error {
	return http.RestSuccessResponse(c, nethttp.StatusOK, DoHealthCheckLivenessResponse{
		Kind:   "health",
		Status: "server is up and running",
	})
}

func (h *HealthCheck) Shutdown() {
	h.isShutdown.Store(true)
}
