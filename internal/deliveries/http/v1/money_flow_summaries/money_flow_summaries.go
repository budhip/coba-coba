package money_flow_summaries

import (
	"fmt"
	"os"
	"time"

	nethttp "net/http"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/constants"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http"
	"bitbucket.org/Amartha/go-fp-transaction/internal/common/validation"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services"

	"github.com/labstack/echo/v4"
)

type moneyFlowSummariesHandler struct {
	moneyFlowService services.MoneyFlowService
}

// New money flow summary handler will initialize the money-flow-summaries/ resources endpoint
func New(app *echo.Group, moneyFlowSvc services.MoneyFlowService) {
	handler := moneyFlowSummariesHandler{
		moneyFlowService: moneyFlowSvc,
	}
	api := app.Group("/money-flow-summaries")
	api.GET("", handler.getSummariesList)
	api.GET("/:summaryID", handler.getSummaryDetailBySummaryID)
	api.GET("/:summaryID/transactions", handler.getDetailedTransactionsBySummaryID)
	api.PATCH("/:summaryID", handler.updateSummary)
	api.GET("/:summaryID/transactions/download", handler.downloadDetailedTransactionsBySummaryID)
}

// getList API to get money flow summary with filters
// @Summary Get money flow summary list with filters
// @Description Get money flow summary list filtered by payment type, transaction date, and status
// @Tags MoneyFlowSummary
// @Accept json
// @Produce json
// @Param paymentType query string false "Payment type filter"
// @Param transactionSourceCreationDateStart query string false "Transaction source creation date start filter (YYYY-MM-DD)"
// @Param transactionSourceCreationDateEnd query string false "Transaction source creation date end filter (YYYY-MM-DD)"
// @Param status query string false "Money flow status filter"
// @Param limit query int false "Limit per page (default: 10)"
// @Param nextCursor query string false "Next cursor for pagination"
// @Param prevCursor query string false "Previous cursor for pagination"
// @Param X-Secret-Key header string true "X-Secret-Key"
// @Success 200 {object} models.GetMoneyFlowSummaryListResponse "Response indicates that the request succeeded"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error"
// @Router /v1/money-flow-summaries [get]
func (h *moneyFlowSummariesHandler) getSummariesList(c echo.Context) error {
	var queryFilter models.GetMoneyFlowSummaryRequest

	if err := c.Bind(&queryFilter); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	if err := validation.ValidateStruct(queryFilter); err != nil {
		return http.RestErrorValidationResponse(c, err)
	}

	opts, err := queryFilter.ToFilterOpts()
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	summaries, total, err := h.moneyFlowService.GetSummariesList(c.Request().Context(), *opts)
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	return http.RestSuccessResponseCursorPagination[models.MoneyFlowSummaryResponse](c, summaries, opts.Limit, total)
}

// @Summary 	Get summary detail by summary id
// @Description Get summary detail by summary id
// @Tags 		MoneyFlowSummary
// @Accept		json
// @Produce		json
// @Param 	summaryID path string true "summary identifier"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param   params query models.DoGetSummaryIDBySummaryIDRequest true "Get summary detail query parameters"
// @Success 200 {object} models.MoneyFlowSummaryBySummaryIDOut "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while get summary detail by summary id"
// @Failure 404 {object} http.RestErrorResponseModel "Data not found. This can happen if data not found while get summary detail by summary id"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while get summary detail by summary id"
// @Router /v1/money-flow-summaries/{summaryID} [get]
func (h *moneyFlowSummariesHandler) getSummaryDetailBySummaryID(c echo.Context) error {
	summaryID := c.Param("summaryID")

	result, err := h.moneyFlowService.GetSummaryDetailBySummaryID(c.Request().Context(), summaryID)
	if err != nil {
		return http.HandleRepositoryError(c, err)
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, result.ToModelResponse())
}

