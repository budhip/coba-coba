package services_test

import (
	"context"
	"os"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func Test_ReconService_ProcessReconTaskQueue(t *testing.T) {
	reconSUT := initReconSUT(t)
	reconSUT.mockSQLRepo.EXPECT().GetReconToolHistoryRepository().Return(reconSUT.mockReconToolHistoryRepo).AnyTimes()
	reconSUT.mockSQLRepo.EXPECT().GetTransactionRepository().Return(reconSUT.mockTransactionRepository).AnyTimes()

	defaultTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	timeNow := time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)

	type args struct {
		ctx context.Context
		id  uint64
	}

	type mockData struct {
		resultFile *os.File
		inputFile  *os.File
	}

	tests := []struct {
		name       string
		args       args
		beforeEach func(args args, md *mockData)
		afterEach  func(args args, md *mockData)
		mockData   mockData
		wantResult []byte
		wantErr    bool
	}{
		{
			name: "success recon task queue",
			args: args{ctx: context.Background()},
			beforeEach: func(args args, md *mockData) {
				rh := &models.ReconToolHistory{
					ID:               int(args.id),
					OrderType:        "TOPUP",
					TransactionType:  "TOPUP",
					TransactionDate:  &defaultTime,
					ResultFilePath:   "my_file.txt",
					UploadedFilePath: "my_file.txt",
					Status:           "SUCCESS",
					CreatedAt:        &timeNow,
					UpdatedAt:        &timeNow,
				}

				reconSUT.mockReconToolHistoryRepo.EXPECT().GetById(args.ctx, args.id).Return(rh, nil)

				rh.Status = models.ReconHistoryStatusProcessing
				reconSUT.mockReconToolHistoryRepo.EXPECT().Update(args.ctx, args.id, rh)

				chanTrx := make(chan models.TransactionStreamResult)
				reconSUT.mockTransactionRepository.EXPECT().StreamAll(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, opts models.TransactionFilterOptions) <-chan models.TransactionStreamResult {
					go func() {
						defer close(chanTrx)
						chanTrx <- models.TransactionStreamResult{
							Data: models.Transaction{
								RefNumber:       "123456",
								TransactionDate: defaultTime,
								Amount:          decimal.NewNullDecimal(decimal.NewFromInt(100000)),
								Metadata:        "",
							},
						}
						chanTrx <- models.TransactionStreamResult{
							Data: models.Transaction{
								RefNumber:       "123456_only_in_db",
								TransactionDate: defaultTime,
								Amount:          decimal.NewNullDecimal(decimal.NewFromInt(100001)),
								Metadata:        "",
							},
						}
						chanTrx <- models.TransactionStreamResult{
							Data: models.Transaction{
								RefNumber:       "ref_number_va_permata",
								TransactionDate: defaultTime,
								Amount:          decimal.NewNullDecimal(decimal.NewFromInt(100021)),
								Metadata:        `{"vaData":{"source":"PERMATA","virtualAccountId":"","virtualAccountNo":"86861101189513"},"eWalletData":null,"toNarrative":"TOPUP CHANEL","fromNarrative":"LENDER","institutionData":null}`,
							},
						}

						// same va number but different amount. status should be not match
						chanTrx <- models.TransactionStreamResult{
							Data: models.Transaction{
								RefNumber:       "ref_number_va_permata",
								TransactionDate: defaultTime,
								Amount:          decimal.NewNullDecimal(decimal.NewFromInt(100023)),
								Metadata:        `{"vaData":{"source":"PERMATA","virtualAccountId":"","virtualAccountNo":"86861101189513"},"eWalletData":null,"toNarrative":"TOPUP CHANEL","fromNarrative":"LENDER","institutionData":null}`,
							},
						}
					}()

					return chanTrx
				})

				md.inputFile, _ = os.CreateTemp("", "test_file_recon_csv_input")
				md.inputFile.Write([]byte("123456,100000,01-Jan-2023,this is remark\n"))
				md.inputFile.Write([]byte("123456_only_in_csv,100000,01-Jan-2023,this is remark\n"))
				md.inputFile.Write([]byte("86861101189513,100021,01-Jan-2023,this is remark\n"))

				// when processing recon this should be skipped (different transaction date)
				md.inputFile.Write([]byte("86861101189513,100021,05-Jan-2023,this is remark\n"))

				md.inputFile.Close()
				md.inputFile, _ = os.Open(md.inputFile.Name())

				reconSUT.mockStorageRepo.EXPECT().NewReader(gomock.Any(), gomock.Any()).Return(md.inputFile, nil)

				md.resultFile, _ = os.CreateTemp("", "test_file_recon_csv_result")
				reconSUT.mockStorageRepo.EXPECT().NewWriter(gomock.Any(), gomock.Any()).Return(md.resultFile)

				rh.Status = models.ReconHistoryStatusSuccess
				reconSUT.mockReconToolHistoryRepo.EXPECT().Update(args.ctx, args.id, rh)
			},
			afterEach: func(args args, md *mockData) {
				os.Remove(md.inputFile.Name())
				os.Remove(md.resultFile.Name())
			},
			wantErr: false,
			wantResult: []byte("identifier,amount,orderType,transactionType,transactionDate,refNumber,lenderId,customerName,reconDate,match,status\n" +
				"123456,100000,TOPUP,TOPUP,01-Jan-2023,123456,,,2023-01-10 07:00:00,true,Match\n" +
				"123456_only_in_csv,100000,TOPUP,TOPUP,01-Jan-2023,,,,2023-01-10 07:00:00,false,\"Not Exists in DB, Exists in CSV\"\n" +
				"86861101189513,100021,TOPUP,TOPUP,01-Jan-2023,ref_number_va_permata,,,2023-01-10 07:00:00,true,Match\n" +
				"123456_only_in_db,100001,TOPUP,TOPUP,01-Jan-2023,123456_only_in_db,,,2023-01-10 07:00:00,false,\"Exists in DB, Not Exists in CSV\"\n" +
				"86861101189513,100023,TOPUP,TOPUP,01-Jan-2023,ref_number_va_permata,,,2023-01-10 07:00:00,false,\"Exists in DB, Not Exists in CSV\"\n"),
		},
		{
			name: "success recon task queue",
			args: args{ctx: context.Background()},
			beforeEach: func(args args, md *mockData) {
				rh := &models.ReconToolHistory{
					ID:               int(args.id),
					OrderType:        "TOPUP",
					TransactionType:  "TOPUP",
					TransactionDate:  &defaultTime,
					ResultFilePath:   "my_file.txt",
					UploadedFilePath: "my_file.txt",
					Status:           "SUCCESS",
					CreatedAt:        &timeNow,
					UpdatedAt:        &timeNow,
				}

				// No mock expectations needed - using structured error logging
				reconSUT.mockReconToolHistoryRepo.EXPECT().GetById(args.ctx, args.id).Return(rh, nil)

				rh.Status = models.ReconHistoryStatusProcessing
				reconSUT.mockReconToolHistoryRepo.EXPECT().Update(args.ctx, args.id, rh)

				chanTrx := make(chan models.TransactionStreamResult)
				reconSUT.mockTransactionRepository.EXPECT().StreamAll(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, opts models.TransactionFilterOptions) <-chan models.TransactionStreamResult {
					go func() {
						defer close(chanTrx)
						chanTrx <- models.TransactionStreamResult{
							Data: models.Transaction{
								RefNumber:       "123456",
								TransactionDate: defaultTime,
								Amount:          decimal.NewNullDecimal(decimal.NewFromInt(100000)),
								Metadata:        "",
							},
						}
						chanTrx <- models.TransactionStreamResult{
							Data: models.Transaction{
								RefNumber:       "123456_only_in_db",
								TransactionDate: defaultTime,
								Amount:          decimal.NewNullDecimal(decimal.NewFromInt(100001)),
								Metadata:        "",
							},
						}
						chanTrx <- models.TransactionStreamResult{
							Data: models.Transaction{
								RefNumber:       "ref_number_va_permata",
								TransactionDate: defaultTime,
								Amount:          decimal.NewNullDecimal(decimal.NewFromInt(100021)),
								Metadata:        `{"vaData":{"source":"PERMATA","virtualAccountId":"","virtualAccountNo":"86861101189513"},"eWalletData":null,"toNarrative":"TOPUP CHANEL","fromNarrative":"LENDER","institutionData":null}`,
							},
						}

						// exists in db and csv, but csv already match with other transaction.
						// so this should be not match
						chanTrx <- models.TransactionStreamResult{
							Data: models.Transaction{
								RefNumber:       "ref_number_va_permata_2",
								TransactionDate: defaultTime,
								Amount:          decimal.NewNullDecimal(decimal.NewFromInt(100021)),
								Metadata:        `{"vaData":{"source":"PERMATA","virtualAccountId":"","virtualAccountNo":"86861101189513"},"eWalletData":null,"toNarrative":"TOPUP CHANEL","fromNarrative":"LENDER","institutionData":null}`,
							},
						}

						// same va number but different amount. status should be not match
						chanTrx <- models.TransactionStreamResult{
							Data: models.Transaction{
								RefNumber:       "ref_number_va_permata",
								TransactionDate: defaultTime,
								Amount:          decimal.NewNullDecimal(decimal.NewFromInt(100023)),
								Metadata:        `{"vaData":{"source":"PERMATA","virtualAccountId":"","virtualAccountNo":"86861101189513"},"eWalletData":null,"toNarrative":"TOPUP CHANEL","fromNarrative":"LENDER","institutionData":null}`,
							},
						}
					}()

					return chanTrx
				})

				md.inputFile, _ = os.CreateTemp("", "test_file_recon_csv_input")
				md.inputFile.Write([]byte("123456,100000,01-Jan-2023,this is remark\n"))
				md.inputFile.Write([]byte("123456_only_in_csv,100000,01-Jan-2023,this is remark\n"))
				md.inputFile.Write([]byte("86861101189513,100021,01-Jan-2023,this is remark\n"))

				// when processing recon this should be skipped (different transaction date)
				md.inputFile.Write([]byte("86861101189513,100021,05-Jan-2023,this is remark\n"))

				md.inputFile.Close()
				md.inputFile, _ = os.Open(md.inputFile.Name())

				reconSUT.mockStorageRepo.EXPECT().NewReader(gomock.Any(), gomock.Any()).Return(md.inputFile, nil)

				md.resultFile, _ = os.CreateTemp("", "test_file_recon_csv_result")
				reconSUT.mockStorageRepo.EXPECT().NewWriter(gomock.Any(), gomock.Any()).Return(md.resultFile)

				rh.Status = models.ReconHistoryStatusSuccess
				reconSUT.mockReconToolHistoryRepo.EXPECT().Update(args.ctx, args.id, rh)
			},
			afterEach: func(args args, md *mockData) {
				os.Remove(md.inputFile.Name())
				os.Remove(md.resultFile.Name())
			},
			wantErr: false,
			wantResult: []byte("identifier,amount,orderType,transactionType,transactionDate,refNumber,lenderId,customerName,reconDate,match,status\n" +
				"123456,100000,TOPUP,TOPUP,01-Jan-2023,123456,,,2023-01-10 07:00:00,true,Match\n" +
				"123456_only_in_csv,100000,TOPUP,TOPUP,01-Jan-2023,,,,2023-01-10 07:00:00,false,\"Not Exists in DB, Exists in CSV\"\n" +
				"86861101189513,100021,TOPUP,TOPUP,01-Jan-2023,ref_number_va_permata,,,2023-01-10 07:00:00,true,Match\n" +
				"123456_only_in_db,100001,TOPUP,TOPUP,01-Jan-2023,123456_only_in_db,,,2023-01-10 07:00:00,false,\"Exists in DB, Not Exists in CSV\"\n" +
				"86861101189513,100021,TOPUP,TOPUP,01-Jan-2023,ref_number_va_permata_2,,,2023-01-10 07:00:00,false,\"Exists in DB, Not Exists in CSV\"\n" +
				"86861101189513,100023,TOPUP,TOPUP,01-Jan-2023,ref_number_va_permata,,,2023-01-10 07:00:00,false,\"Exists in DB, Not Exists in CSV\"\n"),
		},
		{
			name: "failed to get recon history",
			args: args{ctx: context.Background()},
			beforeEach: func(args args, md *mockData) {
				// No mock expectations needed - using structured error logging
				reconSUT.mockReconToolHistoryRepo.EXPECT().GetById(args.ctx, args.id).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed to update recon history status",
			args: args{ctx: context.Background()},
			beforeEach: func(args args, md *mockData) {
				rh := &models.ReconToolHistory{
					ID:               int(args.id),
					OrderType:        "TOPUP",
					TransactionType:  "TOPUP",
					TransactionDate:  &defaultTime,
					ResultFilePath:   "my_file.txt",
					UploadedFilePath: "my_file.txt",
					Status:           "SUCCESS",
					CreatedAt:        &timeNow,
					UpdatedAt:        &timeNow,
				}
				// No mock expectations needed - using structured error logging
				reconSUT.mockReconToolHistoryRepo.EXPECT().GetById(args.ctx, args.id).Return(rh, nil)

				rh.Status = models.ReconHistoryStatusProcessing
				reconSUT.mockReconToolHistoryRepo.EXPECT().Update(args.ctx, args.id, rh).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
		{
			name: "failed to load stream transaction",
			args: args{ctx: context.Background()},
			beforeEach: func(args args, md *mockData) {
				rh := &models.ReconToolHistory{
					ID:               int(args.id),
					OrderType:        "TOPUP",
					TransactionType:  "TOPUP",
					TransactionDate:  &defaultTime,
					ResultFilePath:   "my_file.txt",
					UploadedFilePath: "my_file.txt",
					Status:           "SUCCESS",
					CreatedAt:        &timeNow,
					UpdatedAt:        &timeNow,
				}

				// No mock expectations needed - using structured error logging
				reconSUT.mockReconToolHistoryRepo.EXPECT().GetById(args.ctx, args.id).Return(rh, nil)

				rh.Status = models.ReconHistoryStatusProcessing
				reconSUT.mockReconToolHistoryRepo.EXPECT().Update(args.ctx, args.id, rh)

				chanTrx := make(chan models.TransactionStreamResult)
				reconSUT.mockTransactionRepository.EXPECT().StreamAll(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, opts models.TransactionFilterOptions) <-chan models.TransactionStreamResult {
					go func() {
						defer close(chanTrx)
						chanTrx <- models.TransactionStreamResult{
							Err: assert.AnError,
						}
					}()

					return chanTrx
				})
			},
			wantErr: true,
		},
		{
			name: "failed to convert transaction to recon record",
			args: args{ctx: context.Background()},
			beforeEach: func(args args, md *mockData) {
				rh := &models.ReconToolHistory{
					ID:               int(args.id),
					OrderType:        "TOPUP",
					TransactionType:  "TOPUP",
					TransactionDate:  &defaultTime,
					ResultFilePath:   "my_file.txt",
					UploadedFilePath: "my_file.txt",
					Status:           "SUCCESS",
					CreatedAt:        &timeNow,
					UpdatedAt:        &timeNow,
				}

				// No mock expectations needed - using structured error logging
				reconSUT.mockReconToolHistoryRepo.EXPECT().GetById(args.ctx, args.id).Return(rh, nil)

				rh.Status = models.ReconHistoryStatusProcessing
				reconSUT.mockReconToolHistoryRepo.EXPECT().Update(args.ctx, args.id, rh)

				chanTrx := make(chan models.TransactionStreamResult)
				reconSUT.mockTransactionRepository.EXPECT().StreamAll(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, opts models.TransactionFilterOptions) <-chan models.TransactionStreamResult {
					go func() {
						defer close(chanTrx)
						chanTrx <- models.TransactionStreamResult{
							Data: models.Transaction{
								RefNumber:       "123456",
								TransactionDate: defaultTime,
								Amount:          decimal.NewNullDecimal(decimal.NewFromInt(100000)),
								Metadata:        "this is not valid json {[{!!!",
							},
						}
					}()
					return chanTrx
				})
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.beforeEach != nil {
				tt.beforeEach(tt.args, &tt.mockData)
			}

			err := reconSUT.sut.ProcessReconTaskQueue(tt.args.ctx, tt.args.id)
			assert.Equal(t, tt.wantErr, err != nil)

			if tt.mockData.resultFile != nil {
				resultFile, err := os.ReadFile(tt.mockData.resultFile.Name())
				if err != nil {
					t.Fatal(err)
				}
				assert.Equal(t, string(tt.wantResult), string(resultFile))
			}

			if tt.afterEach != nil {
				tt.afterEach(tt.args, &tt.mockData)
			}
		})
	}
}
