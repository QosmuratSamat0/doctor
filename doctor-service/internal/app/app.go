package app

import (
	"fmt"
	"os"

	"doctor-service/internal/repository"
	httptransport "doctor-service/internal/transport/http"
	"doctor-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

func Run() error {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	repo := repository.NewInMemoryDoctorRepository()
	uc := usecase.NewDoctorUsecase(repo)
	handler := httptransport.NewDoctorHandler(uc)

	router := gin.Default()
	handler.RegisterRoutes(router)

	return router.Run(fmt.Sprintf(":%s", port))
}
