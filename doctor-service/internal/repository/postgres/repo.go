package postgres

import (
	"context"
	"errors"
	"fmt"

	"doctor-service/internal/model"
	"doctor-service/internal/usecase"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) Create(ctx context.Context, doctor model.Doctor) (model.Doctor, error) {
	query := `INSERT INTO doctors (id, full_name, specialization, email) VALUES ($1, $2, $3, $4)`
	_, err := r.pool.Exec(ctx, query, doctor.ID, doctor.FullName, doctor.Specialization, doctor.Email)
	if err != nil {
		return model.Doctor{}, fmt.Errorf("create doctor: %w", err)
	}
	return doctor, nil
}

func (r *PostgresRepo) GetByID(ctx context.Context, id string) (model.Doctor, error) {
	query := `SELECT id, full_name, specialization, email FROM doctors WHERE id = $1`
	var doctor model.Doctor
	err := r.pool.QueryRow(ctx, query, id).Scan(&doctor.ID, &doctor.FullName, &doctor.Specialization, &doctor.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Doctor{}, usecase.ErrDoctorNotFound
		}
		return model.Doctor{}, fmt.Errorf("get doctor by id: %w", err)
	}
	return doctor, nil
}

func (r *PostgresRepo) GetByEmail(ctx context.Context, email string) (model.Doctor, error) {
	query := `SELECT id, full_name, specialization, email FROM doctors WHERE email = $1`
	var doctor model.Doctor
	err := r.pool.QueryRow(ctx, query, email).Scan(&doctor.ID, &doctor.FullName, &doctor.Specialization, &doctor.Email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Doctor{}, usecase.ErrDoctorNotFound
		}
		return model.Doctor{}, fmt.Errorf("get doctor by email: %w", err)
	}
	return doctor, nil
}

func (r *PostgresRepo) List(ctx context.Context) ([]model.Doctor, error) {
	query := `SELECT id, full_name, specialization, email FROM doctors`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list doctors: %w", err)
	}
	defer rows.Close()

	var doctors []model.Doctor
	for rows.Next() {
		var doctor model.Doctor
		if err := rows.Scan(&doctor.ID, &doctor.FullName, &doctor.Specialization, &doctor.Email); err != nil {
			return nil, fmt.Errorf("scan doctor: %w", err)
		}
		doctors = append(doctors, doctor)
	}
	return doctors, nil
}
