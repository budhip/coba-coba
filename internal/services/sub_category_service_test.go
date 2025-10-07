package services_test

import (
	"context"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestSubCategoryService_Create(t *testing.T) {
	testHelper := serviceTestHelper(t)

	type args struct {
		ctx context.Context
		req models.CreateSubCategory
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
			name: "happy path",
			args: args{
				ctx: context.Background(),
				req: models.CreateSubCategory{},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockCategoryRepository.EXPECT().GetByCode(args.ctx, args.req.Code).Return(&models.Category{}, nil)
				testHelper.mockSubCategoryRepository.EXPECT().GetByCode(args.ctx, args.req.Code).Return(nil, nil)
				testHelper.mockSubCategoryRepository.EXPECT().Create(args.ctx, &args.req).Return(&models.SubCategory{}, nil)
			},
			wantErr: false,
		},
		{
			name: "invalid category",
			args: args{
				ctx: context.Background(),
				req: models.CreateSubCategory{},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockCategoryRepository.EXPECT().GetByCode(args.ctx, args.req.Code).Return(nil, nil)
			},
			wantErr: true,
		},
		{
			name: "code is exist",
			args: args{
				ctx: context.Background(),
				req: models.CreateSubCategory{},
			},
			mockData: mockData{},
			doMock: func(args args, mockData mockData) {
				testHelper.mockCategoryRepository.EXPECT().GetByCode(args.ctx, args.req.Code).Return(&models.Category{}, nil)
				testHelper.mockSubCategoryRepository.EXPECT().GetByCode(args.ctx, args.req.Code).Return(&models.SubCategory{}, nil)
			},
			wantErr: true,
		},
		{
			name: "error database",
			args: args{
				ctx: context.Background(),
				req: models.CreateSubCategory{},
			},
			mockData: mockData{},
			doMock: func(args args, mockData mockData) {
				testHelper.mockCategoryRepository.EXPECT().GetByCode(args.ctx, args.req.Code).Return(&models.Category{}, nil)
				testHelper.mockSubCategoryRepository.EXPECT().GetByCode(args.ctx, args.req.Code).Return(nil, nil)
				testHelper.mockSubCategoryRepository.EXPECT().Create(args.ctx, &args.req).Return(nil, assert.AnError)
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

			_, err := testHelper.subCategoryService.Create(tt.args.ctx, tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestSubCategoryService_GetAll(t *testing.T) {
	testHelper := serviceTestHelper(t)
	tests := []struct {
		name    string
		doMock  func()
		wantErr bool
	}{
		{
			name: "success - get all entities",
			doMock: func() {
				testHelper.mockSubCategoryRepository.EXPECT().
					GetAll(gomock.AssignableToTypeOf(context.Background())).
					Return(&[]models.SubCategory{}, nil)
			},
			wantErr: false,
		},
		{
			name: "error - get data from repository",
			doMock: func() {
				testHelper.mockSubCategoryRepository.EXPECT().
					GetAll(gomock.AssignableToTypeOf(context.Background())).
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
			_, err := testHelper.subCategoryService.GetAll(context.Background())
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}
