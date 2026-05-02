package usecase

import (
	"context"
	"time"

	"doctor-service/internal/event"
	"doctor-service/internal/model"

	"github.com/google/uuid"
)

type DoctorRepository interface {
	Create(ctx context.Context, doctor model.Doctor) (model.Doctor, error)
	GetByID(ctx context.Context, id string) (model.Doctor, error)
	GetByEmail(ctx context.Context, email string) (model.Doctor, error)
	List(ctx context.Context) ([]model.Doctor, error)
}

type DoctorUsecase interface {
	Create(ctx context.Context, input CreateDoctorInput) (model.Doctor, error)
	GetByID(ctx context.Context, id string) (model.Doctor, error)
	List(ctx context.Context) ([]model.Doctor, error)
}

type CreateDoctorInput struct {
	FullName       string
	Specialization string
	Email          string
}

type DoctorCreatedEvent struct {
	EventType      string `json:"event_type"`
	OccurredAt     string `json:"occurred_at"`
	ID             string `json:"id"`
	FullName       string `json:"full_name"`
	Specialization string `json:"specialization"`
	Email          string `json:"email"`
}

type doctorUsecase struct {
	repo      DoctorRepository
	publisher event.EventPublisher
}

func NewDoctorUsecase(repo DoctorRepository, publisher event.EventPublisher) DoctorUsecase {
	return &doctorUsecase{repo: repo, publisher: publisher}
}

func (u *doctorUsecase) Create(ctx context.Context, input CreateDoctorInput) (model.Doctor, error) {
	if input.FullName == "" {
		return model.Doctor{}, ErrFullNameRequired
	}
	if input.Email == "" {
		return model.Doctor{}, ErrEmailRequired
	}

	if _, err := u.repo.GetByEmail(ctx, input.Email); err == nil {
		return model.Doctor{}, ErrDoctorEmailExists
	} else if err != ErrDoctorNotFound {
		return model.Doctor{}, err
	}

	doctor := model.Doctor{
		ID:             uuid.New().String(),
		FullName:       input.FullName,
		Specialization: input.Specialization,
		Email:          input.Email,
	}

	created, err := u.repo.Create(ctx, doctor)
	if err == nil && u.publisher != nil {
		_ = u.publisher.Publish(ctx, "doctors.created", DoctorCreatedEvent{
			EventType:      "doctors.created",
			OccurredAt:     time.Now().Format(time.RFC3339),
			ID:             created.ID,
			FullName:       created.FullName,
			Specialization: created.Specialization,
			Email:          created.Email,
		})
	}

	return created, err
}

func (u *doctorUsecase) GetByID(ctx context.Context, id string) (model.Doctor, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *doctorUsecase) List(ctx context.Context) ([]model.Doctor, error) {
	return u.repo.List(ctx)
}
