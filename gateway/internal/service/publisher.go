package service

import (
	"context"
	"encoding/json"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher interface {
	PublishConversion(ctx context.Context, jobId, fileKey string) error
}

type routeKey struct {
	text string
}

type publisher struct {
	exchange string
	con      *amqp.Connection
	rk       routeKey
}

func NewPublisher(con *amqp.Connection, exchangeName, textRouteKey string) (Publisher, error) {
	ch, err := con.Channel()
	if err != nil {
		return nil, err
	}
	defer ch.Close()

	if err = ch.ExchangeDeclare(exchangeName, "topic", true, false, false, false, nil); err != nil {
		return nil, err
	}

	return &publisher{
		exchange: exchangeName,
		con:      con,
		rk: routeKey{
			text: textRouteKey, // "file.text"
		},
	}, nil
}

func (p *publisher) PublishConversion(ctx context.Context, jobId, fileKey string) error {
	req := struct {
		JobId     string `json:"job_id"`
		JobStatus string `json:"job_status"`
		FileKey   string `json:"file_key"`
	}{
		JobId:     jobId,
		JobStatus: "Processing",
		FileKey:   fileKey,
	}

	msg, err := json.Marshal(req)
	if err != nil {
		return err
	}

	ch, err := p.con.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	return ch.PublishWithContext(ctx,
		p.exchange,
		p.rk.text,
		true,
		false,
		amqp.Publishing{
			// UserId:
			DeliveryMode: amqp.Persistent,
			ContentType:  "application/json",
			Body:         msg,
		},
	)
}
