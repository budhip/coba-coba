package services_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestService_IsReportExist(t *testing.T) {
	testHelper := serviceTestHelper(t)
	type args struct {
		ctx        context.Context
		reportName models.ReportName
		reportDate time.Time
	}
	type mockData struct {
	}
	tests := []struct {
		name     string
		args     args
		mockData mockData
		doMock   func(args args, mockData mockData)
	}{
		{
			name: "success - exist return url",
			args: args{
				ctx:        context.Background(),
				reportName: "[TEST] reportName",
				reportDate: *common.CurrentTime(),
			},
			doMock: func(args args, mockData mockData) {
				gcsPayload := &models.CloudStoragePayload{
					Filename: fmt.Sprintf("%d%02d%02d.csv", args.reportDate.Year(), args.reportDate.Month(), args.reportDate.Day()),
					Path:     fmt.Sprintf("%s/%d/%d", args.reportName, args.reportDate.Year(), args.reportDate.Month()),
				}

				testHelper.mockGcs.EXPECT().IsObjectExist(gomock.AssignableToTypeOf(context.Background()), gcsPayload).Return(true, "URL")
			},
		},

		{
			name: "success - not exist",
			args: args{
				ctx:        context.Background(),
				reportName: "[TEST] reportName",
				reportDate: *common.CurrentTime(),
			},
			doMock: func(args args, mockData mockData) {
				gcsPayload := &models.CloudStoragePayload{
					Filename: fmt.Sprintf("%d%02d%02d.csv", args.reportDate.Year(), args.reportDate.Month(), args.reportDate.Day()),
					Path:     fmt.Sprintf("%s/%d/%d", args.reportName, args.reportDate.Year(), args.reportDate.Month()),
				}

				testHelper.mockGcs.EXPECT().IsObjectExist(gomock.AssignableToTypeOf(context.Background()), gcsPayload).Return(false, "")
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}
			isExist, url := testHelper.storageService.IsReportExist(tt.args.ctx, tt.args.reportName, tt.args.reportDate)
			if isExist {
				assert.NotEmpty(t, url)
			} else {
				assert.Empty(t, url)
			}
		})
	}
}
