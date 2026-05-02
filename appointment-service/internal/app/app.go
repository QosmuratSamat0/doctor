package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"appointment-service/internal/client"
	"appointment-service/internal/event"
	"appointment-service/internal/repository/postgres"
	grpctransport "appointment-service/internal/transport/grpc"
	httptransport "appointment-service/internal/transport/http"
	"appointment-service/internal/usecase"
	appointmentpb "appointment-service/proto"

	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
)

func Run() error {
	_ = godotenv.Load()
	ctx := context.Background()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9092"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/appointments?sslmode=disable"
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

	var publisher event.EventPublisher
	if nc != nil {
		publisher = event.NewNatsPublisher(nc)
	}

	repo := postgres.NewPostgresRepo(pool)

	doctorServiceAddr := os.Getenv("DOCTOR_SERVICE_ADDR")
	if doctorServiceAddr == "" {
		doctorServiceAddr = "localhost:9091"
	}

	doctorClient, err := client.NewDoctorGRPCClient(doctorServiceAddr, 2*time.Second)
	if err != nil {
		return fmt.Errorf("create doctor grpc client: %w", err)
	}
	defer doctorClient.Close()

	logger := &stdLogger{}
	uc := usecase.NewAppointmentUsecase(repo, doctorClient, logger, publisher)
	httpHandler := httptransport.NewAppointmentHandler(uc)
	grpcHandler := grpctransport.NewAppointmentHandler(uc)

	router := gin.Default()
	httpHandler.RegisterRoutes(router)

	grpcServer := grpc.NewServer()
	appointmentpb.RegisterAppointmentServiceServer(grpcServer, grpcHandler)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", grpcPort))
	if err != nil {
		return fmt.Errorf("listen grpc: %w", err)
	}

	errCh := make(chan error, 2)

	go func() {
		errCh <- router.Run(fmt.Sprintf(":%s", port))
	}()

	go func() {
		errCh <- grpcServer.Serve(listener)
	}()

	return <-errCh
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
