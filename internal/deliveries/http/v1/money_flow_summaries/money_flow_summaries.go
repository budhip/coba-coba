package money_flow_summaries

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	nethttp "net/http"

	xlog "bitbucket.org/Amartha/go-x/log"

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
	api.PATCH("/:summaryID/activation", handler.updateActivationStatus)
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

	if len(summaries) > 0 {
		xlog.Info(c.Request().Context(), "[PAGINATION-DEBUG-DATA]",
			xlog.String("first_id", summaries[0].ID[:8]),
			xlog.String("last_id", summaries[len(summaries)-1].ID[:8]))
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
// @Description Update money flow summary by summary ID. At least one field must be provided for update. Status transitions are validated: PENDING→IN_PROGRESS, IN_PROGRESS→SUCCESS/FAILED/REJECTED. When status changes to IN_PROGRESS, requestedDate is auto-filled and papaTransactionId is required.
// @Tags 		MoneyFlowSummary
// @Accept		json
// @Produce		json
// @Param 		summaryID path string true "summary identifier"
// @Param		X-Secret-Key header string true "X-Secret-Key"
// @Param   	body body models.UpdateMoneyFlowSummaryRequest true "Update summary request body"
// @Success 	200 {object} models.UpdateMoneyFlowSummaryResponse "Response indicates that the summary has been updated successfully"
// @Failure 	400 {object} http.RestErrorResponseModel "Bad request error. This can happen if validation fails, no fields provided, or invalid status transition"
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

	// Call service to update summary (validation logic is in service layer)
	err := h.moneyFlowService.UpdateSummary(c.Request().Context(), summaryID, *req)
	if err != nil {
		// Check if it's a validation error (status transition, IN_PROGRESS requirements, etc.)
		errMsg := err.Error()
		if strings.Contains(errMsg, "transition") ||
			strings.Contains(errMsg, "required") ||
			strings.Contains(errMsg, "should not") ||
			strings.Contains(errMsg, "not allowed") {
			return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
		}
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

	// Get summary detail
	summary, err := h.moneyFlowService.GetSummaryDetailBySummaryID(c.Request().Context(), summaryID)
	if err != nil {
		return http.HandleRepositoryError(c, err)
	}

	// Generate filename
	timestamp := time.Now().Format("20060102_150405")
	paymentType := strings.ReplaceAll(summary.PaymentType, "_", "-")
	paymentType = strings.ToLower(paymentType)

	var filename string
	if req.RefNumber != "" {
		filename = fmt.Sprintf("transactions_%s_%s_ref-%s_%s.csv",
			paymentType, summaryID[:8], req.RefNumber, timestamp)
	} else {
		filename = fmt.Sprintf("transactions_%s_%s_%s.csv",
			paymentType, summaryID[:8], timestamp)
	}

	// Use bytes.Buffer to buffer entire response
	// DON'T write directly to c.Response().Writer yet
	var buffer bytes.Buffer

	// Execute download to buffer
	err = h.moneyFlowService.DownloadDetailedTransactionsBySummaryID(
		c.Request().Context(),
		models.DownloadDetailedTransactionsRequest{
			SummaryID: summaryID,
			RefNumber: req.RefNumber,
			Writer:    &buffer,
		},
	)

	if err != nil {
		// Error occurred - return proper error response
		// Client gets JSON error, NOT a corrupt CSV file
		xlog.Error(c.Request().Context(), "[DOWNLOAD-CSV-HANDLER] Failed to generate CSV",
			xlog.String("summary_id", summaryID),
			xlog.String("ref_number", req.RefNumber),
			xlog.Err(err))

		// Check specific error types
		if err == context.DeadlineExceeded || err == context.Canceled {
			return http.RestErrorResponse(c, nethttp.StatusRequestTimeout,
				fmt.Errorf("download timeout: request took too long, please try with refNumber filter or smaller date range"))
		}

		return http.HandleRepositoryError(c, err)
	}

	// SUCCESS: All data is ready in buffer
	// NOW we can safely set headers and send response
	xlog.Info(c.Request().Context(), "[DOWNLOAD-CSV-HANDLER] CSV generated successfully, sending to client",
		xlog.String("summary_id", summaryID),
		xlog.String("filename", filename),
		xlog.Int("buffer_size_bytes", buffer.Len()))

	c.Response().Header().Set(echo.HeaderContentType, "text/csv; charset=utf-8")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=%q", filename))
	c.Response().Header().Set("Content-Length", fmt.Sprintf("%d", buffer.Len())) // Set content length
	c.Response().WriteHeader(nethttp.StatusOK)

	// Write buffer to response in one shot
	_, err = buffer.WriteTo(c.Response().Writer)
	if err != nil {
		xlog.Error(c.Request().Context(), "[DOWNLOAD-CSV-HANDLER] Failed to write buffer to response",
			xlog.String("summary_id", summaryID),
			xlog.Err(err))
		// At this point headers already sent, but log the error
		return nil
	}

	xlog.Info(c.Request().Context(), "[DOWNLOAD-CSV-HANDLER] CSV download completed successfully",
		xlog.String("summary_id", summaryID),
		xlog.String("filename", filename))

	return nil
}

// @Summary 	Update money flow summary activation status
// @Description Update money flow summary activation status (active/inactive) by summary ID
// @Tags 		MoneyFlowSummary
// @Accept		json
// @Produce		json
// @Param 		summaryID path string true "summary identifier"
// @Param		X-Secret-Key header string true "X-Secret-Key"
// @Param   	body body models.UpdateActivationStatusRequest true "Update status request body"
// @Success 	200 {object} models.UpdateActivationStatusResponse "Response indicates that the status has been updated successfully"
// @Failure 	400 {object} http.RestErrorResponseModel "Bad request error"
// @Failure 	404 {object} http.RestErrorResponseModel "Data not found"
// @Failure 	500 {object} http.RestErrorResponseModel "Internal server error"
// @Router /v1/money-flow-summaries/{summaryID}/activation [patch]
func (h *moneyFlowSummariesHandler) updateActivationStatus(c echo.Context) error {
	summaryID := c.Param("summaryID")

	// Parse request body
	req := new(models.UpdateActivationStatusRequest)
	if err := c.Bind(req); err != nil {
		return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
	}

	// Update status
	err := h.moneyFlowService.UpdateActivationStatus(c.Request().Context(), summaryID, req.IsActive)
	if err != nil {
		// Check if it's a validation error
		errMsg := err.Error()
		if strings.Contains(errMsg, "only summaries with PENDING status") ||
			strings.Contains(errMsg, "cannot update activation status") {
			return http.RestErrorResponse(c, nethttp.StatusBadRequest, err)
		}
		return http.HandleRepositoryError(c, err)
	}

	// Return success response
	response := models.UpdateActivationStatusResponse{
		Kind:      constants.MoneyFlowKind,
		SummaryID: summaryID,
		IsActive:  req.IsActive,
		Message:   "Money flow summary status updated successfully",
	}

	return http.RestSuccessResponse(c, nethttp.StatusOK, response)
}
