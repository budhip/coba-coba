package models

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/shopspring/decimal"
)

func balanceComparer() cmp.Option {
	return cmp.Comparer(func(x, y Balance) bool {
		return x.Actual().Equal(y.Actual()) &&
			x.Pending().Equal(y.Pending()) &&
			x.Available().Equal(y.Available())
	})
}

func TestBalance_Actual(t *testing.T) {
	type fields struct {
		actualBalance    decimal.Decimal
		pendingBalance   decimal.Decimal
		ignoreValidation bool
	}
	tests := []struct {
		name   string
		fields fields
		want   decimal.Decimal
	}{
		{
			name: "valid available balance",
			fields: fields{
				actualBalance:  decimal.NewFromInt(100),
				pendingBalance: decimal.Zero,
			},
			want: decimal.NewFromInt(100),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Balance{
				actualBalance:            tt.fields.actualBalance,
				pendingBalance:           tt.fields.pendingBalance,
				ignoreBalanceSufficiency: tt.fields.ignoreValidation,
			}
			got := b.Actual()
			if !cmp.Equal(tt.want, got) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(tt.want, got))
			}
		})
	}
}

func TestBalance_Pending(t *testing.T) {
	type fields struct {
		actualBalance    decimal.Decimal
		pendingBalance   decimal.Decimal
		ignoreValidation bool
	}
	tests := []struct {
		name   string
		fields fields
		want   decimal.Decimal
	}{
		{
			name: "valid available balance",
			fields: fields{
				actualBalance:  decimal.NewFromInt(100),
				pendingBalance: decimal.NewFromFloat(420.69),
			},
			want: decimal.NewFromFloat(420.69),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Balance{
				actualBalance:            tt.fields.actualBalance,
				pendingBalance:           tt.fields.pendingBalance,
				ignoreBalanceSufficiency: tt.fields.ignoreValidation,
			}
			got := b.Pending()
			if !cmp.Equal(tt.want, got) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(tt.want, got))
			}
		})
	}
}

func TestBalance_Available(t *testing.T) {
	type fields struct {
		actualBalance    decimal.Decimal
		pendingBalance   decimal.Decimal
		ignoreValidation bool
	}
	tests := []struct {
		name   string
		fields fields
		want   decimal.Decimal
	}{
		{
			name: "valid available balance",
			fields: fields{
				actualBalance:  decimal.NewFromFloat(420.69),
				pendingBalance: decimal.NewFromFloat(100),
			},
			want: decimal.NewFromFloat(320.69),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Balance{
				actualBalance:            tt.fields.actualBalance,
				pendingBalance:           tt.fields.pendingBalance,
				ignoreBalanceSufficiency: tt.fields.ignoreValidation,
			}
			got := b.Available()
			if !cmp.Equal(tt.want, got) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(tt.want, got))
			}
		})
	}
}

func TestBalance_AddFunds(t *testing.T) {
	type fields struct {
		actualBalance    decimal.Decimal
		pendingBalance   decimal.Decimal
		ignoreValidation bool
	}
	type args struct {
		amount decimal.Decimal
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Balance
		wantErr bool
	}{
		{
			name: "success adding funds",
			fields: fields{
				actualBalance:    decimal.NewFromFloat(420.69),
				pendingBalance:   decimal.NewFromFloat(100),
				ignoreValidation: false,
			},
			args: args{
				amount: decimal.NewFromFloat(100),
			},
			want:    NewBalance(decimal.NewFromFloat(520.69), decimal.NewFromFloat(100)),
			wantErr: false,
		},
		{
			name: "error adding funds - invalid amount",
			fields: fields{
				actualBalance:    decimal.NewFromFloat(420.69),
				pendingBalance:   decimal.NewFromFloat(100),
				ignoreValidation: false,
			},
			args: args{
				amount: decimal.NewFromFloat(-100),
			},
			want:    NewBalance(decimal.NewFromFloat(420.69), decimal.NewFromFloat(100)),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Balance{
				actualBalance:            tt.fields.actualBalance,
				pendingBalance:           tt.fields.pendingBalance,
				ignoreBalanceSufficiency: tt.fields.ignoreValidation,
			}
			if err := b.AddFunds(tt.args.amount); (err != nil) != tt.wantErr {
				t.Errorf("AddFunds() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !cmp.Equal(&tt.want, b, balanceComparer()) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(&tt.want, b, balanceComparer()))
			}
		})
	}
}

