package services_test

import (
	"context"
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestEntityService_Create(t *testing.T) {
	testHelper := serviceTestHelper(t)
	testHelper.mockSQLRepository.EXPECT().GetEntityRepository().Return(testHelper.mockEntityRepository).AnyTimes()

	type args struct {
		ctx context.Context
		req models.CreateEntityIn
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
				req: models.CreateEntityIn{
					Code:        "",
					Name:        "",
					Description: "",
				},
			},
			doMock: func(args args, mockData mockData) {
				testHelper.mockEntityRepository.EXPECT().GetByCode(args.ctx, args.req.Code).Return(nil, nil)
				testHelper.mockEntityRepository.EXPECT().Create(args.ctx, &args.req).Return(&models.Entity{}, nil)
			},
			wantErr: false,
		},
		{
			name: "code is exist",
			args: args{
				ctx: context.Background(),
				req: models.CreateEntityIn{},
			},
			mockData: mockData{},
			doMock: func(args args, mockData mockData) {
				testHelper.mockEntityRepository.EXPECT().GetByCode(args.ctx, args.req.Code).Return(&models.Entity{}, nil)
			},
			wantErr: true,
		},
		{
			name: "test error",
			args: args{
				ctx: context.Background(),
				req: models.CreateEntityIn{},
			},
			mockData: mockData{},
			doMock: func(args args, mockData mockData) {
				testHelper.mockEntityRepository.EXPECT().GetByCode(args.ctx, args.req.Code).Return(nil, nil)
				testHelper.mockEntityRepository.EXPECT().Create(args.ctx, &args.req).Return(nil, assert.AnError)
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

			_, err := testHelper.entityService.Create(tt.args.ctx, tt.args.req)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestEntityService_GetAll(t *testing.T) {
	testHelper := serviceTestHelper(t)
	testHelper.mockSQLRepository.EXPECT().GetEntityRepository().Return(testHelper.mockEntityRepository).AnyTimes()

	tests := []struct {
		name    string
		doMock  func()
		wantErr bool
	}{
		{
			name: "success get all entities",
			doMock: func() {
				testHelper.mockEntityRepository.EXPECT().
					List(gomock.AssignableToTypeOf(context.Background())).
					Return(&[]models.Entity{}, nil)
			},
			wantErr: false,
		},
		{
			name: "error get data from repository",
			doMock: func() {
				testHelper.mockEntityRepository.EXPECT().
					List(gomock.AssignableToTypeOf(context.Background())).
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

			_, err := testHelper.entityService.GetAll(context.Background())
			assert.Equal(t, tc.wantErr, err != nil)
		})
	}
}
