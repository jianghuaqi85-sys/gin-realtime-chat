package grpcclient

import (
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type Client struct {
	conn *grpc.ClientConn
}

func NewClient(address string) (*Client, error) {
	retryPolicy := `{
		"loadBalancingPolicy": "round_robin",
		"methodConfig": [{
			"name": [{"service": ""}],
			"retryPolicy": {
				"maxAttempts": 3,
				"initialBackoff": "0.1s",
				"maxBackoff": "1s",
				"backoffMultiplier": 2.0,
				"retryableStatusCodes": ["UNAVAILABLE"]
			}
		}]
	}`

	conn, err := grpc.NewClient(address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithConnectParams(grpc.ConnectParams{
			Backoff: backoff.Config{
				BaseDelay:  1 * time.Second,
				Multiplier: 1.6,
				Jitter:     0.2,
				MaxDelay:   120 * time.Second,
			},
		}),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithDefaultServiceConfig(retryPolicy),
	)
	if err != nil {
		return nil, err
	}

	return &Client{conn: conn}, nil
}

func (c *Client) Conn() grpc.ClientConnInterface {
	if c == nil {
		return nil
	}
	return c.conn
}

func (c *Client) Close() error {
	if c != nil && c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
