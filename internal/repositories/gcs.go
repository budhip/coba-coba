package repositories

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"google.golang.org/api/option"

	"bitbucket.org/Amartha/go-fp-transaction/internal/config"
	"bitbucket.org/Amartha/go-fp-transaction/internal/models"

	"cloud.google.com/go/storage"
)

type CloudStorageRepository interface {
	NewWriter(ctx context.Context, payload *models.CloudStoragePayload) io.WriteCloser
	NewWriterCustom(ctx context.Context, bucketName string, payload *models.CloudStoragePayload) io.WriteCloser
	NewReader(ctx context.Context, payload *models.CloudStoragePayload) (io.ReadCloser, error)
	WriteStream(ctx context.Context, payload *models.CloudStoragePayload, data <-chan []byte) models.WriteStreamResult
	WriteStreamCustomBucket(ctx context.Context, bucketName string, payload *models.CloudStoragePayload, data <-chan []byte) models.WriteStreamResult
	GetURL(payload *models.CloudStoragePayload) (url string)
	GetSignedURL(filePath string, expireDuration time.Duration) (url string, err error)
	Close() error
	DeleteFile(ctx context.Context, payload *models.CloudStoragePayload) error
	IsObjectExist(ctx context.Context, payload *models.CloudStoragePayload) (isExist bool, url string)
	NewReaderBucketCustom(ctx context.Context, bucket, dirFileName string) (io.ReadCloser, error)
}

type cloudStorageClient struct {
	config *config.CloudStorageConfig
	client *storage.Client
}

func NewCloudStorageRepository(cfg *config.Config, opts ...option.ClientOption) (CloudStorageRepository, error) {
	if cfg.CloudStorageConfig.BucketName == "" {
		return nil, fmt.Errorf("failed to init cloud storage bucket name not set")
	}

	client, err := storage.NewClient(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	return &cloudStorageClient{client: client, config: &cfg.CloudStorageConfig}, nil
}

func (cs *cloudStorageClient) GetURL(payload *models.CloudStoragePayload) (url string) {
	dirWithFilename := fmt.Sprintf("%s/%s", payload.Path, payload.Filename)
	return fmt.Sprintf("%s/%s/%s", cs.config.BaseURL, cs.config.BucketName, dirWithFilename)
}

func (cs *cloudStorageClient) NewWriter(ctx context.Context, payload *models.CloudStoragePayload) io.WriteCloser {
	dirWithFilename := fmt.Sprintf("%s/%s", payload.Path, payload.Filename)
	obj := cs.client.Bucket(cs.config.BucketName).Object(dirWithFilename)
	writer := obj.NewWriter(ctx)
	writer.ContentDisposition = fmt.Sprintf("attachment; filename=%s", payload.Filename)
	return writer
}

func (cs *cloudStorageClient) NewWriterCustom(ctx context.Context, bucketName string, payload *models.CloudStoragePayload) io.WriteCloser {
	dirWithFilename := fmt.Sprintf("%s/%s", payload.Path, payload.Filename)
	obj := cs.client.Bucket(bucketName).Object(dirWithFilename)
	writer := obj.NewWriter(ctx)
	writer.ContentDisposition = fmt.Sprintf("attachment; filename=%s", payload.Filename)
	return writer
}

func (cs *cloudStorageClient) WriteStreamCustomBucket(ctx context.Context, bucketName string, payload *models.CloudStoragePayload, data <-chan []byte) models.WriteStreamResult {
	ch := make(chan error)
	dirWithFilename := fmt.Sprintf("%s/%s", payload.Path, payload.Filename)
	r := models.NewWriteStreamResult(ch, fmt.Sprintf("%s/%s/%s", cs.config.BaseURL, bucketName, dirWithFilename))

	go func() {
		writer := cs.NewWriterCustom(ctx, bucketName, payload)
		defer func() {
			if err := writer.Close(); err != nil {
				ch <- err
			}
			close(ch)
		}()

		for v := range data {
			select {
			case <-ctx.Done():
				return
			default:
				if _, err := writer.Write(v); err != nil {
					ch <- err
				}
			}
		}
	}()

	return r
}

func (cs *cloudStorageClient) WriteStream(ctx context.Context, payload *models.CloudStoragePayload, data <-chan []byte) models.WriteStreamResult {
	ch := make(chan error)
	dirWithFilename := fmt.Sprintf("%s/%s", payload.Path, payload.Filename)
	r := models.NewWriteStreamResult(ch, fmt.Sprintf("%s/%s/%s", cs.config.BaseURL, cs.config.BucketName, dirWithFilename))

	go func() {
		writer := cs.NewWriter(ctx, payload)
		defer func() {
			if err := writer.Close(); err != nil {
				ch <- err
			}
			close(ch)
		}()

		for v := range data {
			if _, err := writer.Write(v); err != nil {
				ch <- err
			}
		}
	}()

	return r
}

func (cs *cloudStorageClient) Close() error {
	return cs.client.Close()
}

func (cs *cloudStorageClient) DeleteFile(ctx context.Context, payload *models.CloudStoragePayload) error {
	dirWithFilename := fmt.Sprintf("%s/%s", payload.Path, payload.Filename)
	obj := cs.client.Bucket(cs.config.BucketName).Object(dirWithFilename)
	return obj.Delete(ctx)
}

func (cs *cloudStorageClient) IsObjectExist(ctx context.Context, payload *models.CloudStoragePayload) (isExist bool, url string) {
	dirWithFilename := fmt.Sprintf("%s/%s", payload.Path, payload.Filename)
	_, err := cs.client.Bucket(cs.config.BucketName).Object(dirWithFilename).Attrs(ctx)
	if err == nil {
		isExist = true
		url = fmt.Sprintf("%s/%s/%s", cs.config.BaseURL, cs.config.BucketName, dirWithFilename)
	}

	return
}

func (cs *cloudStorageClient) GetSignedURL(filePath string, expireDuration time.Duration) (url string, err error) {
	url, err = cs.client.Bucket(cs.config.BucketName).SignedURL(filePath, &storage.SignedURLOptions{
		Method:  "GET",
		Expires: time.Now().Add(expireDuration),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get signed url: %w", err)
	}

	return
}

func (cs *cloudStorageClient) NewReader(ctx context.Context, payload *models.CloudStoragePayload) (io.ReadCloser, error) {
	dirWithFilename := fmt.Sprintf("%s/%s", payload.Path, payload.Filename)

	rc, err := cs.client.Bucket(cs.config.BucketName).Object(dirWithFilename).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open object in bucket: %v", err)
	}

	return rc, nil
}

func (cs *cloudStorageClient) NewReaderBucketCustom(ctx context.Context, bucket, dirFileName string) (io.ReadCloser, error) {
	log.Printf("dir file name %v", dirFileName)
	rc, err := cs.client.Bucket(bucket).Object(dirFileName).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to open object in bucket: %v", err)
	}

	return rc, nil
}
