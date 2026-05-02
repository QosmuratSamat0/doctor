package model

import "time"

type Event struct {
	Time    time.Time
	Subject string
	Event   interface{}
}


	