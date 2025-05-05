package main

import (
	"flag"
	"fmt"
	"os"
	"sync"
)

type AWS struct {
	s3bucket struct {
		text  string
		cvmp3 string
	}
	s3Region                 string
	s3CloudFrontDistribution string
	accessKeyId              string
	secretAccessKey          string
}

type RabbitMQ struct {
	host     string
	username string
	password string
	port     string
	exchange string
	route    struct {
		text string
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

		flag.StringVar(&instance.encryptKey, "key", os.Getenv("ENCRYPT_KEY"), "Encryption key")

		flag.StringVar(&instance.aws.s3bucket.text, "s3-text-bucket", os.Getenv("S3_TEXT_BUCKET"), "S3 text bucket name")
		flag.StringVar(&instance.aws.s3bucket.cvmp3, "s3-converted-mp3-bucket", os.Getenv("S3_CONVERTED_MP3_BUCKET"), "S3 converted mp3 bucket name")

		flag.StringVar(&instance.aws.s3Region, "s3-region", os.Getenv("S3_REGION"), "S3 region")
		flag.StringVar(&instance.aws.accessKeyId, "aws-access-key-id", os.Getenv("AWS_ACCESS_KEY_ID"), "AWS access key ID")
		flag.StringVar(&instance.aws.secretAccessKey, "aws-secret-access-key", os.Getenv("AWS_SECRET_ACCESS_KEY"), "AWS secret access key")

		flag.StringVar(&instance.rabbit.host, "rabbit-host", os.Getenv("AMQP_HOST"), "RabbitMQ host")
		flag.StringVar(&instance.rabbit.username, "rabbit-username", os.Getenv("AMQP_USERNAME"), "RabbitMQ username")
		flag.StringVar(&instance.rabbit.password, "rabbit-password", os.Getenv("AMQP_PASSWORD"), "RabbitMQ password")
		flag.StringVar(&instance.rabbit.port, "rabbit-port", os.Getenv("AMQP_PORT"), "RabbitMQ password")
		flag.StringVar(&instance.rabbit.exchange, "rabbit-exchange", os.Getenv("EXCHANGE_KEY"), "RabbitMQ exchange name")
		flag.StringVar(&instance.rabbit.route.text, "rabbit-text-route", os.Getenv("TTS_ROUTE_KEY"), "RabbitMQ text exchange route key")

		flag.StringVar(&instance.grpc.job.host, "grpc-job-host", os.Getenv("GRPC_JOB_HOST"), "Job service host")
		flag.StringVar(&instance.grpc.job.port, "grpc-job-port", os.Getenv("GRPC_JOB_PORT"), "Job service port")

		flag.Parse()
	})

	return instance
}
