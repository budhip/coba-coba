package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_generateAccountNumber(t *testing.T) {
	type args struct {
		categoryCode string
		entityCode   string
		padWidth     int64
		lastSequence int64
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "success generate with pad 8 lastSequence 1",
			args: args{
				categoryCode: "222",
				entityCode:   "001",
				padWidth:     8,
				lastSequence: 1,
			},
			want: "22200100000001",
		},
		{
			name: "success generate with pad 8 lastSequence 100",
			args: args{
				categoryCode: "221",
				entityCode:   "002",
				padWidth:     8,
				lastSequence: 100,
			},
			want: "22100200000100",
		},
		{
			name: "success generate with pad 8 lastSequence 99999999",
			args: args{
				categoryCode: "211",
				entityCode:   "002",
				padWidth:     8,
				lastSequence: 99999999,
			},
			want: "21100299999999",
		},
		{
			name: "fail generate with pad 8 lastSequence 123456789 -lastSequence exceed padding width",
			args: args{
				categoryCode: "222",
				entityCode:   "002",
				padWidth:     8,
				lastSequence: 123456789,
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateAccountNumber(tt.args.categoryCode, tt.args.entityCode, tt.args.padWidth, tt.args.lastSequence)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want, got)
		})
	}
}
