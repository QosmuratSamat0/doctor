package repository

import (
	"context"
	"sync"

	"doctor-service/internal/model"
	"doctor-service/internal/usecase"
	"github.com/google/uuid"
)

type InMemoryDoctorRepository struct {
	mu      sync.RWMutex
	byID    map[uuid.UUID]model.Doctor
	byEmail map[string]uuid.UUID
}

func NewInMemoryDoctorRepository() *InMemoryDoctorRepository {
	return &InMemoryDoctorRepository{
		byID:    make(map[uuid.UUID]model.Doctor),
		byEmail: make(map[string]uuid.UUID),
	}
}

func (r *InMemoryDoctorRepository) Create(_ context.Context, doctor model.Doctor) (model.Doctor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[doctor.ID] = doctor
	r.byEmail[doctor.Email] = doctor.ID
	return doctor, nil
}

func (r *InMemoryDoctorRepository) GetByID(_ context.Context, id uuid.UUID) (model.Doctor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	doctor, ok := r.byID[id]
	if !ok {
		return model.Doctor{}, usecase.ErrDoctorNotFound
	}
	return doctor, nil
}

func (r *InMemoryDoctorRepository) GetByEmail(_ context.Context, email string) (model.Doctor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byEmail[email]
	if !ok {
		return model.Doctor{}, usecase.ErrDoctorNotFound
	}
	return r.byID[id], nil
}

func (r *InMemoryDoctorRepository) List(_ context.Context) ([]model.Doctor, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	doctors := make([]model.Doctor, 0, len(r.byID))
	for _, doctor := range r.byID {
		doctors = append(doctors, doctor)
	}
	return doctors, nil
}

func (r *InMemoryDoctorRepository) WithTx(ctx context.Context, fn func(usecase.DoctorRepository) (model.Doctor, error)) (model.Doctor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return fn(&inMemoryDoctorTxRepo{parent: r})
}

type inMemoryDoctorTxRepo struct {
	parent *InMemoryDoctorRepository
}

func (r *inMemoryDoctorTxRepo) Create(_ context.Context, doctor model.Doctor) (model.Doctor, error) {
	r.parent.byID[doctor.ID] = doctor
	r.parent.byEmail[doctor.Email] = doctor.ID
	return doctor, nil
}

func (r *inMemoryDoctorTxRepo) GetByID(_ context.Context, id uuid.UUID) (model.Doctor, error) {
	doctor, ok := r.parent.byID[id]
	if !ok {
		return model.Doctor{}, usecase.ErrDoctorNotFound
	}
	return doctor, nil
}

func (r *inMemoryDoctorTxRepo) GetByEmail(_ context.Context, email string) (model.Doctor, error) {
	id, ok := r.parent.byEmail[email]
	if !ok {
		return model.Doctor{}, usecase.ErrDoctorNotFound
	}
	return r.parent.byID[id], nil
}

func (r *inMemoryDoctorTxRepo) List(_ context.Context) ([]model.Doctor, error) {
	doctors := make([]model.Doctor, 0, len(r.parent.byID))
	for _, doctor := range r.parent.byID {
		doctors = append(doctors, doctor)
	}
	return doctors, nil
}

func (r *inMemoryDoctorTxRepo) WithTx(ctx context.Context, fn func(usecase.DoctorRepository) (model.Doctor, error)) (model.Doctor, error) {
	return fn(r)
}
