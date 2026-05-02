package model

import (
	"context"
	"time"
)

type Notification struct {
	Title     string
	Message   string
	Timestamp time.Time
}

// Ports
type EventSubscriber interface {
    Subscribe(subject string, handler func(Event)) error
}

type NotificationSender interface {
    Send(ctx context.Context, n Notification) error
}