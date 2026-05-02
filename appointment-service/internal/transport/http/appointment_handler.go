package http

import (
	"errors"
	"net/http"
	"time"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"

	"github.com/gin-gonic/gin"
)

type AppointmentHandler struct {
	uc usecase.AppointmentUsecase
}

type createAppointmentRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DoctorID    string `json:"doctor_id"`
}

type updateStatusRequest struct {
	Status model.Status `json:"status"`
}

type appointmentResponse struct {
	ID          string       `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	DoctorID    string       `json:"doctor_id"`
	Status      model.Status `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

func NewAppointmentHandler(uc usecase.AppointmentUsecase) *AppointmentHandler {
	return &AppointmentHandler{uc: uc}
}

func (h *AppointmentHandler) RegisterRoutes(router *gin.Engine) {
	router.POST("/appointments", h.createAppointment)
	router.GET("/appointments/:id", h.getAppointment)
	router.GET("/appointments", h.listAppointments)
	router.PATCH("/appointments/:id/status", h.updateStatus)
}

func (h *AppointmentHandler) createAppointment(c *gin.Context) {
	var req createAppointmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	appointment, err := h.uc.Create(c.Request.Context(), usecase.CreateAppointmentInput{
		Title:       req.Title,
		Description: req.Description,
		DoctorID:    req.DoctorID,
	})
	if err != nil {
		writeAppointmentError(c, err, "failed to create appointment")
		return
	}

	c.JSON(http.StatusCreated, toAppointmentResponse(appointment))
}

func (h *AppointmentHandler) getAppointment(c *gin.Context) {
	appointment, err := h.uc.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		writeAppointmentError(c, err, "failed to fetch appointment")
		return
	}

	c.JSON(http.StatusOK, toAppointmentResponse(appointment))
}

func (h *AppointmentHandler) listAppointments(c *gin.Context) {
	appointments, err := h.uc.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list appointments"})
		return
	}

	response := make([]appointmentResponse, 0, len(appointments))
	for _, appointment := range appointments {
		response = append(response, toAppointmentResponse(appointment))
	}

	c.JSON(http.StatusOK, response)
}

func (h *AppointmentHandler) updateStatus(c *gin.Context) {
	var req updateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	appointment, err := h.uc.UpdateStatus(c.Request.Context(), c.Param("id"), req.Status)
	if err != nil {
		writeAppointmentError(c, err, "failed to update appointment status")
		return
	}

	c.JSON(http.StatusOK, toAppointmentResponse(appointment))
}

func writeAppointmentError(c *gin.Context, err error, fallback string) {
	switch {
	case errors.Is(err, usecase.ErrTitleRequired),
		errors.Is(err, usecase.ErrDoctorIDRequired),
		errors.Is(err, usecase.ErrInvalidStatus),
		errors.Is(err, usecase.ErrInvalidStatusTransition),
		errors.Is(err, usecase.ErrDoctorNotFound):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, usecase.ErrAppointmentNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, usecase.ErrDoctorServiceUnavailable):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": fallback})
	}
}

func toAppointmentResponse(appointment model.Appointment) appointmentResponse {
	return appointmentResponse{
		ID:          appointment.ID,
		Title:       appointment.Title,
		Description: appointment.Description,
		DoctorID:    appointment.DoctorID,
		Status:      appointment.Status,
		CreatedAt:   appointment.CreatedAt,
		UpdatedAt:   appointment.UpdatedAt,
	}
}
