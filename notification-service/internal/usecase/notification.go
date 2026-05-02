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
	u.logger.Infof("Processing event from subject: %s", event.Subject)

	// Parse event payload
	eventData, err := json.Marshal(event.Event)
	if err != nil {
		u.logger.Errorf("Failed to marshal event: %v", err)
		return err
	}

	// Create notification
	notification := model.Notification{
		Title:     formatTitle(event.Subject),
		Message:   string(eventData),
		Timestamp: time.Now(),
	}

	// Log notification
	u.logger.Infof("Notification created: %s - %s", notification.Title, notification.Message)

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
