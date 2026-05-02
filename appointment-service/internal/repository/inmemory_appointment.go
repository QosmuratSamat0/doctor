package repository

import (
	"context"
	"sync"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"
	"github.com/google/uuid"
)

type InMemoryAppointmentRepository struct {
	mu   sync.RWMutex
	byID map[uuid.UUID]model.Appointment
}

func NewInMemoryAppointmentRepository() *InMemoryAppointmentRepository {
	return &InMemoryAppointmentRepository{
		byID: make(map[uuid.UUID]model.Appointment),
	}
}

func (r *InMemoryAppointmentRepository) Create(_ context.Context, appointment model.Appointment) (model.Appointment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[appointment.ID] = appointment
	return appointment, nil
}

func (r *InMemoryAppointmentRepository) GetByID(_ context.Context, id uuid.UUID) (model.Appointment, error) {
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

func (r *InMemoryAppointmentRepository) WithTx(ctx context.Context, fn func(usecase.AppointmentRepository) (model.Appointment, error)) (model.Appointment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return fn(&inMemoryAppointmentTxRepo{parent: r})
}

type inMemoryAppointmentTxRepo struct {
	parent *InMemoryAppointmentRepository
}

func (r *inMemoryAppointmentTxRepo) Create(_ context.Context, appointment model.Appointment) (model.Appointment, error) {
	r.parent.byID[appointment.ID] = appointment
	return appointment, nil
}

func (r *inMemoryAppointmentTxRepo) GetByID(_ context.Context, id uuid.UUID) (model.Appointment, error) {
	appointment, ok := r.parent.byID[id]
	if !ok {
		return model.Appointment{}, usecase.ErrAppointmentNotFound
	}
	return appointment, nil
}

func (r *inMemoryAppointmentTxRepo) List(_ context.Context) ([]model.Appointment, error) {
	appointments := make([]model.Appointment, 0, len(r.parent.byID))
	for _, appointment := range r.parent.byID {
		appointments = append(appointments, appointment)
	}
	return appointments, nil
}

func (r *inMemoryAppointmentTxRepo) Update(_ context.Context, appointment model.Appointment) (model.Appointment, error) {
	r.parent.byID[appointment.ID] = appointment
	return appointment, nil
}

func (r *inMemoryAppointmentTxRepo) WithTx(ctx context.Context, fn func(usecase.AppointmentRepository) (model.Appointment, error)) (model.Appointment, error) {
	return fn(r)
}
