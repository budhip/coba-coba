package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func cacheTestHelper(t *testing.T) (redismock.ClientMock, CacheRepository) {
	t.Helper()
	t.Parallel()

	db, mock := redismock.NewClientMock()
	cacheRepo := NewCacheRepository(db)

	return mock, cacheRepo
}

func TestCacheRepository_SetIfNotExists(t *testing.T) {
	mock, rc := cacheTestHelper(t)

	type args struct {
		key  string
		data interface{}
		ttl  time.Duration
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test success",
			args: args{
				key:  "123456789",
				data: "Success",
				ttl:  30 * time.Second,
			},
			want:    true,
			wantErr: false,
			doMock: func(args args) {
				mock.ExpectSetNX(args.key, args.data, args.ttl).SetVal(true)
			},
		},
		{
			name: "test error",
			args: args{
				key:  "123456789",
				data: "Success",
				ttl:  30 * time.Second,
			},
			wantErr: true,
			doMock: func(args args) {
				mock.ExpectSetNX(args.key, args.data, args.ttl).SetErr(redis.ErrClosed)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			got, err := rc.SetIfNotExists(context.TODO(), tt.args.key, tt.args.data, tt.args.ttl)
			assert.Equal(t, got, tt.want)
			assert.Equal(t, tt.wantErr, err != nil)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
			mock.ClearExpect()
		})
	}
}

func TestCacheRepository_Set(t *testing.T) {
	mock, rc := cacheTestHelper(t)

	type args struct {
		key  string
		data interface{}
		ttl  time.Duration
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test success",
			args: args{
				key:  "123456789",
				data: "Success",
				ttl:  30 * time.Second,
			},
			want:    true,
			wantErr: false,
			doMock: func(args args) {
				mock.ExpectSet(args.key, args.data, args.ttl).SetVal(args.key)
			},
		},
		{
			name: "test error",
			args: args{
				key:  "123456789",
				data: "Success",
				ttl:  30 * time.Second,
			},
			wantErr: true,
			doMock: func(args args) {
				mock.ExpectSet(args.key, args.data, args.ttl).SetErr(redis.ErrClosed)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			err := rc.Set(context.TODO(), tt.args.key, tt.args.data, tt.args.ttl)
			assert.Equal(t, tt.wantErr, err != nil)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
			mock.ClearExpect()
		})
	}
}

func TestCacheRepository_Get(t *testing.T) {
	mock, rc := cacheTestHelper(t)

	type args struct {
		key  string
		data string
		ttl  time.Duration
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test success",
			args: args{
				key:  "123456789",
				data: "Success",
				ttl:  30 * time.Second,
			},
			want:    "Success",
			wantErr: false,
			doMock: func(args args) {
				mock.ExpectGet(args.key).SetVal(args.data)
			},
		},
		{
			name: "test error - RedisNil",
			args: args{
				key:  "123456789",
				data: "Success",
				ttl:  30 * time.Second,
			},
			want:    "",
			wantErr: true,
			doMock: func(args args) {
				mock.ExpectGet(args.key).RedisNil()
			},
		},
		{
			name: "test error",
			args: args{
				key:  "123456789",
				data: "",
				ttl:  30 * time.Second,
			},
			want:    "",
			wantErr: true,
			doMock: func(args args) {
				mock.ExpectGet(args.key).SetErr(redis.ErrClosed)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			got, err := rc.Get(context.TODO(), tt.args.key)
			assert.Equal(t, got, tt.want)
			assert.Equal(t, err != nil, tt.wantErr)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
			mock.ClearExpect()
		})
	}
}

func TestCacheRepository_Del(t *testing.T) {
	mock, rc := cacheTestHelper(t)

	type args struct {
		key  string
		data string
		ttl  time.Duration
	}
	tests := []struct {
		name    string
		args    args
		doMock  func(args args)
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "test success",
			args: args{
				key: "123456789",
			},
			wantErr: false,
			doMock: func(args args) {
				mock.ExpectDel(args.key).SetVal(1)
			},
		},
		{
			name: "test error",
			args: args{
				key:  "123456789",
				data: "",
				ttl:  30 * time.Second,
			},
			wantErr: true,
			doMock: func(args args) {
				mock.ExpectDel(args.key).SetErr(redis.ErrClosed)
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			err := rc.Del(context.TODO(), tt.args.key)
			assert.Equal(t, err != nil, tt.wantErr)

			if err := mock.ExpectationsWereMet(); err != nil {
				t.Error(err)
			}
			mock.ClearExpect()
		})
	}
}
