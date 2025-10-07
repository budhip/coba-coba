package repositories

import (
	"context"
	"fmt"
	"testing"
	"time"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"cloud.google.com/go/storage"
	"github.com/fsouza/fake-gcs-server/fakestorage"
	"github.com/stretchr/testify/assert"
	"google.golang.org/api/option"
)

type gcsHelper struct {
	server        *fakestorage.Server
	client        *storage.Client
	defaultConfig *config.CloudStorageConfig
}

func newGcsClientHelper(t *testing.T) *gcsHelper {
	t.Helper()
	t.Parallel()

	server, err := fakestorage.NewServerWithOptions(fakestorage.Options{
		NoListener: true,
	})
	assert.NoError(t, err)

	client, err := storage.NewClient(
		context.Background(),
		option.WithoutAuthentication(),
		option.WithHTTPClient(server.HTTPClient()))
	assert.NoError(t, err)

	return &gcsHelper{
		server: server,
		client: client,
		defaultConfig: &config.CloudStorageConfig{
			BaseURL:    "http://test:1337",
			BucketName: "DUMMY_BUCKET",
		},
	}
}

func TestNewCloudStorageRepository(t *testing.T) {
	helper := newGcsClientHelper(t)

	type args struct {
		cfg  *config.Config
		opts []option.ClientOption
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success init cloud storage",
			args: args{
				cfg: &config.Config{
					App: config.App{
						Env:  "test",
						Name: "go-fp-transaction[test]",
					},
					CloudStorageConfig: *helper.defaultConfig,
				},
				opts: []option.ClientOption{
					option.WithoutAuthentication(),
					option.WithHTTPClient(helper.server.HTTPClient()),
				},
			},
			wantErr: false,
		},
		{
			name: "failed init cloud storage (bucket name not set)",
			args: args{
				cfg: &config.Config{
					App: config.App{
						Env:  "test",
						Name: "go-fp-transaction[test]",
					},
					CloudStorageConfig: config.CloudStorageConfig{
						BaseURL:    "",
						BucketName: "",
					},
				},
				opts: []option.ClientOption{
					option.WithoutAuthentication(),
					option.WithHTTPClient(helper.server.HTTPClient()),
				},
			},
			wantErr: true,
		},
		{
			name: "failed init cloud storage (no option provided for testing)",
			args: args{
				cfg: &config.Config{
					App: config.App{
						Env:  "test",
						Name: "go-fp-transaction[test]",
					},
					CloudStorageConfig: *helper.defaultConfig,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCloudStorageRepository(tt.args.cfg, tt.args.opts...)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_cloudStorageClient_Close(t *testing.T) {
	helper := newGcsClientHelper(t)

	type fields struct {
		config *config.CloudStorageConfig
		client *storage.Client
	}
	tests := []struct {
		name    string
		fields  fields
		doMock  func(f fields)
		wantErr bool
	}{
		{
			name: "success close",
			fields: fields{
				config: helper.defaultConfig,
				client: helper.client,
			},
			doMock: func(f fields) {
				helper.server.CreateBucketWithOpts(fakestorage.CreateBucketOpts{
					Name: f.config.BucketName,
				})
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.fields)
			}

			cs := &cloudStorageClient{
				config: tt.fields.config,
				client: tt.fields.client,
			}

			err := cs.Close()
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_cloudStorageClient_DeleteFile(t *testing.T) {
	helper := newGcsClientHelper(t)

	type fields struct {
		config *config.CloudStorageConfig
		client *storage.Client
	}
	type args struct {
		ctx     context.Context
		payload *models.CloudStoragePayload
	}
	tests := []struct {
		name    string
		fields  fields
		doMock  func(a args)
		args    args
		wantErr bool
	}{
		{
			name: "success delete file",
			fields: fields{
				config: helper.defaultConfig,
				client: helper.client,
			},
			doMock: func(a args) {
				helper.server.CreateObject(fakestorage.Object{
					ObjectAttrs: fakestorage.ObjectAttrs{
						BucketName: helper.defaultConfig.BucketName,
						Name:       fmt.Sprintf("%s/%s", a.payload.Path, a.payload.Filename),
					},
					Content: []byte("test"),
				})
			},
			args: args{
				ctx: context.TODO(),
				payload: &models.CloudStoragePayload{
					Filename: "a.txt",
					Path:     "test",
				},
			},
			wantErr: false,
		},
		{
			name: "failed delete file",
			fields: fields{
				config: helper.defaultConfig,
				client: helper.client,
			},
			doMock: func(a args) {
			},
			args: args{
				ctx: context.TODO(),
				payload: &models.CloudStoragePayload{
					Filename: "non_existent_file.txt",
					Path:     "non_existent_path",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			cs := &cloudStorageClient{
				config: tt.fields.config,
				client: tt.fields.client,
			}
			err := cs.DeleteFile(tt.args.ctx, tt.args.payload)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func Test_cloudStorageClient_GetURL(t *testing.T) {
	helper := newGcsClientHelper(t)

	type fields struct {
		config *config.CloudStorageConfig
		client *storage.Client
	}
	type args struct {
		payload *models.CloudStoragePayload
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantUrl string
	}{
		{
			name: "success get url",
			fields: fields{
				config: helper.defaultConfig,
				client: helper.client,
			},
			args: args{
				payload: &models.CloudStoragePayload{
					Filename: "test.txt",
					Path:     "my_path",
				},
			},
			wantUrl: fmt.Sprintf("%s/%s/my_path/test.txt", helper.defaultConfig.BaseURL, helper.defaultConfig.BucketName),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &cloudStorageClient{
				config: tt.fields.config,
				client: tt.fields.client,
			}
			assert.Equalf(t, tt.wantUrl, cs.GetURL(tt.args.payload), "GetURL(%v)", tt.args.payload)
		})
	}
}

func Test_cloudStorageClient_IsObjectExist(t *testing.T) {
	helper := newGcsClientHelper(t)

	type fields struct {
		config *config.CloudStorageConfig
		client *storage.Client
	}
	type args struct {
		ctx     context.Context
		payload *models.CloudStoragePayload
	}
	tests := []struct {
		name        string
		fields      fields
		doMock      func(a args)
		args        args
		wantIsExist bool
		wantUrl     string
	}{
		{
			name: "success is object exist",
			fields: fields{
				config: helper.defaultConfig,
				client: helper.client,
			},
			args: args{
				ctx: context.TODO(),
				payload: &models.CloudStoragePayload{
					Filename: "test.txt",
					Path:     "my_path",
				},
			},
			doMock: func(a args) {
				helper.server.CreateObject(fakestorage.Object{
					ObjectAttrs: fakestorage.ObjectAttrs{
						BucketName: helper.defaultConfig.BucketName,
						Name:       fmt.Sprintf("%s/%s", a.payload.Path, a.payload.Filename),
					},
					Content: []byte("test"),
				})
			},
			wantIsExist: true,
			wantUrl:     fmt.Sprintf("%s/%s/my_path/test.txt", helper.defaultConfig.BaseURL, helper.defaultConfig.BucketName),
		},
		{
			name: "success but object doesn't exists",
			fields: fields{
				config: helper.defaultConfig,
				client: helper.client,
			},
			args: args{
				ctx: context.TODO(),
				payload: &models.CloudStoragePayload{
					Filename: "non_existent_file.txt",
					Path:     "non_existent_path",
				},
			},
			doMock: func(a args) {
			},
			wantIsExist: false,
			wantUrl:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(tt.args)
			}

			cs := &cloudStorageClient{
				config: tt.fields.config,
				client: tt.fields.client,
			}
			gotIsExist, gotUrl := cs.IsObjectExist(tt.args.ctx, tt.args.payload)
			assert.Equalf(t, tt.wantIsExist, gotIsExist, "IsObjectExist(%v, %v)", tt.args.ctx, tt.args.payload)
			assert.Equalf(t, tt.wantUrl, gotUrl, "IsObjectExist(%v, %v)", tt.args.ctx, tt.args.payload)
		})
	}
}

func Test_cloudStorageClient_NewWriter(t *testing.T) {
	helper := newGcsClientHelper(t)

	type fields struct {
		config *config.CloudStorageConfig
		client *storage.Client
	}
	type args struct {
		ctx     context.Context
		payload *models.CloudStoragePayload
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "success create new writer",
			fields: fields{
				config: helper.defaultConfig,
				client: helper.client,
			},
			args: args{
				ctx: context.TODO(),
				payload: &models.CloudStoragePayload{
					Filename: "my_writer.txt",
					Path:     "my_path_for_writer",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cs := &cloudStorageClient{
				config: tt.fields.config,
				client: tt.fields.client,
			}
			assert.NotNilf(t, cs.NewWriter(tt.args.ctx, tt.args.payload), "NewWriter(%v, %v)", tt.args.ctx, tt.args.payload)
		})
	}
}

func Test_cloudStorageClient_WriteStream(t *testing.T) {
	helper := newGcsClientHelper(t)

	type fields struct {
		config *config.CloudStorageConfig
		client *storage.Client
	}
	type args struct {
		ctx     context.Context
		payload *models.CloudStoragePayload
		data    <-chan []byte
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		doMock func(a *args)
	}{
		{
			name: "success init stream channel",
			fields: fields{
				config: helper.defaultConfig,
				client: helper.client,
			},
			args: args{
				ctx: context.TODO(),
				payload: &models.CloudStoragePayload{
					Filename: "my_stream_writer.txt",
					Path:     "my_path_for_stream_writer",
				},
			},
			doMock: func(a *args) {
				chanData := make(chan []byte)
				go func() {
					time.Sleep(100 * time.Millisecond)
					chanData <- []byte("test")
					close(chanData)
				}()
				a.data = chanData
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(&tt.args)
			}

			cs := &cloudStorageClient{
				config: tt.fields.config,
				client: tt.fields.client,
			}
			assert.NotNilf(t, cs.WriteStream(tt.args.ctx, tt.args.payload, tt.args.data), "WriteStream(%v, %v, %v)", tt.args.ctx, tt.args.payload, tt.args.data)
		})
	}
}

func Test_cloudStorageClient_GetSignedURL(t *testing.T) {
	helper := newGcsClientHelper(t)
	defaultDuration := 5 * time.Minute

	type fields struct {
		config *config.CloudStorageConfig
		client *storage.Client
	}
	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		doMock  func(a *args)
		wantErr bool
	}{
		{
			name: "failed get signed url",
			fields: fields{
				config: helper.defaultConfig,
				client: helper.client,
			},
			args: args{
				filePath: "my_path/file_recon_history_that_not_exists.txt",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.doMock != nil {
				tt.doMock(&tt.args)
			}

			cs := &cloudStorageClient{
				config: tt.fields.config,
				client: tt.fields.client,
			}
			_, err := cs.GetSignedURL(tt.args.filePath, defaultDuration)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}
