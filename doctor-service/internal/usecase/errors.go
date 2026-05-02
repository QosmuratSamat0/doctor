package usecase

import "errors"

var (
	ErrDoctorNotFound    = errors.New("doctor not found")
	ErrDoctorEmailExists = errors.New("doctor email already exists")
	ErrFullNameRequired  = errors.New("full_name is required")
	ErrEmailRequired     = errors.New("email is required")
)
