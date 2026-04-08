package app

import (
	"fmt"
	"os"
	"time"

	"appointment-service/internal/client"
	"appointment-service/internal/repository"
	httptransport "appointment-service/internal/transport/http"
	"appointment-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	doctorServiceURL := os.Getenv("DOCTOR_SERVICE_URL")
	if doctorServiceURL == "" {
		doctorServiceURL = "http://localhost:8081"
	}

	repo := repository.NewInMemoryAppointmentRepository()
	doctorClient := client.NewDoctorHTTPClient(doctorServiceURL, 2*time.Second)
	logger := &stdLogger{}
	uc := usecase.NewAppointmentUsecase(repo, doctorClient, logger)
	handler := httptransport.NewAppointmentHandler(uc)

	router := gin.Default()
	handler.RegisterRoutes(router)

	return router.Run(fmt.Sprintf(":%s", port))
}
