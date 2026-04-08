package repository

import (
	"context"
	"sync"

	"doctor-service/internal/model"
	"doctor-service/internal/usecase"
)

type InMemoryDoctorRepository struct {
	mu      sync.RWMutex
	byID    map[string]model.Doctor
	byEmail map[string]string
}

func NewInMemoryDoctorRepository() *InMemoryDoctorRepository {
	return &InMemoryDoctorRepository{
		byID:    make(map[string]model.Doctor),
		byEmail: make(map[string]string),
	}
}

func (r *InMemoryDoctorRepository) Create(_ context.Context, doctor model.Doctor) (model.Doctor, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[doctor.ID] = doctor
	r.byEmail[doctor.Email] = doctor.ID
	return doctor, nil
}

func (r *InMemoryDoctorRepository) GetByID(_ context.Context, id string) (model.Doctor, error) {
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
