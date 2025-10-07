package report

import (
	"os"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	xlog "bitbucket.org/Amartha/go-x/log"

	"go.uber.org/mock/gomock"
)

type testReportHelper struct {
	mockCtrl               *gomock.Controller
	mockTransactionService *mock.MockTransactionService
	mockReconService       *mock.MockReconService
}

func reportTestHelper(t *testing.T) testReportHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTransactionService := mock.NewMockTransactionService(mockCtrl)
	mockReconService := mock.NewMockReconService(mockCtrl)

	Routes(mockTransactionService, mockReconService)

	return testReportHelper{
		mockCtrl:               mockCtrl,
		mockTransactionService: mockTransactionService,
		mockReconService:       mockReconService,
	}
}

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}