// @Summary 	Get detailed transactions by summary id
// @Description Get detailed list of transactions by summary id
// @Tags 		MoneyFlowSummary
// @Accept		json
// @Produce		json
// @Param 	summaryID path string true "summary identifier"
// @Param	X-Secret-Key header string true "X-Secret-Key"
// @Param   params query models.DoGetDetailedTransactionsBySummaryIDRequest true "Get detailed transactions query parameters"
// @Success 200 {object} http.RestPaginationResponseModel[[]models.DetailedTransactionResponse] "Response indicates that the request succeeded and the resources has been fetched and transmitted in the message body"
// @Failure 400 {object} http.RestErrorResponseModel "Bad request error. This can happen if there is an error while get detailed transactions by summary id"
// @Failure 404 {object} http.RestErrorResponseModel "Data not found. This can happen if data not found while get detailed transactions by summary id"
// @Failure 500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while get detailed transactions by summary id"
// @Router /v1/money-flow-summaries/{summaryID}/transactions [get]
func (h *moneyFlowSummariesHandler) getDetailedTransactionsBySummaryID(c echo.Context) error {
	req := new(models.DoGetDetailedTransactionsBySummaryIDRequest)

	summaryID := c.Param("summaryID")

	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	opts, err := req.ToFilterOpts()
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	transactions, total, err := h.moneyFlowService.GetDetailedTransactionsBySummaryID(c.Request().Context(), summaryID, *opts)
	if err != nil {
		return http.HandleRepositoryError(c, err)
	}

	return http.RestSuccessResponseCursorPagination[models.DetailedTransactionResponse](c, transactions, opts.Limit, total)
}

// @Summary 	Update money flow summary
// @Description Update money flow summary by summary ID. At least one field must be provided for update.
// @Tags 		MoneyFlowSummary
// @Accept		json
// @Produce		json
// @Param 		summaryID path string true "summary identifier"
// @Param		X-Secret-Key header string true "X-Secret-Key"
// @Param   	body body models.UpdateMoneyFlowSummaryRequest true "Update summary request body"
// @Success 	200 {object} models.UpdateMoneyFlowSummaryResponse "Response indicates that the summary has been updated successfully"
// @Failure 	400 {object} http.RestErrorResponseModel "Bad request error. This can happen if validation fails or no fields provided"
// @Failure 	404 {object} http.RestErrorResponseModel "Data not found. This can happen if summary ID not found"
// @Failure 	422 {object} http.RestErrorResponseModel "Unprocessable entity. This can happen if data format is invalid"
// @Failure 	500 {object} http.RestErrorResponseModel "Internal server error. This can happen if there is an error while updating summary"
// @Router /v1/money-flow-summaries/{summaryID} [patch]
func (h *moneyFlowSummariesHandler) updateSummary(c echo.Context) error {
	summaryID := c.Param("summaryID")

	// Parse request body
	req := new(models.UpdateMoneyFlowSummaryRequest)
	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	// Validate request
	if err := req.Validate(); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	// Call service to update summary
	err := h.moneyFlowService.UpdateSummary(c.Request().Context(), summaryID, *req)
	if err != nil {
		return http.HandleRepositoryError(c, err)
	}

	// Return success response
	response := models.UpdateMoneyFlowSummaryResponse{
		Kind:      constants.MoneyFlowKind,
		SummaryID: summaryID,
		Message:   "Money flow summary updated successfully",
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, response)

}

// @Summary 	Download detailed transactions by summary id as CSV
// @Description Download detailed list of transactions by summary id in CSV format
// @Tags 		MoneyFlowSummary
// @Accept		json
// @Produce		text/csv
// @Param 		summaryID path string true "summary identifier"
// @Param		X-Secret-Key header string true "X-Secret-Key"
// @Param   	refNumber query string false "Reference number filter"
// @Success 	200 {file} file "CSV file"
// @Failure 	400 {object} http.RestErrorResponseModel "Bad request error"
// @Failure 	404 {object} http.RestErrorResponseModel "Data not found"
// @Failure 	500 {object} http.RestErrorResponseModel "Internal server error"
// @Router /v1/money-flow-summaries/{summaryID}/transactions/download [get]
func (h *moneyFlowSummariesHandler) downloadDetailedTransactionsBySummaryID(c echo.Context) error {
	req := new(models.DoDownloadDetailedTransactionsBySummaryIDRequest)

	summaryID := c.Param("summaryID")
	req.SummaryID = summaryID

	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	// Create temporary file
	file, err := os.CreateTemp("", "detailed_transactions_*.csv")
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}
	defer os.Remove(file.Name()) // Clean up temporary file

	// Download to temporary file
	err = h.moneyFlowService.DownloadDetailedTransactionsBySummaryID(
		c.Request().Context(),
		models.DownloadDetailedTransactionsRequest{
			SummaryID: summaryID,
			RefNumber: req.RefNumber,
			Writer:    file,
		},
	)
	if err != nil {
		file.Close()
		return http.HandleRepositoryError(c, err)
	}

	// Close file before serving
	err = file.Close()
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	// Serve file
	err = c.File(file.Name())
	if err != nil {
		return http.RestErrorResponse(c, nethttp.StatusInternalServerError, err)
	}

	// Generate filename
	timestamp := time.Now().Format("20060102")
	filename := fmt.Sprintf("detailed_transactions_%s_%s.csv", summaryID, timestamp)

	return http.CSVSuccessResponse(c, filename)
}
