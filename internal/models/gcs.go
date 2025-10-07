package models

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
)

type CloudStoragePayload struct {
	Filename string
	Path     string
}

func (c CloudStoragePayload) GetFilePath() string {
	return fmt.Sprintf("%s/%s", c.Path, c.Filename)
}

func NewCloudStoragePayload(input string) CloudStoragePayload {
	input = filepath.Clean(input)

	// Extract the directory and filename.
	path := filepath.Dir(input)
	filename := filepath.Base(input)

	// handle the special case where input might be just a filename.
	if strings.TrimSpace(path) == "." {
		path = "" // If there's no path, just set it to an empty string.
	}

	return CloudStoragePayload{Filename: filename, Path: path}
}

type WriteStreamResult struct {
	errCh <-chan error
	url   string
}

func NewWriteStreamResult(errCh <-chan error, url string) WriteStreamResult {
	return WriteStreamResult{errCh: errCh, url: url}
}

func (r WriteStreamResult) Wait() (string, error) {
	var errs *multierror.Error
	for e := range r.errCh {
		errs = multierror.Append(errs, e)
	}

	return r.url, errs.ErrorOrNil()
}

type ReportName string

const (
	TransactionReportName  ReportName = "transaction_report"
	BalanceReconReportName ReportName = "balance_recon"
	ReconToolFolderName    ReportName = "recon_tools"
)

var BALANCE_RECON_HEADER = []string{"accountNumber", "consumerBalance", "databaseBalance"}
var TRANSACTION_REPORT_HEADER = []string{"id", "transactionDate", "fromAccount", "fromNarrative", "toAccount", "toNarrative", "amount", "status", "method", "typeTransaction", "description", "refNumber", "createdAt", "updatedAt", "metadata", "transactionId"}

const CSV_SEPARATOR = ";"
