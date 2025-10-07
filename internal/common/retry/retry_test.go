package retry_test

import (
	"context"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common/retry"
	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
	svcMock "bitbucket.org/Amartha/go-fp-transaction/internal/services/mock"

	xlog "bitbucket.org/Amartha/go-x/log"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func init() {
	xlog.InitForTest()
}

type retryTestHelper struct {
	mockCtrl   *gomock.Controller
	trxSvcMock *svcMock.MockTransactionService
	retryerSUT retry.Retryer
}

func newRetryTestHelper(t *testing.T, ebCfg *config.ExponentialBackOffConfig) retryTestHelper {
	t.Helper()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	trxSvcMock := svcMock.NewMockTransactionService(mockCtrl)

	return retryTestHelper{
		mockCtrl:   mockCtrl,
		trxSvcMock: trxSvcMock,
		retryerSUT: retry.NewExponentialBackOff(ebCfg),
	}
}

func Test_Retry_ExponentialBackoff(t *testing.T) {
	t.Run("failed - DLQ Operation called and return err", func(t *testing.T) {
		var dlqCallbackCalled int
		testHelper := newRetryTestHelper(t, &config.ExponentialBackOffConfig{MaxRetries: 1})

		testHelper.trxSvcMock.EXPECT().
			StoreBulkTransaction(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
			Return(assert.AnError).AnyTimes()

		err := testHelper.retryerSUT.Retry(
			context.Background(),
			func() error {
				return testHelper.trxSvcMock.StoreBulkTransaction(context.Background(), []models.TransactionReq{})
			},
			func() error {
				dlqCallbackCalled = dlqCallbackCalled + 1
				return assert.AnError
			},
		)
		assert.NotNil(t, err)
		assert.Equal(t, dlqCallbackCalled, 1)
	})

	t.Run("failed - DLQ Operation called", func(t *testing.T) {
		var dlqCallbackCalled int
		testHelper := newRetryTestHelper(t, &config.ExponentialBackOffConfig{MaxRetries: 1})

		testHelper.trxSvcMock.EXPECT().
			StoreBulkTransaction(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
			Return(assert.AnError).AnyTimes()

		err := testHelper.retryerSUT.Retry(
			context.Background(),
			func() error {
				return testHelper.trxSvcMock.StoreBulkTransaction(context.Background(), []models.TransactionReq{})
			},
			func() error {
				dlqCallbackCalled = dlqCallbackCalled + 1
				return nil
			},
		)
		assert.Nil(t, err)
		assert.Equal(t, dlqCallbackCalled, 1)
	})

	t.Run("success - DLQ Operation not called", func(t *testing.T) {
		var dlqCallbackCalled int
		testHelper := newRetryTestHelper(t, &config.ExponentialBackOffConfig{})

		testHelper.trxSvcMock.EXPECT().
			StoreBulkTransaction(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
			Return(nil).
			AnyTimes()

		err := testHelper.retryerSUT.Retry(
			context.Background(),
			func() error {
				return testHelper.trxSvcMock.StoreBulkTransaction(context.Background(), []models.TransactionReq{})
			},
			func() error {
				dlqCallbackCalled = dlqCallbackCalled + 1
				return nil
			},
		)
		assert.Nil(t, err)
		assert.Equal(t, dlqCallbackCalled, 0)
	})

	t.Run("success - force stop retrying", func(t *testing.T) {
		var dlqCallbackCalled int
		var processCount int
		testHelper := newRetryTestHelper(t, &config.ExponentialBackOffConfig{MaxRetries: 5})

		testHelper.trxSvcMock.EXPECT().
			StoreBulkTransaction(gomock.AssignableToTypeOf(context.Background()), gomock.Any()).
			Return(assert.AnError).AnyTimes()

		err := testHelper.retryerSUT.Retry(
			context.Background(),
			func() error {
				processCount = processCount + 1

				err := testHelper.trxSvcMock.StoreBulkTransaction(context.Background(), []models.TransactionReq{})

				// force stop retrying
				return testHelper.retryerSUT.StopRetryWithErr(err)
			},
			func() error {
				dlqCallbackCalled = dlqCallbackCalled + 1
				return nil
			},
		)

		assert.Nil(t, err)
		assert.Equal(t, processCount, 1)
		assert.Equal(t, dlqCallbackCalled, 1)
	})
}
