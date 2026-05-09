package service

import (
	"strconv"
	"strings"

	"windsurf-gateway/internal/database"

	"gorm.io/gorm"
)

type RequestLogFilter struct {
	Query           string
	TokenID         string
	Username        string
	FailureCategory string
	StatusCode      int
}

type RequestRecordService struct {
	db *gorm.DB
}

func NewRequestRecordService(db *gorm.DB) *RequestRecordService {
	return &RequestRecordService{db: db}
}

func (s *RequestRecordService) Create(log *database.RequestLog) error {
	return s.db.Create(log).Error
}

func (s *RequestRecordService) List(page, pageSize int, filter RequestLogFilter) ([]database.RequestLog, int64, error) {
	var logs []database.RequestLog
	var total int64

	query := s.applyFilters(s.db.Model(&database.RequestLog{}), filter)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (s *RequestRecordService) Search(query string, page, pageSize int) ([]database.RequestLog, int64, error) {
	return s.List(page, pageSize, RequestLogFilter{Query: query})
}

func (s *RequestRecordService) applyFilters(query *gorm.DB, filter RequestLogFilter) *gorm.DB {
	if filter.Query != "" {
		like := "%" + strings.TrimSpace(filter.Query) + "%"
		query = query.Where(
			"path LIKE ? OR client_ip LIKE ? OR tenant_address LIKE ? OR request_id LIKE ? OR token_name LIKE ? OR username LIKE ? OR error_message LIKE ?",
			like, like, like, like, like, like, like,
		)
	}
	if filter.TokenID != "" {
		query = query.Where("token_id = ?", filter.TokenID)
	}
	if filter.Username != "" {
		query = query.Where("username = ?", filter.Username)
	}
	if filter.FailureCategory != "" {
		query = query.Where("failure_category = ?", filter.FailureCategory)
	}
	if filter.StatusCode > 0 {
		query = query.Where("status_code = ?", filter.StatusCode)
	}
	return query
}

func ParseStatusCodeFilter(value string) int {
	if value == "" {
		return 0
	}
	statusCode, err := strconv.Atoi(value)
	if err != nil || statusCode < 100 {
		return 0
	}
	return statusCode
}
