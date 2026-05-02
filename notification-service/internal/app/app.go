package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"

	"notification-service/internal/logger"
	"notification-service/internal/subscriber"
	"notification-service/internal/usecase"
)

func Run() error {
	_ = godotenv.Load()

	log := logger.NewLogger()

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}

	var nc *nats.Conn
	var err error

	// Exponential backoff for NATS connection
	backoff := time.Second
	maxBackoff := 32 * time.Second
	maxRetries := 10

	for i := 0; i < maxRetries; i++ {
		nc, err = nats.Connect(natsURL)
		if err == nil {
			break
		}
		log.Infof("Failed to connect to NATS (attempt %d/%d): %v. Retrying in %v...", i+1, maxRetries, err, backoff)
		time.Sleep(backoff)
		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}

	if err != nil {
		return fmt.Errorf("could not connect to NATS after %d retries: %w", maxRetries, err)
	}
	defer nc.Close()

	log.Infof("Connected to NATS at %s", natsURL)

	// Initialize usecase
	notificationUC := usecase.NewNotificationUsecase(log)

	// Initialize and start subscriber
	notifSubscriber := subscriber.NewNotificationSubscriber(nc, notificationUC, log)

	ctx := context.Background()
	return notifSubscriber.Start(ctx)
}
