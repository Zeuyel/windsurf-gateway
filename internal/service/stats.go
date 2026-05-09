package service

import (
	"fmt"
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
	var totalTokens int64
	var totalRequests, successRequests, failedRequests int64
	var avgLatencyMicros float64
	var totalLatencyMicros int64

	s.db.Model(&database.User{}).Count(&totalUsers)
	s.db.Model(&database.User{}).Where("status = ?", "active").Count(&activeUsers)
	s.db.Model(&database.User{}).Where("status = ?", "banned").Count(&bannedUsers)
	s.db.Model(&database.Token{}).Count(&totalTokens)
	s.db.Model(&database.RequestLog{}).Count(&totalRequests)
	s.db.Model(&database.RequestLog{}).Where("status_code >= 200 AND status_code < 400").Count(&successRequests)
	s.db.Model(&database.RequestLog{}).Where("status_code >= 400").Count(&failedRequests)
	s.db.Model(&database.RequestLog{}).Select("COALESCE(AVG(latency), 0)").Scan(&avgLatencyMicros)
	s.db.Model(&database.RequestLog{}).Select("COALESCE(SUM(latency), 0)").Scan(&totalLatencyMicros)

	tokenStats := map[string]interface{}{}
	if tokenService := NewTokenService(s.db, s.cache); tokenService != nil {
		stats, err := tokenService.GetStats()
		if err != nil {
			return nil, err
		}
		tokenStats = stats
	}

	strategy := string(StrategyRoundRobin)
	var lbConfig database.SystemConfig
	if err := s.db.Where("`key` = ?", "load_balancer_strategy").First(&lbConfig).Error; err == nil && lbConfig.Value != "" {
		strategy = lbConfig.Value
	}

	return map[string]interface{}{
		"total_users":            totalUsers,
		"active_users":           activeUsers,
		"banned_users":           bannedUsers,
		"total_tokens":           totalTokens,
		"active_tokens":          tokenStats["active"],
		"available_tokens":       tokenStats["available"],
		"cooldown_tokens":        tokenStats["cooldown"],
		"disabled_tokens":        tokenStats["disabled"],
		"expired_tokens":         tokenStats["expired"],
		"exhausted_tokens":       tokenStats["exhausted"],
		"total_active_requests":  tokenStats["total_active_requests"],
		"total_requests":         totalRequests,
		"success_requests":       successRequests,
		"failed_requests":        failedRequests,
		"avg_latency_ms":         avgLatencyMicros / 1000,
		"total_latency_us":       totalLatencyMicros,
		"load_balancer_strategy": strategy,
		"token_usage":            s.getTokenUsage(10),
		"token_failures":         s.getTokenFailures(10),
		"error_categories":       s.getErrorCategories(10),
		"method_distribution":    s.getMethodDistribution(),
		"top_users":              s.getTopUsers(10),
		"recent_errors":          s.getRecentErrors(10),
	}, nil
}

func (s *StatsService) GetTrend(rangeLabel string) ([]map[string]interface{}, error) {
	now := time.Now()
	var points int
	var step time.Duration
	var truncateToDay bool

	switch rangeLabel {
	case "30d":
		points = 30
		step = 24 * time.Hour
		truncateToDay = true
	case "24h":
		points = 24
		step = time.Hour
	default:
		points = 7
		step = 24 * time.Hour
		truncateToDay = true
	}

	results := make([]map[string]interface{}, 0, points)
	for i := points - 1; i >= 0; i-- {
		end := now.Add(-time.Duration(i) * step)
		var start time.Time
		if truncateToDay {
			start = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
			end = start.Add(24 * time.Hour)
		} else {
			start = end.Truncate(time.Hour)
			end = start.Add(time.Hour)
		}

		stats, err := s.getIntervalRequestStats(start, end)
		if err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"time":          start.Format(time.RFC3339),
			"total_count":   stats.Total,
			"success_count": stats.Success,
			"failed_count":  stats.Failed,
		})
	}

	return results, nil
}