func TestBalance_Withdraw(t *testing.T) {
	type fields struct {
		actualBalance                          decimal.Decimal
		pendingBalance                         decimal.Decimal
		ignoreBalanceSufficiency               bool
		negativeBalanceLimit                   decimal.NullDecimal
		allowedNegativeBalanceTransactionTypes []string
	}
	type args struct {
		amount decimal.Decimal
		opt    []CalculateBalanceOption
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Balance
		wantErr bool
	}{
		{
			name: "success withdraw funds",
			fields: fields{
				actualBalance:            decimal.NewFromFloat(420.69),
				pendingBalance:           decimal.NewFromFloat(100),
				ignoreBalanceSufficiency: false,
			},
			args: args{
				amount: decimal.NewFromFloat(100),
			},
			want:    NewBalance(decimal.NewFromFloat(320.69), decimal.NewFromFloat(100)),
			wantErr: false,
		},
		{
			name: "success withdraw with insufficient balance by ignoring validation",
			fields: fields{
				actualBalance:            decimal.NewFromFloat(420.69),
				pendingBalance:           decimal.NewFromFloat(100),
				ignoreBalanceSufficiency: true,
			},
			args: args{
				amount: decimal.NewFromFloat(500),
			},
			want:    NewBalance(decimal.NewFromFloat(-79.31), decimal.NewFromFloat(100), WithIgnoreBalanceSufficiency()),
			wantErr: false,
		},
		{
			name: "success reserve with negative balance limit",
			fields: fields{
				actualBalance:                          decimal.NewFromFloat(5000),
				pendingBalance:                         decimal.Zero,
				ignoreBalanceSufficiency:               false,
				allowedNegativeBalanceTransactionTypes: []string{"TUPVA"},
				negativeBalanceLimit:                   decimal.NewNullDecimal(decimal.NewFromFloat(5000)),
			},
			args: args{
				amount: decimal.NewFromFloat(10000),
				opt: []CalculateBalanceOption{
					WithTransactionType("TUPVA"),
				},
			},
			want: NewBalance(
				decimal.NewFromFloat(-5000),
				decimal.Zero,
				WithNegativeBalanceLimit(decimal.NewFromFloat(5000)),
				WithAllowedNegativeBalanceTransactionTypes([]string{"TUPVA"}),
			),
			wantErr: false,
		},
		{
			name: "error - negative balance reached",
			fields: fields{
				actualBalance:                          decimal.NewFromFloat(5000),
				pendingBalance:                         decimal.Zero,
				ignoreBalanceSufficiency:               false,
				allowedNegativeBalanceTransactionTypes: []string{"TUPVA"},
				negativeBalanceLimit:                   decimal.NewNullDecimal(decimal.NewFromFloat(5000)),
			},
			args: args{
				amount: decimal.NewFromFloat(10001),
				opt: []CalculateBalanceOption{
					WithTransactionType("TUPVA"),
				},
			},
			want: NewBalance(
				decimal.NewFromFloat(5000),
				decimal.Zero,
				WithNegativeBalanceLimit(decimal.NewFromFloat(5000)),
				WithAllowedNegativeBalanceTransactionTypes([]string{"TUPVA"}),
			),
			wantErr: true,
		},
		{
			name: "error withdraw funds - invalid amount",
			fields: fields{
				actualBalance:            decimal.NewFromFloat(420.69),
				pendingBalance:           decimal.NewFromFloat(100),
				ignoreBalanceSufficiency: false,
			},
			args: args{
				amount: decimal.NewFromFloat(-100),
			},
			want:    NewBalance(decimal.NewFromFloat(420.69), decimal.NewFromFloat(100)),
			wantErr: true,
		},
		{
			name: "error withdraw funds - insufficient available balance",
			fields: fields{
				actualBalance:            decimal.NewFromFloat(1),
				pendingBalance:           decimal.NewFromFloat(100),
				ignoreBalanceSufficiency: false,
			},
			args: args{
				amount: decimal.NewFromFloat(100),
			},
			want:    NewBalance(decimal.NewFromFloat(1), decimal.NewFromFloat(100)),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Balance{
				actualBalance:                          tt.fields.actualBalance,
				pendingBalance:                         tt.fields.pendingBalance,
				ignoreBalanceSufficiency:               tt.fields.ignoreBalanceSufficiency,
				negativeBalanceLimit:                   tt.fields.negativeBalanceLimit,
				allowedNegativeBalanceTransactionTypes: tt.fields.allowedNegativeBalanceTransactionTypes,
			}

			if err := b.Withdraw(tt.args.amount, tt.args.opt...); (err != nil) != tt.wantErr {
				t.Errorf("AddFunds() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !cmp.Equal(&tt.want, b, balanceComparer()) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(&tt.want, b, balanceComparer()))
			}
		})
	}
}

