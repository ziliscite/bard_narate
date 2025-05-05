package midtrans

import (
	"context"
	"resty.dev/v3"
	"time"
)

type Client struct {
	r  *resty.Client
	sk string
}

func New(serverKey, baseUrl string) *Client {
	client := resty.New().
		SetTimeout(5*time.Second).
		SetBasicAuth(serverKey, "").
		//SetAuthScheme("Basic").SetAuthToken(serverKey).
		SetBaseURL(baseUrl)

	return &Client{
		r:  client,
		sk: serverKey,
	}
}

func (c *Client) requestJSON(ctx context.Context) *resty.Request {
	return c.r.R().SetContext(ctx).SetHeader("Content-Type", "application/json").SetHeader("Accept", "application/json")
}
