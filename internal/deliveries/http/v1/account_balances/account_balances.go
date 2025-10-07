package account_balances

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"
)

type accountBalanceHandler struct {
	balanceService services.BalanceService
}

func New(app fiber.Router, balanceService services.BalanceService) {
	ab := accountBalanceHandler{
		balanceService: balanceService,
	}

	endpoint := app.Group("account-balances")
	endpoint.Get("/:accountNumber", ab.getAccountBalance())
}

// getAccountBalance API get balance by account number pas format or t24 format
// @Summary Get balance by account number
// @Description Get balance by account number
// @Tags Balance
// @Accept  json
// @Produce  json
// @Param 	accountNumber path string true "account number"
// @Success 200 {object} models.DoGetAccountBalanceResponse
// @Failure 500 {object} http.RestErrorResponseModel
// @Router /v1/account-balances/:accountNumber [get]
func (ab accountBalanceHandler) getAccountBalance() fiber.Handler {
	return func(c *fiber.Ctx) error {
		req := new(models.DoGetAccountRequest)

		if err := c.ParamsParser(req); err != nil {
			return http.RestErrorResponse(c, fiber.StatusBadRequest, err)
		}

		result, err := ab.balanceService.Get(c.UserContext(), req.AccountNumber)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				return http.RestErrorResponse(c, fiber.StatusNotFound, err)
			}
			return http.RestErrorResponse(c, fiber.StatusInternalServerError, err)
		}

		return http.RestSuccessResponse(c, fiber.StatusOK, result.ToModelResponse())
	}
}
