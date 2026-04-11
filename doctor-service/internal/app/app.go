package app

import (
	"fmt"
	"net"
	"os"

	"doctor-service/internal/repository"
	grpctransport "doctor-service/internal/transport/grpc"
	httptransport "doctor-service/internal/transport/http"
	"doctor-service/internal/usecase"
	doctorpb "doctor-service/proto"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
)

func Run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "9091"
	}

	repo := repository.NewInMemoryDoctorRepository()
	uc := usecase.NewDoctorUsecase(repo)
	httpHandler := httptransport.NewDoctorHandler(uc)
	grpcHandler := grpctransport.NewDoctorHandler(uc)

	router := gin.Default()
	httpHandler.RegisterRoutes(router)

	grpcServer := grpc.NewServer()
	doctorpb.RegisterDoctorServiceServer(grpcServer, grpcHandler)

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
