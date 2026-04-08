package usecase

import (
	"context"
	"fmt"
	"sync/atomic"

	"doctor-service/internal/model"
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

type doctorUsecase struct {
	repo      DoctorRepository
	idCounter atomic.Uint64
}

func NewDoctorUsecase(repo DoctorRepository) DoctorUsecase {
	return &doctorUsecase{repo: repo}
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
		ID:             fmt.Sprintf("doctor-%d", u.idCounter.Add(1)),
		FullName:       input.FullName,
		Specialization: input.Specialization,
		Email:          input.Email,
	}

	return u.repo.Create(ctx, doctor)
}

func (u *doctorUsecase) GetByID(ctx context.Context, id string) (model.Doctor, error) {
	return u.repo.GetByID(ctx, id)
}

func (u *doctorUsecase) List(ctx context.Context) ([]model.Doctor, error) {
	return u.repo.List(ctx)
}
