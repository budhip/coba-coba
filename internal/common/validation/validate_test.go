package validation

import (
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestValidateStruct(t *testing.T) {
	type args struct {
		toValidate interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success DoCreateAccountRequest",
			args: args{
				toValidate: models.DoCreateAccountRequest{
					AccountNumber:   "21100100000001",
					Name:            "John",
					OwnerID:         "12345",
					CategoryCode:    "211",
					SubCategoryCode: "10000",
					EntityCode:      "001",
					Currency:        "IDR",
					AltId:           "12345",
					Status:          "active",
				},
			},
			wantErr: false,
		},
		{
			name: "validate DoCreateAccountRequest",
			args: args{
				toValidate: models.DoCreateAccountRequest{
					CategoryCode:    "211",
					SubCategoryCode: "10000",
					EntityCode:      "001",
					Currency:        "IDR",
				},
			},
			wantErr: true,
		},
		{
			name: "validate CreateSubCategoryRequest",
			args: args{
				toValidate: models.CreateSubCategoryRequest{
					Code:        "100",
					Name:        "001",
					Description: "IDR",
				},
			},
			wantErr: true,
		},
		{
			name: "validate error not register",
			args: args{
				toValidate: struct {
					Name string `json:"name" validate:"required,date"`
				}{
					Name: "12345678901234",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateStruct(tt.args.toValidate)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
