package service

import (
	"time"

	"windsurf-gateway/internal/database"

	"gorm.io/gorm"
)

type StatsService struct {
	db    *gorm.DB
	cache *CacheService
}

func NewStatsService(db *gorm.DB, cache *CacheService) *StatsService {
	return &StatsService{db: db, cache: cache}
}

func (s *StatsService) GetOverview() (map[string]interface{}, error) {
	var totalUsers, activeUsers, bannedUsers int64
	var totalTokens, activeTokens int64
	var totalRequests int64

	s.db.Model(&database.User{}).Count(&totalUsers)
	s.db.Model(&database.User{}).Where("status = ?", "active").Count(&activeUsers)
	s.db.Model(&database.User{}).Where("status = ?", "banned").Count(&bannedUsers)
	s.db.Model(&database.Token{}).Count(&totalTokens)
	s.db.Model(&database.Token{}).Where("status = ?", "active").Count(&activeTokens)
	s.db.Model(&database.RequestLog{}).Count(&totalRequests)

	return map[string]interface{}{
		"total_users":    totalUsers,
		"active_users":   activeUsers,
		"banned_users":   bannedUsers,
		"total_tokens":   totalTokens,
		"active_tokens":  activeTokens,
		"total_requests": totalRequests,
	}, nil
}

func (s *StatsService) GetTrend(days int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	for i := days - 1; i >= 0; i-- {
		date := time.Now().AddDate(0, 0, -i)
		startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.Local)
		endOfDay := startOfDay.Add(24 * time.Hour)

		var count int64
		s.db.Model(&database.RequestLog{}).
			Where("created_at >= ? AND created_at < ?", startOfDay, endOfDay).
			Count(&count)

		results = append(results, map[string]interface{}{
			"date":  startOfDay.Format("2006-01-02"),
			"count": count,
		})
	}

	return results, nil
}

func (s *StatsService) GetTokenStats(tokenID string) (map[string]interface{}, error) {
	var totalRequests int64
	var avgLatency float64

	s.db.Model(&database.RequestLog{}).
		Where("token_id = ?", tokenID).
		Count(&totalRequests)

	s.db.Model(&database.RequestLog{}).
		Where("token_id = ?", tokenID).
		Select("COALESCE(AVG(latency), 0)").
		Scan(&avgLatency)

	return map[string]interface{}{
		"total_requests": totalRequests,
		"avg_latency":    avgLatency,
	}, nil
}

func (s *StatsService) GetUsage(page, pageSize int) ([]database.RequestLog, int64, error) {
	var logs []database.RequestLog
	var total int64

	s.db.Model(&database.RequestLog{}).Count(&total)

	offset := (page - 1) * pageSize
	if err := s.db.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (s *StatsService) CleanupOldLogs(beforeDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -beforeDays)
	result := s.db.Where("created_at < ?", cutoff).Delete(&database.RequestLog{})
	return result.RowsAffected, result.Error
}
