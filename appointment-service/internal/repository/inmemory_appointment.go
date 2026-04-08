package repository

import (
	"context"
	"sync"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"
)

type InMemoryAppointmentRepository struct {
	mu   sync.RWMutex
	byID map[string]model.Appointment
}

func NewInMemoryAppointmentRepository() *InMemoryAppointmentRepository {
	return &InMemoryAppointmentRepository{
		byID: make(map[string]model.Appointment),
	}
}

func (r *InMemoryAppointmentRepository) Create(_ context.Context, appointment model.Appointment) (model.Appointment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[appointment.ID] = appointment
	return appointment, nil
}

func (r *InMemoryAppointmentRepository) GetByID(_ context.Context, id string) (model.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	appointment, ok := r.byID[id]
	if !ok {
		return model.Appointment{}, usecase.ErrAppointmentNotFound
	}
	return appointment, nil
}

func (r *InMemoryAppointmentRepository) List(_ context.Context) ([]model.Appointment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	appointments := make([]model.Appointment, 0, len(r.byID))
	for _, appointment := range r.byID {
		appointments = append(appointments, appointment)
	}
	return appointments, nil
}

func (r *InMemoryAppointmentRepository) Update(_ context.Context, appointment model.Appointment) (model.Appointment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[appointment.ID] = appointment
	return appointment, nil
}
