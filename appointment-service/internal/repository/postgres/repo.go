package postgres

import (
	"context"
	"errors"
	"fmt"

	"appointment-service/internal/model"
	"appointment-service/internal/usecase"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Create(ctx context.Context, appointment model.Appointment) (model.Appointment, error) {
	query := `INSERT INTO appointments (id, title, description, doctor_id, status, created_at, updated_at) 
              VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.pool.Exec(ctx, query, 
		appointment.ID, 
		appointment.Title, 
		appointment.Description, 
		appointment.DoctorID, 
		string(appointment.Status), 
		appointment.CreatedAt, 
		appointment.UpdatedAt)
	if err != nil {
		return model.Appointment{}, fmt.Errorf("create appointment: %w", err)
	}
	return appointment, nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id string) (model.Appointment, error) {
	query := `SELECT id, title, description, doctor_id, status, created_at, updated_at FROM appointments WHERE id = $1`
	var a model.Appointment
	var status string
	err := r.pool.QueryRow(ctx, query, id).Scan(&a.ID, &a.Title, &a.Description, &a.DoctorID, &status, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Appointment{}, usecase.ErrAppointmentNotFound
		}
		return model.Appointment{}, fmt.Errorf("get appointment by id: %w", err)
	}
	a.Status = model.Status(status)
	return a, nil
}

func (r *PostgresRepo) List(ctx context.Context) ([]model.Appointment, error) {
	query := `SELECT id, title, description, doctor_id, status, created_at, updated_at FROM appointments`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list appointments: %w", err)
	}
	defer rows.Close()

	var appointments []model.Appointment
	for rows.Next() {
		var a model.Appointment
		var status string
		if err := rows.Scan(&a.ID, &a.Title, &a.Description, &a.DoctorID, &status, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan appointment: %w", err)
		}
		a.Status = model.Status(status)
		appointments = append(appointments, a)
	}
	return appointments, nil
}

func (r *PostgresRepo) Update(ctx context.Context, appointment model.Appointment) (model.Appointment, error) {
	query := `UPDATE appointments SET title = $2, description = $3, doctor_id = $4, status = $5, updated_at = $6 WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, 
		appointment.ID, 
		appointment.Title, 
		appointment.Description, 
		appointment.DoctorID, 
		string(appointment.Status), 
		appointment.UpdatedAt)
	if err != nil {
		return model.Appointment{}, fmt.Errorf("update appointment: %w", err)
	}
	return appointment, nil
}