func (s *StatsService) GetTokenStats(tokenID string) (map[string]interface{}, error) {
	var token database.Token
	if err := s.db.First(&token, "id = ?", tokenID).Error; err != nil {
		return nil, err
	}

	var totalRequests, successRequests, failedRequests int64
	var avgLatencyMicros float64
	var lastRequest time.Time

	s.db.Model(&database.RequestLog{}).Where("token_id = ?", tokenID).Count(&totalRequests)
	s.db.Model(&database.RequestLog{}).Where("token_id = ? AND status_code >= 200 AND status_code < 400", tokenID).Count(&successRequests)
	s.db.Model(&database.RequestLog{}).Where("token_id = ? AND status_code >= 400", tokenID).Count(&failedRequests)
	s.db.Model(&database.RequestLog{}).Where("token_id = ?", tokenID).Select("COALESCE(AVG(latency), 0)").Scan(&avgLatencyMicros)
	s.db.Model(&database.RequestLog{}).Where("token_id = ?", tokenID).Select("MAX(created_at)").Scan(&lastRequest)

	return map[string]interface{}{
		"token":            token,
		"total_requests":   totalRequests,
		"success_requests": successRequests,
		"failed_requests":  failedRequests,
		"avg_latency_ms":   avgLatencyMicros / 1000,
		"last_request":     lastRequest,
		"error_categories": s.getTokenErrorCategories(tokenID, 10),
	}, nil
}

