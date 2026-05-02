package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"notification-service/internal/logger"
	"notification-service/internal/model"
	"notification-service/internal/usecase"
)

type NotificationSubscriber interface {
	Start(ctx context.Context) error
}

type natsSubscriber struct {
	nc  *nats.Conn
	uc  usecase.NotificationUsecase
	log logger.Logger
}

func NewNotificationSubscriber(nc *nats.Conn, uc usecase.NotificationUsecase, log logger.Logger) NotificationSubscriber {
	return &natsSubscriber{
		nc:  nc,
		uc:  uc,
		log: log,
	}
}

func (s *natsSubscriber) Start(ctx context.Context) error {
	handler := func(m *nats.Msg) {
		var payload interface{}
		if err := json.Unmarshal(m.Data, &payload); err != nil {
			s.log.Errorf("Error deserializing event: %v", err)
			return
		}

		timestamp := fmt.Sprintf("%v", time.Now())
		if meta, err := m.Metadata(); err == nil {
			timestamp = fmt.Sprintf("%v", meta.Timestamp)
		}

		event := model.Event{
			Time:    timestamp,
			Subject: m.Subject,
			Event:   payload,
		}

		if err := s.uc.HandleEvent(ctx, event); err != nil {
			s.log.Errorf("Error handling event: %v", err)
		}
	}

	subjects := []string{"doctors.created", "appointments.created", "appointments.status_updated"}
	for _, subj := range subjects {
		_, err := s.nc.Subscribe(subj, handler)
		if err != nil {
			return fmt.Errorf("error subscribing to %s: %w", subj, err)
		}
	}

	s.log.Infof("Subscribed to subjects: %v", subjects)

	// Wait for termination signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	s.log.Infof("Notification service is running. Press Ctrl+C to exit.")
	<-sigCh

	s.log.Infof("Shutting down notification service...")
	return nil
}
