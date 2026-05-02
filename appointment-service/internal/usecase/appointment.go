package usecase

import (
	"context"
	"time"

	"appointment-service/internal/event"
	"appointment-service/internal/model"

	"github.com/google/uuid"
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

type AppointmentCreatedEvent struct {
	EventType  string       `json:"event_type"`
	OccurredAt string       `json:"occurred_at"`
	ID         string       `json:"id"`
	Title      string       `json:"title"`
	DoctorID   string       `json:"doctor_id"`
	Status     model.Status `json:"status"`
}

type AppointmentStatusUpdatedEvent struct {
	EventType  string       `json:"event_type"`
	OccurredAt string       `json:"occurred_at"`
	ID         string       `json:"id"`
	OldStatus  model.Status `json:"old_status"`
	NewStatus  model.Status `json:"new_status"`
}

type appointmentUsecase struct {
	repo          AppointmentRepository
	doctorChecker DoctorVerifier
	logger        Logger
	publisher     event.EventPublisher
}

func NewAppointmentUsecase(repo AppointmentRepository, doctorChecker DoctorVerifier, logger Logger, publisher event.EventPublisher) AppointmentUsecase {
	return &appointmentUsecase{
		repo:          repo,
		doctorChecker: doctorChecker,
		logger:        logger,
		publisher:     publisher,
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
		ID:          uuid.New().String(),
		Title:       input.Title,
		Description: input.Description,
		DoctorID:    input.DoctorID,
		Status:      model.StatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created, err := u.repo.Create(ctx, appointment)
	if err == nil && u.publisher != nil {
		_ = u.publisher.Publish(ctx, "appointments.created", AppointmentCreatedEvent{
			EventType:  "appointments.created",
			OccurredAt: time.Now().Format(time.RFC3339),
			ID:         created.ID,
			Title:      created.Title,
			DoctorID:   created.DoctorID,
			Status:     created.Status,
		})
	}

	return created, err
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

	oldStatus := appointment.Status
	appointment.Status = status
	appointment.UpdatedAt = time.Now().UTC()
	updated, err := u.repo.Update(ctx, appointment)
	if err == nil && u.publisher != nil {
		_ = u.publisher.Publish(ctx, "appointments.status_updated", AppointmentStatusUpdatedEvent{
			EventType:  "appointments.status_updated",
			OccurredAt: time.Now().Format(time.RFC3339),
			ID:         updated.ID,
			OldStatus:  oldStatus,
			NewStatus:  updated.Status,
		})
	}

	return updated, err
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
