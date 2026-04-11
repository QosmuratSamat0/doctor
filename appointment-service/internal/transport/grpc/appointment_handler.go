package grpc

import (
	"context"
	"errors"
	"time"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"
	appointmentpb "appointment-service/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AppointmentHandler struct {
	appointmentpb.UnimplementedAppointmentServiceServer
	uc usecase.AppointmentUsecase
}

func NewAppointmentHandler(uc usecase.AppointmentUsecase) *AppointmentHandler {
	return &AppointmentHandler{uc: uc}
}

func (h *AppointmentHandler) CreateAppointment(ctx context.Context, req *appointmentpb.CreateAppointmentRequest) (*appointmentpb.AppointmentResponse, error) {
	appointment, err := h.uc.Create(ctx, usecase.CreateAppointmentInput{
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		DoctorID:    req.GetDoctorId(),
	})
	if err != nil {
		return nil, mapAppointmentError(err, "failed to create appointment")
	}

	return toAppointmentResponse(appointment), nil
}

func (h *AppointmentHandler) GetAppointment(ctx context.Context, req *appointmentpb.GetAppointmentRequest) (*appointmentpb.AppointmentResponse, error) {
	appointment, err := h.uc.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, mapAppointmentError(err, "failed to fetch appointment")
	}

	return toAppointmentResponse(appointment), nil
}

func (h *AppointmentHandler) ListAppointments(ctx context.Context, _ *appointmentpb.ListAppointmentsRequest) (*appointmentpb.ListAppointmentsResponse, error) {
	appointments, err := h.uc.List(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list appointments")
	}

	response := &appointmentpb.ListAppointmentsResponse{
		Appointments: make([]*appointmentpb.AppointmentResponse, 0, len(appointments)),
	}
	for _, appointment := range appointments {
		response.Appointments = append(response.Appointments, toAppointmentResponse(appointment))
	}

	return response, nil
}

func (h *AppointmentHandler) UpdateAppointmentStatus(ctx context.Context, req *appointmentpb.UpdateStatusRequest) (*appointmentpb.AppointmentResponse, error) {
	appointment, err := h.uc.UpdateStatus(ctx, req.GetId(), model.Status(req.GetStatus()))
	if err != nil {
		return nil, mapAppointmentError(err, "failed to update appointment status")
	}

	return toAppointmentResponse(appointment), nil
}

func mapAppointmentError(err error, fallbackMessage string) error {
	switch {
	case errors.Is(err, usecase.ErrTitleRequired),
		errors.Is(err, usecase.ErrDoctorIDRequired),
		errors.Is(err, usecase.ErrInvalidStatus),
		errors.Is(err, usecase.ErrInvalidStatusTransition):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, usecase.ErrDoctorNotFound):
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, usecase.ErrAppointmentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, usecase.ErrDoctorServiceUnavailable):
		return status.Error(codes.Unavailable, err.Error())
	default:
		return status.Error(codes.Internal, fallbackMessage)
	}
}

func toAppointmentResponse(appointment model.Appointment) *appointmentpb.AppointmentResponse {
	return &appointmentpb.AppointmentResponse{
		Id:          appointment.ID,
		Title:       appointment.Title,
		Description: appointment.Description,
		DoctorId:    appointment.DoctorID,
		Status:      string(appointment.Status),
		CreatedAt:   appointment.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:   appointment.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}
