package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"windsurf-gateway/internal/database"
)

type CacheService struct {
	redis *database.RedisClient
}

func NewCacheService(redis *database.RedisClient) *CacheService {
	return &CacheService{redis: redis}
}

func (s *CacheService) Set(key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.redis.GetClient().Set(context.Background(), key, data, expiration).Err()
}

func (s *CacheService) Get(key string, dest interface{}) error {
	data, err := s.redis.GetClient().Get(context.Background(), key).Result()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(data), dest)
}

func (s *CacheService) Delete(key string) error {
	return s.redis.GetClient().Del(context.Background(), key).Err()
}

func (s *CacheService) Exists(key string) (bool, error) {
	n, err := s.redis.GetClient().Exists(context.Background(), key).Result()
	return n > 0, err
}

func (s *CacheService) Incr(key string) (int64, error) {
	return s.redis.GetClient().Incr(context.Background(), key).Result()
}

func (s *CacheService) Expire(key string, expiration time.Duration) error {
	return s.redis.GetClient().Expire(context.Background(), key, expiration).Err()
}

func (s *CacheService) GetRateLimit(key string, limit int, window time.Duration) (bool, int, error) {
	ctx := context.Background()
	client := s.redis.GetClient()

	count, err := client.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, err
	}

	if count == 1 {
		client.Expire(ctx, key, window)
	}

	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return count <= int64(limit), remaining, nil
}

func (s *CacheService) CacheToken(token *database.Token) error {
	key := fmt.Sprintf("token:%s", token.ID)
	return s.Set(key, token, 30*time.Minute)
}

func (s *CacheService) GetCachedToken(id string) (*database.Token, error) {
	var token database.Token
	key := fmt.Sprintf("token:%s", id)
	if err := s.Get(key, &token); err != nil {
		return nil, err
	}
	return &token, nil
}
