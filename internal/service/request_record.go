package service

import (
	"windsurf-gateway/internal/database"

	"gorm.io/gorm"
)

type RequestRecordService struct {
	db *gorm.DB
}

func NewRequestRecordService(db *gorm.DB) *RequestRecordService {
	return &RequestRecordService{db: db}
}

func (s *RequestRecordService) Create(log *database.RequestLog) error {
	return s.db.Create(log).Error
}

func (s *RequestRecordService) List(page, pageSize int) ([]database.RequestLog, int64, error) {
	var logs []database.RequestLog
	var total int64

	s.db.Model(&database.RequestLog{}).Count(&total)

	offset := (page - 1) * pageSize
	if err := s.db.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (s *RequestRecordService) Search(query string, page, pageSize int) ([]database.RequestLog, int64, error) {
	var logs []database.RequestLog
	var total int64

	q := s.db.Model(&database.RequestLog{}).
		Where("path LIKE ? OR client_ip LIKE ? OR tenant_address LIKE ?",
			"%"+query+"%", "%"+query+"%", "%"+query+"%")

	q.Count(&total)

	offset := (page - 1) * pageSize
	if err := q.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
