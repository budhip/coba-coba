package monitoring

import (
	"testing"
)

func Test_getSegmentName(t *testing.T) {
	type args struct {
		fullFuncName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "package.Receiver.Method",
			args: args{
				fullFuncName: "github.com/username/project/package.(*Receiver).Method",
			},
			want: "package.Receiver.Method",
		},
		{
			name: "package.Receiver.Method",
			args: args{
				fullFuncName: "github.com/username/project/package.Receiver.Method",
			},
			want: "package.Receiver.Method",
		},
		{
			name: "package.Function",
			args: args{
				fullFuncName: "github.com/username/project/package.Function",
			},
			want: "package.Function",
		},
		{
			name: "main.main.main",
			args: args{
				fullFuncName: "main.main.main",
			},
			want: "main.main.main",
		},
		{
			name: "main.main",
			args: args{
				fullFuncName: "main.main",
			},
			want: "main.main",
		},
		{
			name: "http.Server.Serve",
			args: args{
				fullFuncName: "net/http.(*Server).Serve",
			},
			want: "http.Server.Serve",
		},
		{
			name: "runtime.goexit",
			args: args{
				fullFuncName: "runtime.goexit",
			},
			want: "runtime.goexit",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getSegmentName(tt.args.fullFuncName); got != tt.want {
				t.Errorf("getSegmentName() = %v, want %v", got, tt.want)
			}
		})
	}
}
