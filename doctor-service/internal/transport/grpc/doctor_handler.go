package grpc

import (
	"context"
	"errors"

	"doctor-service/internal/model"
	"doctor-service/internal/usecase"
	doctorpb "doctor-service/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type DoctorHandler struct {
	doctorpb.UnimplementedDoctorServiceServer
	uc usecase.DoctorUsecase
}

func NewDoctorHandler(uc usecase.DoctorUsecase) *DoctorHandler {
	return &DoctorHandler{uc: uc}
}

func (h *DoctorHandler) CreateDoctor(ctx context.Context, req *doctorpb.CreateDoctorRequest) (*doctorpb.DoctorResponse, error) {
	doctor, err := h.uc.Create(ctx, usecase.CreateDoctorInput{
		FullName:       req.GetFullName(),
		Specialization: req.GetSpecialization(),
		Email:          req.GetEmail(),
	})
	if err != nil {
		return nil, mapDoctorError(err, codes.Internal, "failed to create doctor")
	}

	return toDoctorResponse(doctor), nil
}

func (h *DoctorHandler) GetDoctor(ctx context.Context, req *doctorpb.GetDoctorRequest) (*doctorpb.DoctorResponse, error) {
	doctor, err := h.uc.GetByID(ctx, req.GetId())
	if err != nil {
		return nil, mapDoctorError(err, codes.Internal, "failed to fetch doctor")
	}

	return toDoctorResponse(doctor), nil
}

func (h *DoctorHandler) ListDoctors(ctx context.Context, _ *doctorpb.ListDoctorsRequest) (*doctorpb.ListDoctorsResponse, error) {
	doctors, err := h.uc.List(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list doctors")
	}

	response := &doctorpb.ListDoctorsResponse{
		Doctors: make([]*doctorpb.DoctorResponse, 0, len(doctors)),
	}
	for _, doctor := range doctors {
		response.Doctors = append(response.Doctors, toDoctorResponse(doctor))
	}

	return response, nil
}

func mapDoctorError(err error, fallbackCode codes.Code, fallbackMessage string) error {
	switch {
	case errors.Is(err, usecase.ErrFullNameRequired), errors.Is(err, usecase.ErrEmailRequired):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, usecase.ErrDoctorEmailExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, usecase.ErrDoctorNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(fallbackCode, fallbackMessage)
	}
}

func toDoctorResponse(doctor model.Doctor) *doctorpb.DoctorResponse {
	return &doctorpb.DoctorResponse{
		Id:             doctor.ID,
		FullName:       doctor.FullName,
		Specialization: doctor.Specialization,
		Email:          doctor.Email,
	}
}
