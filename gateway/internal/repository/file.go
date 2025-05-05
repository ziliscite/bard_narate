package repository

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ziliscite/bard_narate/gateway/internal/domain"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

var partSize int64 = 10 << 20 // 10 MB

type SmallFileWriter interface {
	Save(ctx context.Context, bucket string, file *domain.File) error
}

type LargeFileWriter interface {
	SaveLarge(ctx context.Context, bucket string, file *domain.File) error
}

type FileWriter interface {
	SmallFileWriter
	LargeFileWriter
}

type SmallFileReader interface {
	// Read reads the file from the bucket.
	Read(ctx context.Context, bucket string, key string) (*domain.File, error)
}

type LargeFileReader interface {
	ReadLarge(ctx context.Context, bucket string, key string) (*domain.File, error)
}

type FileReader interface {
	SmallFileReader
	LargeFileReader
}

type FileDeleter interface {
	Delete(ctx context.Context, bucket string, key string) error
}

type SmallFileStore interface {
	SmallFileReader
	SmallFileWriter
	FileDeleter
}

type FileStore interface {
	FileWriter
	FileReader
	FileDeleter
}

type store struct {
	s3c *s3.Client
}

func NewStore(s3c *s3.Client) FileStore {
	return &store{
		s3c: s3c,
	}
}

// Save saves the file to an object in a bucket.
func (s *store) Save(ctx context.Context, bucket string, file *domain.File) error {
	if _, err := s.s3c.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(file.Name()),
		Body:        file.Body(),
		ContentType: aws.String(file.Type()),
	}); err != nil {
		return fmt.Errorf("failed to upload file %s to bucket %s: %w", file.Name(), bucket, err)
	}

	if err := s3.NewObjectExistsWaiter(s.s3c).Wait(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file.Name()),
	}, time.Minute); err != nil {
		return fmt.Errorf("failed to confirm existence of uploaded file %s in bucket %s: %w", file.Name(), bucket, err)
	}

	return nil
}

// SaveLarge uses an upload manager to upload data to an object in a bucket.
// The upload manager breaks large data into parts and uploads the parts concurrently.
func (s *store) SaveLarge(ctx context.Context, bucket string, file *domain.File) error {
	var size int64 = 10 << 20 // 10 MB
	uploader := manager.NewUploader(s.s3c, func(u *manager.Uploader) {
		u.PartSize = size
	})

	if _, err := uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(file.Name()),
		Body:        file.Body(),
		ContentType: aws.String(file.Type()),
	}); err != nil {
		var apiErr smithy.APIError
		errors.As(err, &apiErr)

		switch {
		case apiErr.ErrorCode() == "EntityTooLarge":
			return fmt.Errorf("file exceeds maximum size of 5TB for multipart upload to bucket %s: %w", bucket, err)
		default:
			return fmt.Errorf("failed to upload file %s to bucket %s: %w", file.Name(), bucket, err)
		}
	}

	if err := s3.NewObjectExistsWaiter(s.s3c).Wait(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(file.Name()),
	}, time.Minute); err != nil {
		return fmt.Errorf("failed to confirm existence of uploaded file %s in bucket %s: %w", file.Name(), bucket, err)
	}

	return nil
}

func (s *store) Read(ctx context.Context, bucket string, fileKey string) (*domain.File, error) {
	result, err := s.s3c.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileKey),
	})

	if err != nil {
		var noKey *types.NoSuchKey
		errors.As(err, &noKey)
		switch {
		case errors.As(err, &noKey):
			return nil, ErrNotExist
		default:
			return nil, fmt.Errorf("failed to read object %s from bucket %s: %w", fileKey, bucket, err)
		}
	}

	if result.ContentLength == nil {
		return nil, fmt.Errorf("failed to read object %s from bucket %s: %w", fileKey, bucket, err)
	}

	if *result.ContentLength == 0 {
		return nil, fmt.Errorf("failed to read object %s from bucket %s: %w", fileKey, bucket, err)
	}

	if result.ContentType == nil {
		return nil, fmt.Errorf("failed to read object %s from bucket %s: %w", fileKey, bucket, err)
	}

	if *result.ContentType == "" {
		return nil, fmt.Errorf("failed to read object %s from bucket %s: %w", fileKey, bucket, err)
	}

	return domain.NewFile(fileKey, *result.ContentType, result.Body), nil
}

// ReadLarge uses a download manager to download an object from a bucket.
// The download manager gets the data in parts and writes them to a buffer until all of
// the data has been downloaded.
func (s *store) ReadLarge(ctx context.Context, bucket string, fileKey string) (*domain.File, error) {
	downloader := manager.NewDownloader(s.s3c, func(d *manager.Downloader) {
		d.PartSize = partSize
	})

	headObject, err := s.s3c.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileKey),
	})
	if err != nil {
		var noKey *types.NoSuchKey
		errors.As(err, &noKey)
		switch {
		case errors.As(err, &noKey):
			return nil, ErrNotExist
		default:
			return nil, fmt.Errorf("failed to head object %s from bucket %s: %w", fileKey, bucket, err)
		}
	}

	if headObject.ContentType == nil {
		return nil, fmt.Errorf("failed to download object %s from bucket %s: %w", fileKey, bucket, err)
	}

	if *headObject.ContentType == "" {
		return nil, fmt.Errorf("failed to download object %s from bucket %s: %w", fileKey, bucket, err)
	}

	buffer := manager.NewWriteAtBuffer([]byte{})

	if _, err = downloader.Download(ctx, buffer, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileKey),
	}); err != nil {
		return nil, fmt.Errorf("failed to download object %s from bucket %s: %w", fileKey, bucket, err)
	}

	return domain.NewFile(fileKey, *headObject.ContentType, bytes.NewReader(buffer.Bytes())), nil
}

func (s *store) Delete(ctx context.Context, bucket string, key string) error {
	if _, err := s.s3c.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}); err != nil {
		var noKey *types.NoSuchKey
		errors.As(err, &noKey)
		switch {
		case errors.As(err, &noKey):
			return ErrNotExist
		default:
			return fmt.Errorf("failed to delete object %s from bucket %s: %w", key, bucket, err)
		}
	}

	if err := s3.NewObjectExistsWaiter(s.s3c).Wait(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, time.Minute); err != nil {
		return fmt.Errorf("failed attempt to wait for object %s in bucket %s to be deleted", key, bucket)
	}

	return nil
}
