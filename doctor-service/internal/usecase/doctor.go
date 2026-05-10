package usecase

import (
	"context"
	"time"

	"doctor-service/internal/cache"
	"doctor-service/internal/event"
	"doctor-service/internal/model"
	"log"

	"github.com/google/uuid"
)

type DoctorRepository interface {
	Create(ctx context.Context, doctor model.Doctor) (model.Doctor, error)
	GetByID(ctx context.Context, id uuid.UUID) (model.Doctor, error)
	GetByEmail(ctx context.Context, email string) (model.Doctor, error)
	List(ctx context.Context) ([]model.Doctor, error)
	WithTx(ctx context.Context, fn func(DoctorRepository) (model.Doctor, error)) (model.Doctor, error)
}

type DoctorUsecase interface {
	Create(ctx context.Context, input CreateDoctorInput) (model.Doctor, error)
	GetByID(ctx context.Context, id uuid.UUID) (model.Doctor, error)
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
	cache     cache.CacheRepository
}

func NewDoctorUsecase(repo DoctorRepository, publisher event.EventPublisher, cache cache.CacheRepository) DoctorUsecase {
	return &doctorUsecase{repo: repo, publisher: publisher, cache: cache}
}

func (u *doctorUsecase) Create(ctx context.Context, input CreateDoctorInput) (model.Doctor, error) {
	if input.FullName == "" {
		return model.Doctor{}, ErrFullNameRequired
	}
	if input.Email == "" {
		return model.Doctor{}, ErrEmailRequired
	}

	created, err := u.repo.WithTx(ctx, func(repo DoctorRepository) (model.Doctor, error) {
		if _, err := repo.GetByEmail(ctx, input.Email); err == nil {
			return model.Doctor{}, ErrDoctorEmailExists
		} else if err != ErrDoctorNotFound {
			return model.Doctor{}, err
		}

		doctor := model.Doctor{
			ID:             uuid.New(),
			FullName:       input.FullName,
			Specialization: input.Specialization,
			Email:          input.Email,
		}

		return repo.Create(ctx, doctor)
	})
	if err == nil {
		if u.publisher != nil {
			_ = u.publisher.Publish(ctx, "doctors.created", DoctorCreatedEvent{
				EventType:      "doctors.created",
				OccurredAt:     time.Now().Format(time.RFC3339),
				ID:             created.ID.String(),
				FullName:       created.FullName,
				Specialization: created.Specialization,
				Email:          created.Email,
			})
		}
		// Invalidate list key (Write-Through as per prompt, though it says invalidate)
		if u.cache != nil {
			if err := u.cache.InvalidateDoctorsList(ctx); err != nil {
				log.Printf("Cache invalidation error: %v", err)
			}
		}
	}

	return created, err
}

func (u *doctorUsecase) GetByID(ctx context.Context, id uuid.UUID) (model.Doctor, error) {
	if u.cache != nil {
		if cached, err := u.cache.GetDoctor(ctx, id.String()); err == nil && cached != nil {
			return *cached, nil
		} else if err != nil {
			log.Printf("Cache get error: %v", err)
		}
	}

	doc, err := u.repo.GetByID(ctx, id)
	if err == nil && u.cache != nil {
		if err := u.cache.SetDoctor(ctx, &doc); err != nil {
			log.Printf("Cache set error: %v", err)
		}
	}
	return doc, err
}

func (u *doctorUsecase) List(ctx context.Context) ([]model.Doctor, error) {
	if u.cache != nil {
		if cached, err := u.cache.GetDoctorsList(ctx); err == nil && cached != nil {
			docs := make([]model.Doctor, len(cached))
			for i, v := range cached {
				docs[i] = *v
			}
			return docs, nil
		} else if err != nil {
			log.Printf("Cache list error: %v", err)
		}
	}

	docs, err := u.repo.List(ctx)
	if err == nil && u.cache != nil {
		ptrDocs := make([]*model.Doctor, len(docs))
		for i := range docs {
			ptrDocs[i] = &docs[i]
		}
		if err := u.cache.SetDoctorsList(ctx, ptrDocs); err != nil {
			log.Printf("Cache list set error: %v", err)
		}
	}
	return docs, err
}
