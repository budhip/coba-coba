package models

import (
	"encoding/base64"
	"fmt"
	"mime/multipart"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"

	"github.com/shopspring/decimal"
)

const (
	kindReconHistory = "reconTool"

	// ReconTaskName is the name of the task that will be used in the queue
	ReconTaskName = "RECON_FILE"

	ReconHistoryStatusPending    = "PENDING"
	ReconHistoryStatusProcessing = "PROCESSING"
	ReconHistoryStatusSuccess    = "SUCCESS"
	ReconHistoryStatusFailed     = "FAILED"
)

type UploadReconFileRequest struct {
	OrderType       string                `json:"orderType" validate:"required,alpha,noStartEndSpaces" example:"TOPUP"`
	TransactionType string                `json:"transactionType" validate:"required,alpha,noStartEndSpaces" example:"TOPUP"`
	TransactionDate string                `json:"transactionDate" validate:"required,date" example:"2006-12-02"`
	ReconFile       *multipart.FileHeader `json:"reconFile" validate:"required" example:"csv file"`
}

type UploadReconFileResponse struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

type GetURLReconFileResponse struct {
	Kind          string `json:"kind"`
	ResultFileURL string `json:"resultFileUrl"`
}

func NewGetURLReconFileResponse(resultFileURL string) *GetURLReconFileResponse {
	return &GetURLReconFileResponse{
		Kind:          "reconToolResultUrl",
		ResultFileURL: resultFileURL,
	}
}

func NewUploadReconFileResponse() *UploadReconFileResponse {
	return &UploadReconFileResponse{
		Kind:    "reconTool",
		Message: "Processing",
	}
}

type ReconToolHistory struct {
	ID               int
	OrderType        string
	TransactionType  string
	TransactionDate  *time.Time
	ResultFilePath   string
	UploadedFilePath string
	Status           string
	CreatedAt        *time.Time
	UpdatedAt        *time.Time
}

func (rth ReconToolHistory) GetCursor() string {
	offsetBytes := []byte(rth.CreatedAt.Format(time.RFC3339Nano))
	return base64.StdEncoding.EncodeToString(offsetBytes)
}

func (rth ReconToolHistory) ToModelResponse() DoGetReconToolHistoryResponse {
	return DoGetReconToolHistoryResponse{
		Kind:             kindReconHistory,
		ID:               fmt.Sprint(rth.ID),
		OrderType:        rth.OrderType,
		TransactionType:  rth.TransactionType,
		TransactionDate:  rth.TransactionDate.In(common.GetLocation()).Format(common.DateFormatYYYYMMDD),
		ResultFilePath:   rth.ResultFilePath,
		UploadedFilePath: rth.UploadedFilePath,
		Status:           rth.Status,
		ReconDate:        rth.CreatedAt.In(common.GetLocation()).Format(common.DateFormatYYYYMMDDWithTime),
		CreatedAt:        rth.CreatedAt.In(common.GetLocation()).Format(common.DateFormatYYYYMMDDWithTime),
		UpdatedAt:        rth.UpdatedAt.In(common.GetLocation()).Format(common.DateFormatYYYYMMDDWithTime),
	}
}

type CreateReconToolHistoryIn struct {
	OrderType        string
	TransactionType  string
	TransactionDate  string
	UploadedFilePath string
	Status           string
}

type ReconPublisher struct {
	ID   string `json:"id"`
	Task string `json:"task"`
}

type ReconToolHistoryFilterOptions struct {
	OrderType       string
	TransactionType string
	StartReconDate  *time.Time
	EndReconDate    *time.Time

	// Pagination filter
	Limit           int
	AscendingOrder  bool
	AfterCreatedAt  *time.Time
	BeforeCreatedAt *time.Time
}

type DoGetListReconToolHistoryRequest struct {
	OrderType       string `query:"orderType" example:"TOPUP"`
	TransactionType string `query:"transactionType" example:"TOPUP"`
	StartReconDate  string `query:"startReconDate" example:"2023-01-01"`
	EndReconDate    string `query:"endReconDate" example:"2023-01-07"`
	Limit           int    `query:"limit" example:"10"`
	NextCursor      string `query:"nextCursor" example:"abc"`
	PrevCursor      string `query:"prevCursor" example:"cba"`
}

type DoGetReconToolHistoryResponse struct {
	Kind             string `json:"kind" example:"reconTool"`
	ID               string `json:"id" example:"1"`
	OrderType        string `json:"orderType" example:"TOPUP"`
	TransactionType  string `json:"transactionType" example:"TOPUP"`
	TransactionDate  string `json:"transactionDate" example:"2023-10-25 08:08:26"`
	ResultFilePath   string `json:"resultFilePath" example:"/tmp/result.csv"`
	UploadedFilePath string `json:"uploadedFilePath" example:"/tmp/uploaded.csv"`
	Status           string `json:"status" example:"active"`
	ReconDate        string `json:"reconDate" example:"2023-10-25 08:08:26"`
	CreatedAt        string `json:"createdAt" example:"2006-01-02 15:04:05"`
	UpdatedAt        string `json:"updatedAt" example:"2006-01-02 15:04:05"`
}

