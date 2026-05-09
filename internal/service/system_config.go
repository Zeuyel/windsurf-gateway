package service

import (
	"errors"
	"windsurf-gateway/internal/database"

	"gorm.io/gorm"
)

type SystemConfigService struct {
	db *gorm.DB
}

func NewSystemConfigService(db *gorm.DB) *SystemConfigService {
	return &SystemConfigService{db: db}
}

func (s *SystemConfigService) Get(key string) (string, error) {
	values := make([]string, 0)
	if err := s.db.Model(&database.SystemConfig{}).Where("`key` = ?", key).Limit(1).Pluck("value", &values).Error; err != nil {
		return "", err
	}
	if len(values) == 0 {
		return "", gorm.ErrRecordNotFound
	}
	return values[0], nil
}

func (s *SystemConfigService) Set(key, value string) error {
	var config database.SystemConfig
	result := s.db.Where("`key` = ?", key).Limit(1).Find(&config)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		config = database.SystemConfig{Key: key, Value: value}
		return s.db.Create(&config).Error
	}

	config.Value = value
	return s.db.Save(&config).Error
}

func (s *SystemConfigService) EnsureDefaults() error {
	defaults := map[string]string{
		"load_balancer_strategy": "round_robin",
	}

	for key, value := range defaults {
		_, err := s.Get(key)
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := s.Set(key, value); err != nil {
			return err
		}
	}
	return nil
}

func (s *SystemConfigService) GetAll() ([]database.SystemConfig, error) {
	var configs []database.SystemConfig
	if err := s.db.Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

func (s *SystemConfigService) GetSystemStats() (map[string]interface{}, error) {
	var totalUsers, activeTokens int64
	var totalRequests int64

	s.db.Model(&database.User{}).Count(&totalUsers)
	s.db.Model(&database.Token{}).Where("status = ?", "active").Count(&activeTokens)
	s.db.Model(&database.RequestLog{}).Count(&totalRequests)

	return map[string]interface{}{
		"total_users":    totalUsers,
		"active_tokens":  activeTokens,
		"total_requests": totalRequests,
	}, nil
}