func TestBalance_Reserve(t *testing.T) {
	type fields struct {
		actualBalance                          decimal.Decimal
		pendingBalance                         decimal.Decimal
		ignoreBalanceSufficiency               bool
		negativeBalanceLimit                   decimal.NullDecimal
		allowedNegativeBalanceTransactionTypes []string
	}
	type args struct {
		amount decimal.Decimal
		opt    []CalculateBalanceOption
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Balance
		wantErr bool
	}{
		{
			name: "success reserve funds",
			fields: fields{
				actualBalance:            decimal.NewFromFloat(420.69),
				pendingBalance:           decimal.NewFromFloat(100),
				ignoreBalanceSufficiency: false,
			},
			args: args{
				amount: decimal.NewFromFloat(100),
			},
			want:    NewBalance(decimal.NewFromFloat(420.69), decimal.NewFromFloat(200)),
			wantErr: false,
		},
		{
			name: "success reserve with insufficient balance by ignoring validation",
			fields: fields{
				actualBalance:            decimal.NewFromFloat(300),
				pendingBalance:           decimal.NewFromFloat(100),
				ignoreBalanceSufficiency: true,
			},
			args: args{
				amount: decimal.NewFromFloat(500),
			},
			want:    NewBalance(decimal.NewFromFloat(300), decimal.NewFromFloat(600), WithIgnoreBalanceSufficiency()),
			wantErr: false,
		},
		{
			name: "success reserve with negative balance limit",
			fields: fields{
				actualBalance:                          decimal.NewFromFloat(5000),
				pendingBalance:                         decimal.Zero,
				ignoreBalanceSufficiency:               false,
				allowedNegativeBalanceTransactionTypes: []string{"TUPVA"},
				negativeBalanceLimit:                   decimal.NewNullDecimal(decimal.NewFromFloat(5000)),
			},
			args: args{
				amount: decimal.NewFromFloat(10000),
				opt: []CalculateBalanceOption{
					WithTransactionType("TUPVA"),
				},
			},
			want: NewBalance(
				decimal.NewFromFloat(5000),
				decimal.NewFromFloat(10000),
				WithNegativeBalanceLimit(decimal.NewFromFloat(5000)),
				WithAllowedNegativeBalanceTransactionTypes([]string{"TUPVA"}),
			),
			wantErr: false,
		},
		{
			name: "error - negative balance reached",
			fields: fields{
				actualBalance:                          decimal.NewFromFloat(5000),
				pendingBalance:                         decimal.Zero,
				ignoreBalanceSufficiency:               false,
				allowedNegativeBalanceTransactionTypes: []string{"TUPVA"},
				negativeBalanceLimit:                   decimal.NewNullDecimal(decimal.NewFromFloat(5000)),
			},
			args: args{
				amount: decimal.NewFromFloat(10001),
				opt: []CalculateBalanceOption{
					WithTransactionType("TUPVA"),
				},
			},
			want: NewBalance(
				decimal.NewFromFloat(5000),
				decimal.Zero,
				WithNegativeBalanceLimit(decimal.NewFromFloat(5000)),
				WithAllowedNegativeBalanceTransactionTypes([]string{"TUPVA"}),
			),
			wantErr: true,
		},
		{
			name: "error reserve funds - invalid amount",
			fields: fields{
				actualBalance:            decimal.NewFromFloat(420.69),
				pendingBalance:           decimal.NewFromFloat(100),
				ignoreBalanceSufficiency: false,
			},
			args: args{
				amount: decimal.NewFromFloat(-100),
			},
			want:    NewBalance(decimal.NewFromFloat(420.69), decimal.NewFromFloat(100)),
			wantErr: true,
		},
		{
			name: "error reserve funds - insufficient available balance",
			fields: fields{
				actualBalance:            decimal.NewFromFloat(1),
				pendingBalance:           decimal.NewFromFloat(100),
				ignoreBalanceSufficiency: false,
			},
			args: args{
				amount: decimal.NewFromFloat(100),
			},
			want:    NewBalance(decimal.NewFromFloat(1), decimal.NewFromFloat(100)),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Balance{
				actualBalance:                          tt.fields.actualBalance,
				pendingBalance:                         tt.fields.pendingBalance,
				ignoreBalanceSufficiency:               tt.fields.ignoreBalanceSufficiency,
				negativeBalanceLimit:                   tt.fields.negativeBalanceLimit,
				allowedNegativeBalanceTransactionTypes: tt.fields.allowedNegativeBalanceTransactionTypes,
			}

			if err := b.Reserve(tt.args.amount, tt.args.opt...); (err != nil) != tt.wantErr {
				t.Errorf("AddFunds() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !cmp.Equal(&tt.want, b, balanceComparer()) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(&tt.want, b, balanceComparer()))
			}
		})
	}
}

