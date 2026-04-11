package app

import (
	"fmt"
	"net"
	"os"
	"time"

	"appointment-service/internal/client"
	"appointment-service/internal/repository"
	grpctransport "appointment-service/internal/transport/grpc"
	httptransport "appointment-service/internal/transport/http"
	"appointment-service/internal/usecase"
	appointmentpb "appointment-service/proto"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func Run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9092"
	}

	doctorServiceAddr := os.Getenv("DOCTOR_SERVICE_ADDR")
	if doctorServiceAddr == "" {
		doctorServiceAddr = "localhost:9091"
	}

	repo := repository.NewInMemoryAppointmentRepository()
	doctorClient, err := client.NewDoctorGRPCClient(doctorServiceAddr, 2*time.Second)
	if err != nil {
		return fmt.Errorf("create doctor grpc client: %w", err)
	}
	defer doctorClient.Close()

	logger := &stdLogger{}
	uc := usecase.NewAppointmentUsecase(repo, doctorClient, logger)
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
