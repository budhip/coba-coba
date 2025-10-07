package account

import (
	"os"
	"testing"

	xlog "bitbucket.org/Amartha/go-x/log"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/http/middleware"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	mockRepo "bitbucket.org/Amartha/go-fp-transaction/internal/repositories/mock"
	"bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/mock/gomock"
)

type testAccountHelper struct {
	router                   *fiber.App
	mockCtrl                 *gomock.Controller
	mockAccountService       *mock.MockAccountService
	mockBalanceService       *mock.MockBalanceService
	mockWalletAccountService *mock.MockWalletAccountService
}

func accountTestHelper(t *testing.T) testAccountHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockAccountSvc := mock.NewMockAccountService(mockCtrl)
	mockBalanceSvc := mock.NewMockBalanceService(mockCtrl)
	mockWalletAccountSvc := mock.NewMockWalletAccountService(mockCtrl)
	mockCacheRepo := mockRepo.NewMockCacheRepository(mockCtrl)
	mockDlqProcessorService := mock.NewMockDLQProcessorService(mockCtrl)

	app := fiber.New()
	v1Group := app.Group("/api/v1")
	m := middleware.NewMiddleware(config.Config{}, mockCacheRepo, mockDlqProcessorService)

	New(v1Group, mockAccountSvc, mockWalletAccountSvc, mockBalanceSvc, m)

	return testAccountHelper{
		router:                   app,
		mockCtrl:                 mockCtrl,
		mockAccountService:       mockAccountSvc,
		mockBalanceService:       mockBalanceSvc,
		mockWalletAccountService: mockWalletAccountSvc,
	}
}

func TestMain(m *testing.M) {
	xlog.InitForTest()
	os.Exit(m.Run())
}
