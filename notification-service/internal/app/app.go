package app

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"strconv"

	"notification-service/internal/jobqueue"
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

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	var rdb *redis.Client
	opt, redisErr := redis.ParseURL(redisURL)
	if redisErr != nil {
		log.Errorf("warning: could not parse Redis URL %s: %v", redisURL, redisErr)
	} else {
		rdb = redis.NewClient(opt)
		if err := rdb.Ping(context.Background()).Err(); err != nil {
			log.Errorf("warning: could not connect to Redis at %s: %v", redisURL, err)
			rdb = nil
		}
	}

	gatewayURL := os.Getenv("GATEWAY_URL")
	if gatewayURL == "" {
		gatewayURL = "http://localhost:8080"
	}

	poolSizeStr := os.Getenv("WORKER_POOL_SIZE")
	poolSize, _ := strconv.Atoi(poolSizeStr)
	if poolSize <= 0 {
		poolSize = 3
	}

	jobQueue := jobqueue.NewJobQueue(rdb, gatewayURL, poolSize)
	defer jobQueue.Stop()

	// Initialize usecase
	notificationUC := usecase.NewNotificationUsecase(log, jobQueue)

	// Initialize and start subscriber
	notifSubscriber := subscriber.NewNotificationSubscriber(nc, notificationUC, log)

	ctx := context.Background()
	return notifSubscriber.Start(ctx)
}
