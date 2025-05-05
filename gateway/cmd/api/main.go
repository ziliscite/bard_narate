package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/ziliscite/bard_narate/gateway/internal/controller"
	"github.com/ziliscite/bard_narate/gateway/internal/repository"
	"github.com/ziliscite/bard_narate/gateway/internal/service"
	pb "github.com/ziliscite/bard_narate/gateway/pkg/protobuf"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log/slog"
	"os"
)

func main() {
	cfg := getConfig()

	s3c := s3.NewFromConfig(aws.Config{
		Region: cfg.aws.s3Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.aws.accessKeyId,
			cfg.aws.secretAccessKey,
			"",
		),
	})
	fs := repository.NewStore(s3c)
	ts := service.NewTextService(fs, cfg.aws.s3bucket.text)

	conn, err := amqp.Dial(cfg.rabbit.dsn())
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	defer conn.Close()

	ps, err := service.NewPublisher(conn, cfg.rabbit.exchange, cfg.rabbit.route.text)
	if err != nil {
		panic(err)
	}

	jobClient, err := grpc.NewClient(fmt.Sprintf("%s:%s", cfg.grpc.job.host, cfg.grpc.job.port), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("Failed to connect to token service client", "error", err)
		os.Exit(1)
	}
	defer jobClient.Close()
	jsc := pb.NewJobServiceClient(jobClient)

	cv := controller.NewConverter(ts, ps, jsc)

	router := gin.New()
	router.MaxMultipartMemory = 1 << 30 // 1GB

	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	router.POST("/text-to-audio", cv.TextToAudio)
	router.GET("/text-to-audio/:id", cv.JobStatus)

	if err := router.Run(":8080"); err != nil {
		panic(err)
	}
}