func (req DoGetListReconToolHistoryRequest) ToFilterOpts() (*ReconToolHistoryFilterOptions, error) {
	opts := &ReconToolHistoryFilterOptions{
		OrderType:       req.OrderType,
		TransactionType: req.TransactionType,
		Limit:           req.Limit,
	}

	if req.Limit < 0 {
		return nil, GetErrMap(ErrKeyLimitMustBeGreaterThanZero)
	}

	if req.StartReconDate == "" || req.EndReconDate == "" {
		if req.StartReconDate != "" || req.EndReconDate != "" {
			return nil, GetErrMap(ErrKeyStartDateAndEndDateRequiredIfOneIsFilled)
		}
	} else {
		startDate, err := common.ParseStringToDatetime(common.DateFormatYYYYMMDD, req.StartReconDate)
		if err != nil {
			return nil, GetErrMap(ErrKeyInvalidFormatDate, fmt.Sprintf("date %s format must be YYYY-MM-DD", req.StartReconDate))
		}
		opts.StartReconDate = &startDate

		endDate, err := common.ParseStringToDatetime(common.DateFormatYYYYMMDD, req.EndReconDate)
		if err != nil {
			return nil, GetErrMap(ErrKeyInvalidFormatDate, fmt.Sprintf("date %s format must be YYYY-MM-DD", req.EndReconDate))
		}
		opts.EndReconDate = &endDate

		if startDate.After(endDate) {
			return nil, GetErrMap(ErrKeyStartDateIsAfterEndDate)
		}
	}

	if req.Limit == 0 {
		// default limit
		opts.Limit = 10
	}

	// use over-fetch limit for check next page exists or not
	opts.Limit += 1

	// forward pagination
	if req.NextCursor != "" {
		afterTime, err := decodeReconToolHistoryCursor(req.NextCursor)
		if err != nil {
			return nil, err
		}
		opts.AfterCreatedAt = &afterTime
	}

	// backward pagination
	if req.NextCursor == "" && req.PrevCursor != "" {
		prevTime, err := decodeReconToolHistoryCursor(req.PrevCursor)
		if err != nil {
			return nil, err
		}
		opts.BeforeCreatedAt = &prevTime

		// reverse order
		opts.AscendingOrder = true
	}

	return opts, nil
}

func decodeReconToolHistoryCursor(cursor string) (decodedTime time.Time, err error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return decodedTime, fmt.Errorf("failed to parse offset string: %w", err)
	}

	decodedTime, err = time.Parse(time.RFC3339Nano, string(decodedBytes))
	if err != nil {
		return decodedTime, fmt.Errorf("failed to parse offset time: %w", err)
	}

	return decodedTime, nil
}

type StatusReconRecord int

var CSVHeaderReconRecord = []string{
	"identifier",
	"amount",
	"orderType",
	"transactionType",
	"transactionDate",
	"refNumber",
	"lenderId",
	"customerName",
	"reconDate",
	"match",
	"status",
}

const (
	StatusReconRecordNotChecked StatusReconRecord = iota
	StatusReconRecordNotExistsDBExistsCSV
	StatusReconRecordExistsDBNotExistsCSV
	StatusReconRecordMatch
)

func (s StatusReconRecord) Title() string {
	switch s {
	case StatusReconRecordNotChecked:
		return "Not Checked"
	case StatusReconRecordNotExistsDBExistsCSV:
		return "Not Exists in DB, Exists in CSV"
	case StatusReconRecordExistsDBNotExistsCSV:
		return "Exists in DB, Not Exists in CSV"
	case StatusReconRecordMatch:
		return "Match"
	default:
		return "Unknown"
	}
}

type ReconRecord struct {
	Identifier   string
	Amount       decimal.Decimal
	RefNumber    string
	CustomerName string
	LenderID     string
	Match        bool

	// PaymentDate is the date of the transaction in DD-MMM-YYYY format
	PaymentDate string

	Status StatusReconRecord
}

func (rr ReconRecord) ToCSVRow(rth ReconToolHistory) []string {
	return []string{
		rr.Identifier,
		rr.Amount.String(),
		rth.OrderType,
		rth.TransactionType,
		rr.PaymentDate,
		rr.RefNumber,
		rr.LenderID,
		rr.CustomerName,
		rth.CreatedAt.In(common.GetLocation()).Format(common.DateFormatYYYYMMDDWithTime),
		fmt.Sprint(rr.Match),
		rr.Status.Title(),
	}
}

func (rr ReconRecord) ToCSVRowWithErr(rth ReconToolHistory, err error) []string {
	return []string{
		rr.Identifier,
		rr.Amount.String(),
		rth.OrderType,
		rth.TransactionType,
		rr.PaymentDate,
		rr.RefNumber,
		rr.LenderID,
		rr.CustomerName,
		rth.CreatedAt.In(common.GetLocation()).Format(common.DateFormatYYYYMMDDWithTime),
		"-",
		err.Error(),
	}
}
