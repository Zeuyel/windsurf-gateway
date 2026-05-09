package service

import (
	"windsurf-gateway/internal/database"

	"gorm.io/gorm"
)

type PluginService struct {
	db *gorm.DB
}

func NewPluginService(db *gorm.DB) *PluginService {
	return &PluginService{db: db}
}

func (s *PluginService) Create(plugin *database.Plugin) error {
	return s.db.Create(plugin).Error
}

func (s *PluginService) GetList() ([]database.Plugin, error) {
	var plugins []database.Plugin
	if err := s.db.Order("created_at DESC").Find(&plugins).Error; err != nil {
		return nil, err
	}
	return plugins, nil
}

func (s *PluginService) GetByID(id uint) (*database.Plugin, error) {
	var plugin database.Plugin
	if err := s.db.First(&plugin, id).Error; err != nil {
		return nil, err
	}
	return &plugin, nil
}

func (s *PluginService) IncrementDownload(id uint) error {
	return s.db.Model(&database.Plugin{}).Where("id = ?", id).
		UpdateColumn("download_count", gorm.Expr("download_count + 1")).Error
}

func (s *PluginService) Delete(id uint) error {
	return s.db.Delete(&database.Plugin{}, id).Error
}
