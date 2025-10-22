package order

import (
	"os"
	"testing"

	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http/middleware"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	mockRepo "bitbucket.org/Amartha/go-fp-transaction/internal/repositories/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	xlog "bitbucket.org/Amartha/go-x/log"

	"go.uber.org/mock/gomock"
)

type orderTestHelper struct {
	router         *echo.Echo
	mockCtrl       *gomock.Controller
	mockTrxService *mock.MockTransactionService
}

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}

func getOrderTestHelper(t *testing.T) orderTestHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockTrxService := mock.NewMockTransactionService(mockCtrl)
	mockCacheRepo := mockRepo.NewMockCacheRepository(mockCtrl)
	mockDlqProcessorService := mock.NewMockDLQProcessorService(mockCtrl)

	app := echo.New()

	v1Group := app.Group("/api/v1")
	app.Pre(echomiddleware.RemoveTrailingSlash())
	m := middleware.NewMiddleware(config.Config{}, mockCacheRepo, mockDlqProcessorService)

	New(v1Group, mockTrxService, m)

	return orderTestHelper{
		router:         app,
		mockCtrl:       mockCtrl,
		mockTrxService: mockTrxService,
	}
}
