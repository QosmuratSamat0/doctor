package http

import (
	"errors"
	"net/http"

	"doctor-service/internal/model"
	"doctor-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type DoctorHandler struct {
	uc usecase.DoctorUsecase
}

type createDoctorRequest struct {
	FullName       string `json:"full_name"`
	Specialization string `json:"specialization"`
	Email          string `json:"email"`
}

type doctorResponse struct {
	ID             string `json:"id"`
	FullName       string `json:"full_name"`
	Specialization string `json:"specialization"`
	Email          string `json:"email"`
}

func NewDoctorHandler(uc usecase.DoctorUsecase) *DoctorHandler {
	return &DoctorHandler{uc: uc}
}

func (h *DoctorHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/doctors", h.createDoctor)
	router.GET("/doctors/:id", h.getDoctor)
	router.GET("/doctors", h.listDoctors)
}

func (h *DoctorHandler) createDoctor(c *gin.Context) {
	var req createDoctorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	doctor, err := h.uc.Create(c.Request.Context(), usecase.CreateDoctorInput{
		FullName:       req.FullName,
		Specialization: req.Specialization,
		Email:          req.Email,
	})
	if err != nil {
		switch {
		case errors.Is(err, usecase.ErrFullNameRequired), errors.Is(err, usecase.ErrEmailRequired):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		case errors.Is(err, usecase.ErrDoctorEmailExists):
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create doctor"})
		}
		return
	}

	c.JSON(http.StatusCreated, toDoctorResponse(doctor))
}

func (h *DoctorHandler) getDoctor(c *gin.Context) {
	doctor, err := h.uc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		if errors.Is(err, usecase.ErrDoctorNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch doctor"})
		return
	}

	c.JSON(http.StatusOK, toDoctorResponse(doctor))
}

func (h *DoctorHandler) listDoctors(c *gin.Context) {
	doctors, err := h.uc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list doctors"})
		return
	}

	response := make([]doctorResponse, 0, len(doctors))
	for _, doctor := range doctors {
		response = append(response, toDoctorResponse(doctor))
	}

	c.JSON(http.StatusOK, response)
}

func toDoctorResponse(doctor model.Doctor) doctorResponse {
	return doctorResponse{
		ID:             doctor.ID,
		FullName:       doctor.FullName,
		Specialization: doctor.Specialization,
		Email:          doctor.Email,
	}
}
