package services

import (
	"context"
	"fmt"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

type StorageService interface {
	IsReportExist(ctx context.Context, reportName models.ReportName, reportDate time.Time) (isExist bool, url string)
}

type storage service

// IsReportExist will check on bucket wherever report is exist or not. If exist, will return report's url.
func (svc *storage) IsReportExist(ctx context.Context, reportName models.ReportName, reportDate time.Time) (isExist bool, url string) {
	gcsPayload := models.CloudStoragePayload{
		Filename: fmt.Sprintf("%d%02d%02d.csv", reportDate.Year(), reportDate.Month(), reportDate.Day()),
		Path:     fmt.Sprintf("%s/%d/%d", reportName, reportDate.Year(), reportDate.Month()),
	}

	return svc.srv.cloudStorage.IsObjectExist(ctx, &gcsPayload)
}
