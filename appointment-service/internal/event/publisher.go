package event

import (
	"context"
	"encoding/json"

	"github.com/nats-io/nats.go"
)

type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload interface{}) error
}

type NatsPublisher struct {
	nc *nats.Conn
}

func NewNatsPublisher(nc *nats.Conn) *NatsPublisher {
	return &NatsPublisher{nc: nc}
}

func (p *NatsPublisher) Publish(ctx context.Context, subject string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.nc.Publish(subject, data)
}
