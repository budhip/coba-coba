package safeaccess

import (
	"context"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"

	"bitbucket.org/Amartha/go-fp-transaction/internal/models"
)

type gcsHelper struct {
	server *fakestorage.Server
	client *storage.Client
	obj    *storage.ObjectHandle
}

func newGCSHelper(t *testing.T) *gcsHelper {
	t.Helper()

	server, err := fakestorage.NewServerWithOptions(fakestorage.Options{
		NoListener: true,
	})
	assert.NoError(t, err)

	server.CreateBucketWithOpts(fakestorage.CreateBucketOpts{Name: "DUMMY_BUCKET"})

	client, err := storage.NewClient(
		context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(server.HTTPClient()))
	assert.NoError(t, err)

	return &gcsHelper{
		server: server,
		client: client,
		obj:    client.Bucket("DUMMY_BUCKET").Object("DUMMY_PATH.json"),
	}
}

func TestGCSJson_LoadFile(t *testing.T) {
	helper := newGCSHelper(t)
	defer helper.server.Stop()

	type args struct {
		ctx context.Context
	}
	type testCase[T any] struct {
		name    string
		g       GCSJson[T]
		doMock  func()
		args    args
		wantErr bool
	}
	tests := []testCase[models.TransactionType]{
		{
			name: "success load file from GCS",
			doMock: func() {
				w := helper.obj.NewWriter(context.Background())
				_, err := w.Write([]byte(`{"transactionTypeCode": "TUPVA", "transactionTypeName": "Topup via VA"}`))
				assert.NoError(t, err)
				err = w.Close()
				assert.NoError(t, err)
			},
			g: GCSJson[models.TransactionType]{
				object: helper.obj,
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
		{
			name: "failed load file from GCS",
			doMock: func() {
				w := helper.obj.NewWriter(context.Background())
				_, err := w.Write([]byte(`-{---INVALID_JSON---}-`))
				assert.NoError(t, err)
				err = w.Close()
				assert.NoError(t, err)
			},
			g: GCSJson[models.TransactionType]{
				object: helper.obj,
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock()
			}

			if err := tt.g.LoadFile(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("LoadFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGCSJson_UpdateFile(t *testing.T) {
	helper := newGCSHelper(t)
	defer helper.server.Stop()

	type args struct {
		ctx context.Context
	}
	type testCase[T any] struct {
		name    string
		g       GCSJson[T]
		args    args
		wantErr bool
	}
	tests := []testCase[models.TransactionType]{
		{
			name: "success update file",
			g: GCSJson[models.TransactionType]{
				object: helper.obj,
				val: Value[models.TransactionType]{
					data: models.TransactionType{
						TransactionTypeCode: "TUPVA",
						TransactionTypeName: "Topup via VA",
					},
				},
			},
			args: args{
				ctx: context.Background(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.g.UpdateFile(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("UpdateFile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
