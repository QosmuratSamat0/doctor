package model

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusNew        Status = "new"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

type Appointment struct {
	ID          uuid.UUID
	Title       string
	Description string
	DoctorID    uuid.UUID
	Status      Status
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
