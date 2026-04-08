package usecase

import "errors"

var (
	ErrAppointmentNotFound      = errors.New("appointment not found")
	ErrTitleRequired            = errors.New("title is required")
	ErrDoctorIDRequired         = errors.New("doctor_id is required")
	ErrInvalidStatus            = errors.New("status must be one of: new, in_progress, done")
	ErrInvalidStatusTransition  = errors.New("transition from done back to new is not allowed")
	ErrDoctorNotFound           = errors.New("doctor does not exist")
	ErrDoctorServiceUnavailable = errors.New("doctor service is unavailable; appointment operation cannot proceed")
)
