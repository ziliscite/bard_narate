package main

import (
	"github.com/gin-gonic/gin"
	"github.com/ziliscite/bard_narate/job/internal/domain"
	"github.com/ziliscite/bard_narate/job/internal/service"
	"net/http"
)

type GetJobRequest struct {
	ID string `uri:"id" binding:"required"`
}

type GetJobResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

//type GetJobComplete struct {
//	ID      string `json:"id"`
//	Status  string `json:"status"`
//	FileKey string `json:"file_key"`
//}

func (s *HttpServer) GetJob(ctx *gin.Context) {
	var req GetJobRequest
	if err := ctx.BindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job, err := s.jobService.Get(ctx, req.ID)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	if job.Status != domain.Completed {
		ctx.JSON(http.StatusAccepted, GetJobResponse{
			ID:     job.ID,
			Status: job.Status.String(),
		})
		return
	}

	ctx.JSON(http.StatusOK, job)
}

func NewServer(jobService service.JobService) *HttpServer {
	return &HttpServer{
		jobService: jobService,
	}
}

type HttpServer struct {
	jobService service.JobService
}

func (s *HttpServer) Register(r *gin.Engine) {
	r.GET("/jobs/:id", s.GetJob)
}
