package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/ziliscite/bard_narate/gateway/internal/service"
	pb "github.com/ziliscite/bard_narate/gateway/pkg/protobuf"
	"net/http"
)

type Converter interface {
	// TextToAudio should take a multipart request of a text file and return a job id.
	//
	// Pipeline as follows:
	//
	// create new job ->
	// send file to S3 ->
	// queue a conversion job ->
	// update job to processing ->
	// return job id to client
	TextToAudio(c *gin.Context)
	JobStatus(c *gin.Context)
}

type converter struct {
	ts  service.TextService
	ps  service.Publisher
	jsc pb.JobServiceClient
}

func NewConverter(ts service.TextService, ps service.Publisher, jsc pb.JobServiceClient) Converter {
	// r.MaxMultipartMemory = 1 << 30 // 1GB
	return &converter{
		ts:  ts,
		ps:  ps,
		jsc: jsc,
	}
}

func (cv *converter) TextToAudio(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get file"})
		return
	}

	// check header
	if file.Header == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to get file header"})
		return
	}

	// check the content type
	mimeType := file.Header.Get("Content-Type")
	if mimeType != "text/plain" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file type. must be text/plain"})
		return
	}

	txt, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer txt.Close()

	key, err := cv.ts.Save(c.Request.Context(), file.Filename, txt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save text to S3"})
		return
	}

	// create a new job, take from other grpc serv
	resp, err := cv.jsc.New(c.Request.Context(), &pb.NewJobRequest{
		FileKey: key,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create job"})
		return
	}

	// publish to file exchange
	// this should be consumed by tts service AND job update service
	if err = cv.ps.PublishConversion(c.Request.Context(), resp.Job.Id, resp.Job.FileKey); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to publish job"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"id": resp.Job.Id})
}

func (cv *converter) JobStatus(c *gin.Context) {
	id := c.Param("id")
	resp, err := cv.jsc.Get(c.Request.Context(), &pb.GetJobRequest{
		Id: id,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get job status"})
		return
	}

	if resp.Job.Status != pb.Status_Completed {
		c.JSON(http.StatusAccepted, gin.H{"status": resp.Job.Status.String()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       resp.Job.Id,
		"status":   resp.Job.Status.String(),
		"file_key": resp.Job.FileKey,
	})
}

// DownloadAudio download audio file from s3 using the filekey
func (cv *converter) DownloadAudio(c *gin.Context) {
	id := c.Param("id")
	_, err := cv.jsc.Get(c.Request.Context(), &pb.GetJobRequest{
		Id: id,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get job status"})
		return
	}
}
