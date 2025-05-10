package service

import (
	"context"
	"github.com/ziliscite/bard_narate/job/internal/domain"
	"github.com/ziliscite/bard_narate/job/internal/repository"
)

type JobService interface {
	New(ctx context.Context, fileKey string) (*domain.Job, error)
	Get(ctx context.Context, id string) (*domain.Job, error)
	Update(ctx context.Context, job *domain.Job) error
}

type jobService struct {
	jr repository.JobRepository
}

func NewJobService(jr repository.JobRepository) JobService {
	return &jobService{
		jr: jr,
	}
}

func (js *jobService) New(ctx context.Context, fileKey string) (*domain.Job, error) {
	job := domain.NewJob(fileKey)
	if err := js.jr.Save(ctx, job); err != nil {
		return nil, err
	}

	return job, nil
}

func (js *jobService) Get(ctx context.Context, id string) (*domain.Job, error) {
	return js.jr.Load(ctx, id)
}

func (js *jobService) UpdateStatus(ctx context.Context, id string, status domain.JobStatus) error {
	job, err := js.jr.Load(ctx, id)
	if err != nil {
		return err
	}

	job.SetStatus(status)
	return js.jr.Update(ctx, job)
}

func (js *jobService) Update(ctx context.Context, job *domain.Job) error {
	return js.jr.Update(ctx, job)
}
