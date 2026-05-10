package usecase

import (
	"context"
	"time"

	"appointment-service/internal/cache"
	"appointment-service/internal/event"
	"appointment-service/internal/model"
	"log"

	"github.com/google/uuid"
)

type AppointmentRepository interface {
	Create(ctx context.Context, appointment model.Appointment) (model.Appointment, error)
	GetByID(ctx context.Context, id uuid.UUID) (model.Appointment, error)
	List(ctx context.Context) ([]model.Appointment, error)
	Update(ctx context.Context, appointment model.Appointment) (model.Appointment, error)
	WithTx(ctx context.Context, fn func(AppointmentRepository) (model.Appointment, error)) (model.Appointment, error)
}

type DoctorVerifier interface {
	VerifyDoctorExists(ctx context.Context, doctorID string) error
}

type Logger interface {
	Errorf(format string, args ...any)
}

type AppointmentUsecase interface {
	Create(ctx context.Context, input CreateAppointmentInput) (model.Appointment, error)
	GetByID(ctx context.Context, id uuid.UUID) (model.Appointment, error)
	List(ctx context.Context) ([]model.Appointment, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status model.Status) (model.Appointment, error)
}

type CreateAppointmentInput struct {
	Title       string
	Description string
	DoctorID    uuid.UUID
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
	DoctorID   string       `json:"doctor_id"`
	OldStatus  model.Status `json:"old_status"`
	NewStatus  model.Status `json:"new_status"`
}

type appointmentUsecase struct {
	repo          AppointmentRepository
	doctorChecker DoctorVerifier
	logger        Logger
	publisher     event.EventPublisher
	cache         cache.CacheRepository
}

func NewAppointmentUsecase(repo AppointmentRepository, doctorChecker DoctorVerifier, logger Logger, publisher event.EventPublisher, cache cache.CacheRepository) AppointmentUsecase {
	return &appointmentUsecase{
		repo:          repo,
		doctorChecker: doctorChecker,
		logger:        logger,
		publisher:     publisher,
		cache:         cache,
	}
}

func (u *appointmentUsecase) Create(ctx context.Context, input CreateAppointmentInput) (model.Appointment, error) {
	if input.Title == "" {
		return model.Appointment{}, ErrTitleRequired
	}
	if input.DoctorID == uuid.Nil {
		return model.Appointment{}, ErrDoctorIDRequired
	}

	if err := u.verifyDoctor(ctx, input.DoctorID.String(), "create"); err != nil {
		return model.Appointment{}, err
	}

	now := time.Now().UTC()
	appointment := model.Appointment{
		ID:          uuid.New(),
		Title:       input.Title,
		Description: input.Description,
		DoctorID:    input.DoctorID,
		Status:      model.StatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created, err := u.repo.WithTx(ctx, func(repo AppointmentRepository) (model.Appointment, error) {
		return repo.Create(ctx, appointment)
	})
	if err == nil {
		if u.publisher != nil {
			_ = u.publisher.Publish(ctx, "appointments.created", AppointmentCreatedEvent{
				EventType:  "appointments.created",
				OccurredAt: time.Now().Format(time.RFC3339),
				ID:         created.ID.String(),
				Title:      created.Title,
				DoctorID:   created.DoctorID.String(),
				Status:     created.Status,
			})
		}
		// Write-Around: invalidate list key
		if u.cache != nil {
			if err := u.cache.InvalidateAppointmentsList(ctx); err != nil {
				log.Printf("Cache invalidation error: %v", err)
			}
		}
	}

	return created, err
}

func (u *appointmentUsecase) GetByID(ctx context.Context, id uuid.UUID) (model.Appointment, error) {
	if u.cache != nil {
		if cached, err := u.cache.GetAppointment(ctx, id.String()); err == nil && cached != nil {
			return *cached, nil
		} else if err != nil {
			log.Printf("Cache get error: %v", err)
		}
	}

	appt, err := u.repo.GetByID(ctx, id)
	if err == nil && u.cache != nil {
		if err := u.cache.SetAppointment(ctx, &appt); err != nil {
			log.Printf("Cache set error: %v", err)
		}
	}
	return appt, err
}

func (u *appointmentUsecase) List(ctx context.Context) ([]model.Appointment, error) {
	if u.cache != nil {
		if cached, err := u.cache.GetAppointmentsList(ctx); err == nil && cached != nil {
			appts := make([]model.Appointment, len(cached))
			for i, v := range cached {
				appts[i] = *v
			}
			return appts, nil
		} else if err != nil {
			log.Printf("Cache list error: %v", err)
		}
	}

	appts, err := u.repo.List(ctx)
	if err == nil && u.cache != nil {
		ptrAppts := make([]*model.Appointment, len(appts))
		for i := range appts {
			ptrAppts[i] = &appts[i]
		}
		if err := u.cache.SetAppointmentsList(ctx, ptrAppts); err != nil {
			log.Printf("Cache list set error: %v", err)
		}
	}
	return appts, err
}

func (u *appointmentUsecase) UpdateStatus(ctx context.Context, id uuid.UUID, status model.Status) (model.Appointment, error) {
	if !isValidStatus(status) {
		return model.Appointment{}, ErrInvalidStatus
	}

	var oldStatus model.Status
	updated, err := u.repo.WithTx(ctx, func(repo AppointmentRepository) (model.Appointment, error) {
		appointment, err := repo.GetByID(ctx, id)
		if err != nil {
			return model.Appointment{}, err
		}

		if appointment.Status == model.StatusDone && status == model.StatusNew {
			return model.Appointment{}, ErrInvalidStatusTransition
		}

		if err := u.verifyDoctor(ctx, appointment.DoctorID.String(), "update"); err != nil {
			return model.Appointment{}, err
		}

		oldStatus = appointment.Status
		appointment.Status = status
		appointment.UpdatedAt = time.Now().UTC()
		return repo.Update(ctx, appointment)
	})
	if err == nil {
		if u.publisher != nil {
			_ = u.publisher.Publish(ctx, "appointments.status_updated", AppointmentStatusUpdatedEvent{
				EventType:  "appointments.status_updated",
				OccurredAt: time.Now().Format(time.RFC3339),
				ID:         updated.ID.String(),
				DoctorID:   updated.DoctorID.String(),
				OldStatus:  oldStatus,
				NewStatus:  updated.Status,
			})
		}
		// Write-Through: update individual key and invalidate list key
		if u.cache != nil {
			if err := u.cache.SetAppointment(ctx, &updated); err != nil {
				log.Printf("Cache update error: %v", err)
			}
			if err := u.cache.InvalidateAppointmentsList(ctx); err != nil {
				log.Printf("Cache invalidation error: %v", err)
			}
		}
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
