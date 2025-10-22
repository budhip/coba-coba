package order

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	nethttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandlerCreateOrder(t *testing.T) {
	testHelper := getOrderTestHelper(t)
	uuid, err := uuid.Parse("0afb1662-0763-4ec2-bc72-01dfe0f91ff8")
	assert.NoError(t, err)
	now, err := time.Parse(common.DateFormatYYYYMMDD, "2022-01-01")
	assert.NoError(t, err)
	statusActive := 0

	tests := []struct {
		name     string
		req      models.CreateOrderRequest
		wantRes  string
		wantCode int
		doMock   func(req models.CreateOrderRequest)
	}{
		{
			name: "happy path",
			req: models.CreateOrderRequest{
				OrderTime: &now,
				OrderType: "OrderType",
				RefNumber: "RefNumber",
				Transactions: []models.CreateOrderTransactionRequest{
					{
						ID:                   &uuid,
						Amount:               decimal.Zero,
						Currency:             "Currency",
						SourceAccountId:      "SourceAccountId",
						DestinationAccountId: "DestinationAccountId",
						Description:          "Description",
						Method:               "Method",
						TransactionType:      "TransactionType",
						TransactionTime:      &now,
						Status:               &statusActive,
						Meta:                 nil,
					},
				},
			},
			wantCode: nethttp.StatusCreated,
			wantRes:  `{"kind":"order","orderTime":"2022-01-01T00:00:00Z","orderType":"OrderType","refNumber":"RefNumber","transactions":[{"id":"0afb1662-0763-4ec2-bc72-01dfe0f91ff8","amount":"0","currency":"Currency","sourceAccountId":"SourceAccountId","destinationAccountId":"DestinationAccountId","description":"Description","method":"Method","transactionType":"TransactionType","transactionTime":"2022-01-01T00:00:00Z","status":0,"meta":null}]}`,
			doMock: func(req models.CreateOrderRequest) {
				testHelper.mockTrxService.EXPECT().
					NewStoreBulkTransaction(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf([]models.TransactionReq{})).
					Return(nil)
			},
		},
		{
			name: "failed - error service",
			req: models.CreateOrderRequest{
				OrderTime: &now,
				OrderType: "OrderType",
				RefNumber: "RefNumber",
				Transactions: []models.CreateOrderTransactionRequest{
					{
						ID:                   &uuid,
						Amount:               decimal.Zero,
						Currency:             "Currency",
						SourceAccountId:      "SourceAccountId",
						DestinationAccountId: "DestinationAccountId",
						Description:          "Description",
						Method:               "Method",
						TransactionType:      "TransactionType",
						TransactionTime:      &now,
						Status:               &statusActive,
						Meta:                 nil,
					},
				},
			},
			wantCode: nethttp.StatusInternalServerError,
			wantRes:  `{"status":"error","code":500,"message":"assert.AnError general error for testing"}`,
			doMock: func(req models.CreateOrderRequest) {
				testHelper.mockTrxService.EXPECT().
					NewStoreBulkTransaction(gomock.AssignableToTypeOf(context.Background()), gomock.AssignableToTypeOf([]models.TransactionReq{})).
					Return(assert.AnError)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.req)
			}

			var b bytes.Buffer
			errEncode := json.NewEncoder(&b).Encode(tt.req)
			require.NoError(t, errEncode)

			req := httptest.NewRequest("POST", "/api/v1/orders", &b)
			req.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			testHelper.router.ServeHTTP(rec, req)

			resp := rec.Result()
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, tt.wantRes, strings.TrimSuffix(string(body), "\n"))
			require.Equal(t, tt.wantCode, resp.StatusCode)
		})
	}
}
