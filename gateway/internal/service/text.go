package service

import (
	"context"
	"github.com/ziliscite/bard_narate/gateway/internal/domain"
	"github.com/ziliscite/bard_narate/gateway/internal/repository"
	"github.com/ziliscite/bard_narate/gateway/pkg/encryptor"
	"io"
)

type TextService interface {
	// Save saves the file to the bucket and returns the key.
	// The S3 key that is used to store the file is an unencrypted filename.
	// The returned key is the encrypted filename.
	Save(ctx context.Context, filename string, file io.Reader) (string, error)

	// Get retrieves the file from the bucket using the key.
	// The key is the encrypted filename.
	// Decrypt the key to get the original filename.
	Get(ctx context.Context, key string) (*domain.File, error)
}

type textService struct {
	bucket string
	enc    *encryptor.Encryptor
	fs     repository.SmallFileStore
}

func NewTextService(fs repository.SmallFileStore, textBucket string) TextService {
	return &textService{
		bucket: textBucket,
		fs:     fs,
	}
}

func (t *textService) Save(ctx context.Context, filename string, file io.Reader) (string, error) {
	// encrypt the filename to get the key
	key, err := t.enc.Encrypt(filename)
	if err != nil {
		return "", err
	}

	// create a new file with the filename
	txt := domain.NewFile(filename, "text/plain", file)
	if err = t.fs.Save(ctx, t.bucket, txt); err != nil {
		return "", err
	}

	return key, nil
}

func (t *textService) Get(ctx context.Context, key string) (*domain.File, error) {
	// decrypt the key to get the original filename
	filename, err := t.enc.Decrypt(key)
	if err != nil {
		return nil, err
	}

	// read the file from the bucket
	return t.fs.Read(ctx, t.bucket, string(filename))
}