func TestBalance_CancelReservation(t *testing.T) {
	type fields struct {
		actualBalance    decimal.Decimal
		pendingBalance   decimal.Decimal
		ignoreValidation bool
	}
	type args struct {
		amount decimal.Decimal
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Balance
		wantErr bool
	}{
		{
			name: "success cancel reservation funds",
			fields: fields{
				actualBalance:    decimal.NewFromFloat(420.69),
				pendingBalance:   decimal.NewFromFloat(100),
				ignoreValidation: false,
			},
			args: args{
				amount: decimal.NewFromFloat(100),
			},
			want:    NewBalance(decimal.NewFromFloat(420.69), decimal.NewFromFloat(0)),
			wantErr: false,
		},
		{
			name: "success cancel reserve with insufficient balance by ignoring validation",
			fields: fields{
				actualBalance:    decimal.NewFromFloat(300),
				pendingBalance:   decimal.NewFromFloat(100),
				ignoreValidation: true,
			},
			args: args{
				amount: decimal.NewFromFloat(500),
			},
			want:    NewBalance(decimal.NewFromFloat(300), decimal.NewFromFloat(-400), WithIgnoreBalanceSufficiency()),
			wantErr: false,
		},
		{
			name: "error cancel reservation funds - invalid amount",
			fields: fields{
				actualBalance:    decimal.NewFromFloat(420.69),
				pendingBalance:   decimal.NewFromFloat(100),
				ignoreValidation: false,
			},
			args: args{
				amount: decimal.NewFromFloat(-100),
			},
			want:    NewBalance(decimal.NewFromFloat(420.69), decimal.NewFromFloat(100)),
			wantErr: true,
		},
		{
			name: "error cancel reservation funds - insufficient pending balance",
			fields: fields{
				actualBalance:    decimal.NewFromFloat(100),
				pendingBalance:   decimal.NewFromFloat(1),
				ignoreValidation: false,
			},
			args: args{
				amount: decimal.NewFromFloat(100),
			},
			want:    NewBalance(decimal.NewFromFloat(100), decimal.NewFromFloat(1)),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Balance{
				actualBalance:            tt.fields.actualBalance,
				pendingBalance:           tt.fields.pendingBalance,
				ignoreBalanceSufficiency: tt.fields.ignoreValidation,
			}
			if err := b.CancelReservation(tt.args.amount); (err != nil) != tt.wantErr {
				t.Errorf("AddFunds() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !cmp.Equal(&tt.want, b, balanceComparer()) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(&tt.want, b, balanceComparer()))
			}
		})
	}
}

