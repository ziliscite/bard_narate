package service

import (
	"context"
	"errors"
	"github.com/ziliscite/bard_narate/gateway/internal/domain"
	"github.com/ziliscite/bard_narate/gateway/internal/repository"
	"github.com/ziliscite/bard_narate/gateway/pkg/encryptor"
)

type AudioService interface {
	Get(ctx context.Context, key string) (*domain.File, error)
}

type audioService struct {
	bucket string
	enc    *encryptor.Encryptor
	fs     repository.FileReader
}

func NewAudioService(fs repository.FileReader, audioBucket string) AudioService {
	return &audioService{
		bucket: audioBucket,
		fs:     fs,
	}
}

func (a *audioService) Get(ctx context.Context, key string) (*domain.File, error) {
	file, err := a.fs.ReadLarge(ctx, a.bucket, key)
	if err != nil {
		return nil, err
	}

	if file == nil {
		return nil, errors.New("file not found")
	}

	return file, nil
}
