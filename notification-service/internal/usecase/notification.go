package usecase

import (
	"context"
	"encoding/json"
	"time"

	"notification-service/internal/logger"
	"notification-service/internal/model"
)

type NotificationUsecase interface {
	HandleEvent(ctx context.Context, event model.Event) error
}

type notificationUsecase struct {
	logger logger.Logger
}

func NewNotificationUsecase(log logger.Logger) NotificationUsecase {
	return &notificationUsecase{
		logger: log,
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
