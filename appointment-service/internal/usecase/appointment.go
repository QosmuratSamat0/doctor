package usecase

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"appointment-service/internal/model"
)

type AppointmentRepository interface {
	Create(ctx context.Context, appointment model.Appointment) (model.Appointment, error)
	GetByID(ctx context.Context, id string) (model.Appointment, error)
	List(ctx context.Context) ([]model.Appointment, error)
	Update(ctx context.Context, appointment model.Appointment) (model.Appointment, error)
}

type DoctorVerifier interface {
	VerifyDoctorExists(ctx context.Context, doctorID string) error
}

type Logger interface {
	Errorf(format string, args ...any)
}

type AppointmentUsecase interface {
	Create(ctx context.Context, input CreateAppointmentInput) (model.Appointment, error)
	GetByID(ctx context.Context, id string) (model.Appointment, error)
	List(ctx context.Context) ([]model.Appointment, error)
	UpdateStatus(ctx context.Context, id string, status model.Status) (model.Appointment, error)
}

type CreateAppointmentInput struct {
	Title       string
	Description string
	DoctorID    string
}

type appointmentUsecase struct {
	repo          AppointmentRepository
	doctorChecker DoctorVerifier
	logger        Logger
	idCounter     atomic.Uint64
}

func NewAppointmentUsecase(repo AppointmentRepository, doctorChecker DoctorVerifier, logger Logger) AppointmentUsecase {
	return &appointmentUsecase{
		repo:          repo,
		doctorChecker: doctorChecker,
		logger:        logger,
	}
}

func (u *appointmentUsecase) Create(ctx context.Context, input CreateAppointmentInput) (model.Appointment, error) {
	if input.Title == "" {
		return model.Appointment{}, ErrTitleRequired
	}
	if input.DoctorID == "" {
		return model.Appointment{}, ErrDoctorIDRequired
	}

	if err := u.verifyDoctor(ctx, input.DoctorID, "create"); err != nil {
		return model.Appointment{}, err
	}

	now := time.Now().UTC()
	appointment := model.Appointment{
		ID:          fmt.Sprintf("appointment-%d", u.idCounter.Add(1)),
		Title:       input.Title,
		Description: input.Description,
		DoctorID:    input.DoctorID,
		Status:      model.StatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return u.repo.Create(ctx, appointment)
}

func (u *appointmentUsecase) GetByID(ctx context.Context, id string) (model.Appointment, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *appointmentUsecase) List(ctx context.Context) ([]model.Appointment, error) {
	return u.repo.List(ctx)
}

func (u *appointmentUsecase) UpdateStatus(ctx context.Context, id string, status model.Status) (model.Appointment, error) {
	if !isValidStatus(status) {
		return model.Appointment{}, ErrInvalidStatus
	}

	appointment, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return model.Appointment{}, err
	}

	if appointment.Status == model.StatusDone && status == model.StatusNew {
		return model.Appointment{}, ErrInvalidStatusTransition
	}

	if err := u.verifyDoctor(ctx, appointment.DoctorID, "update"); err != nil {
		return model.Appointment{}, err
	}

	appointment.Status = status
	appointment.UpdatedAt = time.Now().UTC()
	return u.repo.Update(ctx, appointment)
}

func (u *appointmentUsecase) verifyDoctor(ctx context.Context, doctorID string, action string) error {
	if err := u.doctorChecker.VerifyDoctorExists(ctx, doctorID); err != nil {
		if err == ErrDoctorNotFound {
			return err
		}
		u.logger.Errorf("doctor verification failed during %s for doctor_id=%s: %v", action, doctorID, err)
		return ErrDoctorServiceUnavailable
	}
	return nil
}

func isValidStatus(status model.Status) bool {
	switch status {
	case model.StatusNew, model.StatusInProgress, model.StatusDone:
		return true
	default:
		return false
	}
}
