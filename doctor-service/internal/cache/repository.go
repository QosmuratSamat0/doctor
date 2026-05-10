package cache

import (
	"context"
	"doctor-service/internal/model"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type CacheRepository interface {
	GetDoctor(ctx context.Context, id string) (*model.Doctor, error)
	SetDoctor(ctx context.Context, doc *model.Doctor) error
	GetDoctorsList(ctx context.Context) ([]*model.Doctor, error)
	SetDoctorsList(ctx context.Context, docs []*model.Doctor) error
	InvalidateDoctorsList(ctx context.Context) error
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

func (r *redisCacheRepository) GetDoctor(ctx context.Context, id string) (*model.Doctor, error) {
	key := fmt.Sprintf("doctor:%s", id)
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var doc model.Doctor
	if err := json.Unmarshal([]byte(val), &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (r *redisCacheRepository) SetDoctor(ctx context.Context, doc *model.Doctor) error {
	key := fmt.Sprintf("doctor:%s", doc.ID.String())
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *redisCacheRepository) GetDoctorsList(ctx context.Context) ([]*model.Doctor, error) {
	key := "doctors:list"
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var docs []*model.Doctor
	if err := json.Unmarshal([]byte(val), &docs); err != nil {
		return nil, err
	}
	return docs, nil
}

func (r *redisCacheRepository) SetDoctorsList(ctx context.Context, docs []*model.Doctor) error {
	key := "doctors:list"
	data, err := json.Marshal(docs)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, key, data, r.ttl).Err()
}

func (r *redisCacheRepository) InvalidateDoctorsList(ctx context.Context) error {
	key := "doctors:list"
	return r.client.Del(ctx, key).Err()
}
