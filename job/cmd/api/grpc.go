package main

import (
	"context"
	"github.com/ziliscite/bard_narate/job/internal/service"
	pb "github.com/ziliscite/bard_narate/job/pkg/protobuf"
)

type Server struct {
	c  Config
	js service.JobService
	pb.UnimplementedJobServiceServer
}

func NewGRPCServer(c Config, js service.JobService) *Server {
	return &Server{
		c:  c,
		js: js,
	}
}

func (s *Server) New(ctx context.Context, req *pb.NewJobRequest) (*pb.NewJobResponse, error) {
	job, err := s.js.New(ctx, req.FileKey)
	if err != nil {
		return nil, err
	}

	return &pb.NewJobResponse{
		Job: &pb.Job{
			Id:      job.ID,
			Status:  pb.Status(job.Status),
			FileKey: job.FileKey,
		},
	}, nil
}

func (s *Server) Get(ctx context.Context, req *pb.GetJobRequest) (*pb.GetJobResponse, error) {
	job, err := s.js.Get(ctx, req.Id)
	if err != nil {
		return nil, err
	}

	// might need to check if status is complete or not
	// cuz if not, then file key should not be returned
	// or idk; maybe js handle it in the gateway
	return &pb.GetJobResponse{
		Job: &pb.Job{
			Id:      job.ID,
			Status:  pb.Status(job.Status),
			FileKey: job.FileKey,
		},
	}, nil
}
