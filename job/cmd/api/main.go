package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/ziliscite/bard_narate/job/internal/repository"
	"github.com/ziliscite/bard_narate/job/internal/service"
	pb "github.com/ziliscite/bard_narate/job/pkg/protobuf"
	"google.golang.org/grpc"
	"net"
	"time"
)

func main() {
	cfg := getConfig()

	dcl := dynamodb.NewFromConfig(aws.Config{
		Region: cfg.aws.s3Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			cfg.aws.accessKeyId,
			cfg.aws.secretAccessKey,
			"",
		),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jr := repository.NewJobRepository(dcl, cfg.aws.dynamo.tableName)
	if err := jr.AutoMigrate(ctx); err != nil {
		panic(err)
	}

	// get rabbitmq connection
	conn, err := amqp.Dial(cfg.rabbit.dsn())
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	js := service.NewJobService(jr)

	con, err := NewConsumer(conn, cfg.rabbit.exchange, cfg.rabbit.route.job, cfg.rabbit.queue.job, js)
	if err != nil {
		panic(err)
	}

	go func() {
		if err = con.consume(); err != nil {
			panic(err)
		}
	}()

	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%v", cfg.grpc.job.host, cfg.grpc.job.host))
	if err != nil {
		panic(err)
	}
	defer listen.Close()

	grp := NewGRPCServer(cfg, js)
	srv := grpc.NewServer()
	pb.RegisterJobServiceServer(srv, grp)

	if err = srv.Serve(listen); err != nil {
		panic(err)
	}
}