func TestBalance_Commit(t *testing.T) {
	type fields struct {
		actualBalance    decimal.Decimal
		pendingBalance   decimal.Decimal
		ignoreValidation bool
	}
	type args struct {
		amount decimal.Decimal
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Balance
		wantErr bool
	}{
		{
			name: "success commit funds",
			fields: fields{
				actualBalance:    decimal.NewFromFloat(420.69),
				pendingBalance:   decimal.NewFromFloat(100),
				ignoreValidation: false,
			},
			args: args{
				amount: decimal.NewFromFloat(100),
			},
			want:    NewBalance(decimal.NewFromFloat(320.69), decimal.NewFromFloat(0)),
			wantErr: false,
		},
		{
			name: "success commit with insufficient balance by ignoring validation",
			fields: fields{
				actualBalance:    decimal.NewFromFloat(300),
				pendingBalance:   decimal.NewFromFloat(100),
				ignoreValidation: true,
			},
			args: args{
				amount: decimal.NewFromFloat(500),
			},
			want:    NewBalance(decimal.NewFromFloat(-200), decimal.NewFromFloat(-400), WithIgnoreBalanceSufficiency()),
			wantErr: false,
		},
		{
			name: "error commit funds - invalid amount",
			fields: fields{
				actualBalance:    decimal.NewFromFloat(420.69),
				pendingBalance:   decimal.NewFromFloat(100),
				ignoreValidation: false,
			},
			args: args{
				amount: decimal.NewFromFloat(-100),
			},
			want:    NewBalance(decimal.NewFromFloat(420.69), decimal.NewFromFloat(100)),
			wantErr: true,
		},
		{
			name: "error commit funds - insufficient pending balance",
			fields: fields{
				actualBalance:    decimal.NewFromFloat(100),
				pendingBalance:   decimal.NewFromFloat(1),
				ignoreValidation: false,
			},
			args: args{
				amount: decimal.NewFromFloat(100),
			},
			want:    NewBalance(decimal.NewFromFloat(100), decimal.NewFromFloat(1)),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Balance{
				actualBalance:            tt.fields.actualBalance,
				pendingBalance:           tt.fields.pendingBalance,
				ignoreBalanceSufficiency: tt.fields.ignoreValidation,
			}
			if err := b.Commit(tt.args.amount); (err != nil) != tt.wantErr {
				t.Errorf("AddFunds() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !cmp.Equal(&tt.want, b, balanceComparer()) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(&tt.want, b, balanceComparer()))
			}
		})
	}
}

func TestNewBalance(t *testing.T) {
	type args struct {
		actualBalance  decimal.Decimal
		pendingBalance decimal.Decimal
		options        []BalanceOption
	}
	tests := []struct {
		name string
		args args
		want Balance
	}{
		{
			name: "success create balance",
			args: args{
				actualBalance:  decimal.NewFromFloat(420.69),
				pendingBalance: decimal.NewFromFloat(100),
			},
			want: Balance{
				actualBalance:  decimal.NewFromFloat(420.69),
				pendingBalance: decimal.NewFromFloat(100),
			},
		},
		{
			name: "success create balance with ignore sufficiency balance validation",
			args: args{
				actualBalance:  decimal.NewFromFloat(420.69),
				pendingBalance: decimal.NewFromFloat(100),
				options:        []BalanceOption{WithIgnoreBalanceSufficiency()},
			},
			want: Balance{
				actualBalance:            decimal.NewFromFloat(420.69),
				pendingBalance:           decimal.NewFromFloat(100),
				ignoreBalanceSufficiency: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewBalance(tt.args.actualBalance, tt.args.pendingBalance, tt.args.options...)

			if !cmp.Equal(tt.want, got, balanceComparer()) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(&tt.want, got, balanceComparer()))
			}
		})
	}
}

func TestBalance_JSON(t *testing.T) {
	type fields struct {
		actualBalance            decimal.Decimal
		pendingBalance           decimal.Decimal
		ignoreBalanceSufficiency bool
	}
	tests := []struct {
		name    string
		fields  fields
		want    Balance
		wantErr bool
	}{
		{
			name: "success marshal and unmarshal balance",
			fields: fields{
				actualBalance:  decimal.NewFromFloat(420.69),
				pendingBalance: decimal.NewFromFloat(100),
			},
			want: NewBalance(decimal.NewFromFloat(420.69), decimal.NewFromFloat(100)),
		},
		{
			name: "success marshal and unmarshal with zero balance",
			fields: fields{
				actualBalance:  decimal.Zero,
				pendingBalance: decimal.Zero,
			},
			want: NewBalance(decimal.Zero, decimal.Zero),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := Balance{
				actualBalance:            tt.fields.actualBalance,
				pendingBalance:           tt.fields.pendingBalance,
				ignoreBalanceSufficiency: tt.fields.ignoreBalanceSufficiency,
			}
			gotRaw, err := json.Marshal(b)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			var got Balance
			err = json.Unmarshal(gotRaw, &got)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !cmp.Equal(tt.want, got, balanceComparer()) {
				t.Errorf("Result and Expected differ: (-got +want)\n%s", cmp.Diff(tt.want, got, balanceComparer()))
			}
		})
	}
}
