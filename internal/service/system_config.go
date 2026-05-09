package service

import (
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
	var config database.SystemConfig
	if err := s.db.Where("`key` = ?", key).First(&config).Error; err != nil {
		return "", err
	}
	return config.Value, nil
}

func (s *SystemConfigService) Set(key, value string) error {
	var config database.SystemConfig
	result := s.db.Where("`key` = ?", key).First(&config)

	if result.Error != nil {
		config = database.SystemConfig{Key: key, Value: value}
		return s.db.Create(&config).Error
	}

	config.Value = value
	return s.db.Save(&config).Error
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
