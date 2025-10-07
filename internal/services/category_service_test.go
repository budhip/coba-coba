package services_test

import (
	"context"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCategoryService_Create(t *testing.T) {
	testHelper := serviceTestHelper(t)
	testHelper.mockSQLRepository.EXPECT().GetCategoryRepository().Return(testHelper.mockCategoryRepository).AnyTimes()

	type args struct {
		ctx context.Context
		req models.CreateCategoryIn
	}
	type mockData struct {
	}
	tests := []struct {
		name     string
		args     args
		mockData mockData
		doMock   func(args args, mockData mockData)
		wantErr  bool
	}{
		{
			name: "test success",
			args: args{
				ctx: context.Background(),
				req: models.CreateCategoryIn{
					Code:        "",
					Name:        "",
					Description: "",
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockCategoryRepository.EXPECT().GetByCode(args.ctx, args.req.Code).Return(nil, nil)
				testHelper.mockCategoryRepository.EXPECT().Create(args.ctx, &args.req).Return(&models.Category{}, nil)
			},
			wantErr: false,
		},
		{
			name: "code is exist",
			args: args{
				ctx: context.Background(),
				req: models.CreateCategoryIn{},
			},
			mockData: mockData{},
			doMock: func(args args, mockData mockData) {
				testHelper.mockCategoryRepository.EXPECT().GetByCode(args.ctx, args.req.Code).Return(&models.Category{}, nil)
			},
			wantErr: true,
		},
		{
			name: "test error",
			args: args{
				ctx: context.Background(),
				req: models.CreateCategoryIn{},
			},
			mockData: mockData{},
			doMock: func(args args, mockData mockData) {
				testHelper.mockCategoryRepository.EXPECT().GetByCode(args.ctx, args.req.Code).Return(nil, nil)
				testHelper.mockCategoryRepository.EXPECT().Create(args.ctx, &args.req).Return(nil, assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args, tt.mockData)
			}

			_, err := testHelper.categoryService.Create(tt.args.ctx, tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestCategoryService_GetAll(t *testing.T) {
	testHelper := serviceTestHelper(t)
	testHelper.mockSQLRepository.EXPECT().GetCategoryRepository().Return(testHelper.mockCategoryRepository).AnyTimes()

	tests := []struct {
		name    string
		doMock  func()
		wantErr bool
	}{
		{
			name: "happy path",
			doMock: func() {
				testHelper.mockCategoryRepository.EXPECT().List(gomock.AssignableToTypeOf(context.Background())).
					Return(&[]models.Category{}, nil)
			},
			wantErr: false,
		},
		{
			name: "error repository",
			doMock: func() {
				testHelper.mockCategoryRepository.EXPECT().List(gomock.AssignableToTypeOf(context.Background())).
					Return(nil, assert.AnError)
			},
			wantErr: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.doMock != nil {
				tc.doMock()
			}

			_, err := testHelper.categoryService.GetAll(context.Background())
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}
