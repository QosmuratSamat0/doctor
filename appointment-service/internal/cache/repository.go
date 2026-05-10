package cache

import (
	"appointment-service/internal/model"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheRepository interface {
	GetAppointment(ctx context.Context, id string) (*model.Appointment, error)
	SetAppointment(ctx context.Context, appt *model.Appointment) error
	GetAppointmentsList(ctx context.Context) ([]*model.Appointment, error)
	SetAppointmentsList(ctx context.Context, appts []*model.Appointment) error
	InvalidateAppointmentsList(ctx context.Context) error
	InvalidateAppointment(ctx context.Context, id string) error
}

type redisCacheRepository struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCacheRepository(client *redis.Client, ttl time.Duration) CacheRepository {
	return &redisCacheRepository{
		client: client,
		ttl:    ttl,
	}
}

func (r *redisCacheRepository) GetAppointment(ctx context.Context, id string) (*model.Appointment, error) {
	key := fmt.Sprintf("appointment:%s", id)
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var appt model.Appointment
	if err := json.Unmarshal([]byte(val), &appt); err != nil {
		return nil, err
	}
	return &appt, nil
}

func (r *redisCacheRepository) SetAppointment(ctx context.Context, appt *model.Appointment) error {
	key := fmt.Sprintf("appointment:%s", appt.ID.String())
	data, err := json.Marshal(appt)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *redisCacheRepository) GetAppointmentsList(ctx context.Context) ([]*model.Appointment, error) {
	key := "appointments:list"
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var appts []*model.Appointment
	if err := json.Unmarshal([]byte(val), &appts); err != nil {
		return nil, err
	}
	return appts, nil
}

func (r *redisCacheRepository) SetAppointmentsList(ctx context.Context, appts []*model.Appointment) error {
	key := "appointments:list"
	data, err := json.Marshal(appts)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *redisCacheRepository) InvalidateAppointmentsList(ctx context.Context) error {
	return r.client.Del(ctx, "appointments:list").Err()
}

func (r *redisCacheRepository) InvalidateAppointment(ctx context.Context, id string) error {
	key := fmt.Sprintf("appointment:%s", id)
	return r.client.Del(ctx, key).Err()
}
