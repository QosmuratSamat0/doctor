package model

import "github.com/google/uuid"

type Doctor struct {
	ID             uuid.UUID
	FullName       string
	Specialization string
	Email          string
}
