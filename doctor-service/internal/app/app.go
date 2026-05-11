package app

import (
	"context"
	"fmt"
	"net"
	"os"

	"doctor-service/internal/cache"
	"doctor-service/internal/event"
	"doctor-service/internal/middleware"
	"doctor-service/internal/repository/postgres"
	grpctransport "doctor-service/internal/transport/grpc"
	"doctor-service/internal/usecase"
	doctorpb "doctor-service/proto"
	"strconv"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
)

func Run() error {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf("Info: godotenv.Load() skipped or failed: %v (using system environment variables)\n", err)
	} else {
		fmt.Println("Successfully loaded .env file")
	}
	ctx := context.Background()

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9091"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/doctors?sslmode=disable"
	}

	if err := runMigrations(dbURL, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect to postgres: %w", err)
	}
	defer pool.Close()

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = "nats://localhost:4222"
	}
	nc, err := nats.Connect(natsURL)
	if err != nil {
		fmt.Printf("warning: could not connect to NATS at %s: %v\n", natsURL, err)
	} else {
		defer nc.Close()
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	var rdb *redis.Client
	var cacheRepo cache.CacheRepository
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		fmt.Printf("warning: could not parse Redis URL %s: %v\n", redisURL, err)
	} else {
		rdb = redis.NewClient(opt)
		if err := rdb.Ping(ctx).Err(); err != nil {
			fmt.Printf("warning: could not connect to Redis at %s: %v\n", redisURL, err)
			rdb = nil
		} else {
			ttlStr := os.Getenv("CACHE_TTL_SECONDS")
			ttlSec, _ := strconv.Atoi(ttlStr)
			if ttlSec <= 0 {
				ttlSec = 60
			}
			cacheRepo = cache.NewRedisCacheRepository(rdb, time.Duration(ttlSec)*time.Second)
		}
	}

	var publisher event.EventPublisher
	if nc != nil {
		publisher = event.NewNatsPublisher(nc)
	}

	repo := postgres.NewPostgresRepo(pool)
	uc := usecase.NewDoctorUsecase(repo, publisher, cacheRepo)
	grpcHandler := grpctransport.NewDoctorHandler(uc)

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(middleware.RateLimitInterceptor(rdb)),
	)
	doctorpb.RegisterDoctorServiceServer(grpcServer, grpcHandler)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	return grpcServer.Serve(listener)
}

func runMigrations(dbURL string, migrationsPath string) error {
	m, err := migrate.New("file://"+migrationsPath, dbURL)
	if err != nil {
		return fmt.Errorf("create migrate instance: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