func (s *StatsService) GetUsage(page, pageSize int) ([]database.RequestLog, int64, error) {
	var logs []database.RequestLog
	var total int64

	if err := s.db.Model(&database.RequestLog{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := s.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (s *StatsService) CleanupOldLogs(beforeDays int) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -beforeDays)
	result := s.db.Where("created_at < ?", cutoff).Delete(&database.RequestLog{})
	return result.RowsAffected, result.Error
}

type intervalRequestStats struct {
	Total   int64
	Success int64
	Failed  int64
}

func (s *StatsService) getIntervalRequestStats(start, end time.Time) (*intervalRequestStats, error) {
	stats := &intervalRequestStats{}

	type queryResult struct {
		Total   int64
		Success int64
		Failed  int64
	}
	var result queryResult
	if err := s.db.Model(&database.RequestLog{}).
		Select(
			"COUNT(*) AS total, "+
				"SUM(CASE WHEN status_code >= 200 AND status_code < 400 THEN 1 ELSE 0 END) AS success, "+
				"SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END) AS failed",
		).
		Where("created_at >= ? AND created_at < ?", start, end).
		Scan(&result).Error; err != nil {
		return nil, err
	}

	stats.Total = result.Total
	stats.Success = result.Success
	stats.Failed = result.Failed
	return stats, nil
}

func (s *StatsService) getTokenUsage(limit int) []map[string]interface{} {
	type row struct {
		TokenID   string
		TokenName string
		Requests  int64
	}
	var rows []row
	_ = s.db.Model(&database.RequestLog{}).
		Select("token_id, MAX(token_name) AS token_name, COUNT(*) AS requests").
		Where("token_id <> ''").
		Group("token_id").
		Order("requests DESC").
		Limit(limit).
		Scan(&rows).Error

	result := make([]map[string]interface{}, 0, len(rows))
	for _, item := range rows {
		result = append(result, map[string]interface{}{
			"token_id":   item.TokenID,
			"token_name": coalesceLabel(item.TokenName, item.TokenID),
			"requests":   item.Requests,
		})
	}
	return result
}

func (s *StatsService) getTokenFailures(limit int) []map[string]interface{} {
	type row struct {
		TokenID   string
		TokenName string
		Failures  int64
	}
	var rows []row
	_ = s.db.Model(&database.RequestLog{}).
		Select("token_id, MAX(token_name) AS token_name, COUNT(*) AS failures").
		Where("token_id <> '' AND status_code >= 400").
		Group("token_id").
		Order("failures DESC").
		Limit(limit).
		Scan(&rows).Error

	result := make([]map[string]interface{}, 0, len(rows))
	for _, item := range rows {
		result = append(result, map[string]interface{}{
			"token_id":   item.TokenID,
			"token_name": coalesceLabel(item.TokenName, item.TokenID),
			"failures":   item.Failures,
		})
	}
	return result
}

func (s *StatsService) getErrorCategories(limit int) []map[string]interface{} {
	type row struct {
		FailureCategory string
		Count           int64
	}
	var rows []row
	_ = s.db.Model(&database.RequestLog{}).
		Select("failure_category, COUNT(*) AS count").
		Where("failure_category <> ''").
		Group("failure_category").
		Order("count DESC").
		Limit(limit).
		Scan(&rows).Error

	result := make([]map[string]interface{}, 0, len(rows))
	for _, item := range rows {
		result = append(result, map[string]interface{}{
			"failure_category": item.FailureCategory,
			"count":            item.Count,
		})
	}
	return result
}

func (s *StatsService) getMethodDistribution() []map[string]interface{} {
	type row struct {
		Method string
		Count  int64
	}
	var rows []row
	_ = s.db.Model(&database.RequestLog{}).
		Select("method, COUNT(*) AS count").
		Group("method").
		Order("count DESC").
		Scan(&rows).Error

	result := make([]map[string]interface{}, 0, len(rows))
	for _, item := range rows {
		result = append(result, map[string]interface{}{
			"method": coalesceLabel(item.Method, "UNKNOWN"),
			"count":  item.Count,
		})
	}
	return result
}

func (s *StatsService) getTopUsers(limit int) []map[string]interface{} {
	type row struct {
		Username string
		Requests int64
	}
	var rows []row
	_ = s.db.Model(&database.RequestLog{}).
		Select("username, COUNT(*) AS requests").
		Where("username <> ''").
		Group("username").
		Order("requests DESC").
		Limit(limit).
		Scan(&rows).Error

	result := make([]map[string]interface{}, 0, len(rows))
	for _, item := range rows {
		result = append(result, map[string]interface{}{
			"username": coalesceLabel(item.Username, "anonymous"),
			"requests": item.Requests,
		})
	}
	return result
}

func (s *StatsService) getRecentErrors(limit int) []map[string]interface{} {
	var rows []database.RequestLog
	_ = s.db.Where("status_code >= 400 OR failure_category <> ''").Order("created_at DESC").Limit(limit).Find(&rows).Error

	result := make([]map[string]interface{}, 0, len(rows))
	for _, item := range rows {
		result = append(result, map[string]interface{}{
			"request_id":       item.RequestID,
			"token_name":       coalesceLabel(item.TokenName, item.TokenID),
			"path":             item.Path,
			"status_code":      item.StatusCode,
			"failure_category": item.FailureCategory,
			"error_message":    item.ErrorMessage,
			"created_at":       item.CreatedAt.Format(time.RFC3339),
		})
	}
	return result
}

func (s *StatsService) getTokenErrorCategories(tokenID string, limit int) []map[string]interface{} {
	type row struct {
		FailureCategory string
		Count           int64
	}
	var rows []row
	_ = s.db.Model(&database.RequestLog{}).
		Select("failure_category, COUNT(*) AS count").
		Where("token_id = ? AND failure_category <> ''", tokenID).
		Group("failure_category").
		Order("count DESC").
		Limit(limit).
		Scan(&rows).Error

	result := make([]map[string]interface{}, 0, len(rows))
	for _, item := range rows {
		result = append(result, map[string]interface{}{
			"failure_category": item.FailureCategory,
			"count":            item.Count,
		})
	}
	return result
}

func coalesceLabel(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func (s *StatsService) DebugString() string {
	return fmt.Sprintf("StatsService<db=%p>", s.db)
}
