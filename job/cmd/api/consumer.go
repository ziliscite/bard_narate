package main

import (
	"context"
	"encoding/json"
	"errors"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/ziliscite/bard_narate/job/internal/domain"
	"github.com/ziliscite/bard_narate/job/internal/service"
	"time"
)

type mq struct {
	q   string
	con *amqp.Connection
}

type Consumer struct {
	mq mq
	js service.JobService
}

func NewConsumer(con *amqp.Connection, exchange, route, queue string, js service.JobService) (*Consumer, error) {
	ch, err := con.Channel()
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	// Declare an exchange
	if err = ch.ExchangeDeclare(exchange, "topic", true, false, false, false, nil); err != nil {
		return nil, err
	}

	// declare a queue
	aq, err := ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return nil, err
	}

	// bind consumer queue to the exchange
	// topic route should be like "file.text" or in this case "file.#" to listen to all file types
	if err = ch.QueueBind(aq.Name, route, exchange, false, nil); err != nil {
		return nil, err
	}

	return &Consumer{
		mq: mq{
			q:   queue,
			con: con,
		},
		js: js,
	}, nil
}

func (c *Consumer) consume() error {
	ch, err := c.mq.con.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	videos, err := ch.Consume(c.mq.q, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	forever := make(chan bool)
	go func() {
		for v := range videos {
			if err = c.consumeJob(ctx, v.Body); err != nil {
				v.Nack(false, false)
				continue
			}

			v.Ack(false)
		}
	}()

	<-forever
	return nil
}

func (c *Consumer) consumeJob(ctx context.Context, msg []byte) error {
	var req struct {
		JobId     string `json:"job_id"`
		JobStatus string `json:"job_status"`
		// no need filekey
	}

	if err := json.Unmarshal(msg, &req); err != nil {
		return err
	}

	if req.JobId == "" || req.JobStatus == "" {
		return errors.New("job_id and job_status are required")
	}

	var status domain.JobStatus

	switch req.JobStatus {
	case "Pending":
		status = domain.Pending
	case "Processing":
		status = domain.Processing
	case "Converting":
		status = domain.Converting
	case "Completed":
		status = domain.Completed
	case "Failed":
		status = domain.Failed
	default:
		return errors.New("invalid job status")
	}

	if err := c.js.UpdateStatus(ctx, req.JobId, status); err != nil {
		return err
	}

	return nil
}
