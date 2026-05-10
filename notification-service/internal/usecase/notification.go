package usecase

import (
	"context"
	"encoding/json"
	"time"

	"fmt"
	"notification-service/internal/jobqueue"
	"notification-service/internal/logger"
	"notification-service/internal/model"
)

type NotificationUsecase interface {
	HandleEvent(ctx context.Context, event model.Event) error
}

type notificationUsecase struct {
	logger   logger.Logger
	jobQueue jobqueue.JobQueue
}

func NewNotificationUsecase(log logger.Logger, jobQueue jobqueue.JobQueue) NotificationUsecase {
	return &notificationUsecase{
		logger:   log,
		jobQueue: jobQueue,
	}
}

func (u *notificationUsecase) HandleEvent(ctx context.Context, event model.Event) error {
	logPayload := struct {
		Time    string      `json:"time"`
		Subject string      `json:"subject"`
		Event   interface{} `json:"event"`
	}{
		Time:    event.Time.UTC().Format(time.RFC3339),
		Subject: event.Subject,
		Event:   event.Event,
	}

	data, err := json.Marshal(logPayload)
	if err != nil {
		u.logger.Errorf("Failed to marshal event log: %v", err)
		return err
	}

	u.logger.Infof(string(data))

	if event.Subject == "appointments.status_updated" {
		var payload map[string]interface{}
		// Re-marshal and unmarshal to ensure we have a map
		eventData, _ := json.Marshal(event.Event)
		if err := json.Unmarshal(eventData, &payload); err != nil {
			return nil // Logged but didn't handle job
		}

		if payload["new_status"] == "done" && u.jobQueue != nil {
			id := fmt.Sprintf("%v", payload["id"])
			occurredAt := fmt.Sprintf("%v", payload["occurred_at"])
			doctorID := fmt.Sprintf("%v", payload["doctor_id"])
			eventType := "appointments.status_updated"

			job := jobqueue.Job{
				IdempotencyKey: jobqueue.GenerateIdempotencyKey(eventType, id, occurredAt),
				AppointmentID:  id,
				DoctorID:       doctorID,
				OccurredAt:     occurredAt,
				Channel:        "email",
				Recipient:      "patient@clinic.kz",
				Message:        fmt.Sprintf("Your appointment %s with doctor %s is complete.", id, doctorID),
			}
			u.jobQueue.Enqueue(job)
		}
	}

	return nil
}

func formatTitle(subject string) string {
	titleMap := map[string]string{
		"doctors.created":              "New Doctor",
		"appointments.created":         "New Appointment",
		"appointments.status_updated":  "Appointment Status Updated",
	}

	if title, ok := titleMap[subject]; ok {
		return title
	}
	return subject
}
