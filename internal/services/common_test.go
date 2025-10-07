package services

import (
	"testing"

	"bitbucket.org/Amartha/go-fp-transaction/internal/common"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

func Test_checkDatabaseError(t *testing.T) {
	type args struct {
		err  error
		code []string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "ErrNoRows",
			args: args{
				err:  common.ErrNoRows,
				code: []string{models.ErrKeySubCategoryCodeNotFound},
			},
			wantErr: true,
		},
		{
			name: "DatabaseError",
			args: args{
				err:  common.ErrDataTrxDuplicate,
				code: []string{models.ErrKeyDatabaseError},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := checkDatabaseError(tt.args.err, tt.args.code...); (err != nil) != tt.wantErr {
				t.Errorf("checkDatabaseError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
