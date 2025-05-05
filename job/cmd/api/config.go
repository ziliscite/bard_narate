package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
)

type AWS struct {
	dynamo struct {
		tableName string
	}
	s3Region        string
	accessKeyId     string
	secretAccessKey string
}

type RabbitMQ struct {
	host     string
	username string
	password string
	port     string
	exchange string
	route    struct {
		job string
	}
	queue struct {
		job string
	}
}

func (r RabbitMQ) dsn() string {
	return fmt.Sprintf("amqps://%s:%s@%s:%s", r.username, r.password, r.host, r.port)
}

type GRPC struct {
	job struct {
		host string
		port string
	}
}

type Config struct {
	port       int
	encryptKey string
	aws        AWS
	rabbit     RabbitMQ
	grpc       GRPC
}

var (
	instance Config
	once     sync.Once
)

func getConfig() Config {
	once.Do(func() {
		instance = Config{}

		flag.IntVar(&instance.port, "port", 8080, "Server Port")

		flag.StringVar(&instance.aws.s3Region, "s3-region", os.Getenv("S3_REGION"), "S3 region")
		flag.StringVar(&instance.aws.accessKeyId, "aws-access-key-id", os.Getenv("AWS_ACCESS_KEY_ID"), "AWS access key ID")
		flag.StringVar(&instance.aws.secretAccessKey, "aws-secret-access-key", os.Getenv("AWS_SECRET_ACCESS_KEY"), "AWS secret access key")

		flag.StringVar(&instance.rabbit.host, "rabbit-host", os.Getenv("AMQP_HOST"), "RabbitMQ host")
		flag.StringVar(&instance.rabbit.username, "rabbit-username", os.Getenv("AMQP_USERNAME"), "RabbitMQ username")
		flag.StringVar(&instance.rabbit.password, "rabbit-password", os.Getenv("AMQP_PASSWORD"), "RabbitMQ password")
		flag.StringVar(&instance.rabbit.port, "rabbit-port", os.Getenv("AMQP_PORT"), "RabbitMQ password")
		flag.StringVar(&instance.rabbit.exchange, "rabbit-exchange", os.Getenv("EXCHANGE_KEY"), "RabbitMQ exchange name")
		flag.StringVar(&instance.rabbit.route.job, "rabbit-job-route", os.Getenv("JOB_ROUTE_KEY"), "RabbitMQ text exchange route key")
		flag.StringVar(&instance.rabbit.queue.job, "rabbit-job-queue", os.Getenv("JOB_QUEUE_NAME"), "RabbitMQ text exchange queue key")

		flag.StringVar(&instance.grpc.job.host, "grpc-job-host", os.Getenv("GRPC_JOB_HOST"), "Job service host")
		flag.StringVar(&instance.grpc.job.port, "grpc-job-port", os.Getenv("GRPC_JOB_PORT"), "Job service port")

		flag.Parse()
	})

	return instance
}
