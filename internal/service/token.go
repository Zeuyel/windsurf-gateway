package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"windsurf-gateway/internal/database"
	"windsurf-gateway/internal/logger"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	TokenStatusActive   = "active"
	TokenStatusDisabled = "disabled"
	TokenStatusExpired  = "expired"

	TokenPoolAvailable = "available"
	TokenPoolCooldown  = "cooldown"
	TokenPoolDisabled  = "disabled"
	TokenPoolExpired   = "expired"
	TokenPoolExhausted = "exhausted"
)

type TokenService struct {
	db    *gorm.DB
	cache *CacheService
}

func NewTokenService(db *gorm.DB, cache *CacheService) *TokenService {
	return &TokenService{db: db, cache: cache}
}

func (s *TokenService) Create(token *database.Token) error {
	token.ID = uuid.New().String()[:20]
	now := time.Now()
	token.Status, token.PoolStatus = normalizeTokenState(token, now)
	if token.Weight <= 0 {
		token.Weight = 1
	}
	return s.db.Create(token).Error
}

func (s *TokenService) GetByID(id string) (*database.Token, error) {
	if err := s.RefreshTokenStateByID(id); err != nil {
		return nil, err
	}

	var token database.Token
	if err := s.db.First(&token, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (s *TokenService) List(page, pageSize int, status, poolStatus string) ([]database.Token, int64, error) {
	if err := s.RefreshTokenStates(); err != nil {
		return nil, 0, err
	}

	var tokens []database.Token
	var total int64

	query := s.db.Model(&database.Token{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if poolStatus != "" {
		query = query.Where("pool_status = ?", poolStatus)
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("updated_at DESC").Offset(offset).Limit(pageSize).Find(&tokens).Error; err != nil {
		return nil, 0, err
	}

	return tokens, total, nil
}

func (s *TokenService) Update(id string, updates map[string]interface{}) error {
	normalized, err := normalizeTokenUpdates(updates)
	if err != nil {
		return err
	}
	if len(normalized) == 0 {
		return nil
	}
	if err := s.db.Model(&database.Token{}).Where("id = ?", id).Updates(normalized).Error; err != nil {
		return err
	}
	return s.RefreshTokenStateByID(id)
}

func (s *TokenService) Delete(id string) error {
	return s.db.Where("id = ?", id).Delete(&database.Token{}).Error
}

func (s *TokenService) UnlockCooldown(id string) error {
	updates := map[string]interface{}{
		"cooldown_until":       nil,
		"consecutive_failures": 0,
		"last_error":           "",
		"last_error_at":        nil,
	}
	if err := s.db.Model(&database.Token{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return err
	}
	return s.RefreshTokenStateByID(id)
}

func (s *TokenService) GetActiveTokens() ([]database.Token, error) {
	if err := s.RefreshTokenStates(); err != nil {
		return nil, err
	}

	var tokens []database.Token
	if err := s.db.Where("status = ?", TokenStatusActive).Order("updated_at DESC").Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func (s *TokenService) GetAvailablePoolTokens() ([]database.Token, error) {
	if err := s.RefreshTokenStates(); err != nil {
		return nil, err
	}

	var tokens []database.Token
	if err := s.db.Where("status = ? AND pool_status = ?", TokenStatusActive, TokenPoolAvailable).
		Order("updated_at DESC").Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func (s *TokenService) IncrementUsage(id string) error {
	return s.db.Model(&database.Token{}).Where("id = ?", id).
		UpdateColumn("used_requests", gorm.Expr("used_requests + 1")).Error
}

func (s *TokenService) ValidateToken(tokenStr string) (*database.Token, error) {
	if err := s.RefreshTokenStates(); err != nil {
		return nil, err
	}

	var token database.Token
	if err := s.db.Where("token = ?", tokenStr).First(&token).Error; err != nil {
		return nil, errors.New("token not found")
	}
	if token.Status != TokenStatusActive {
		return nil, errors.New("token is not active")
	}
	if token.PoolStatus == TokenPoolDisabled {
		return nil, errors.New("token is disabled")
	}
	if token.PoolStatus == TokenPoolExpired || token.IsExpired() {
		return nil, errors.New("token has expired")
	}
	return &token, nil
}

func (s *TokenService) BatchImport(tokens []database.Token) (int, int, error) {
	success := 0
	failed := 0

	for i := range tokens {
		tokens[i].ID = uuid.New().String()[:20]
		tokens[i].MaxRequests = 0
		now := time.Now()
		tokens[i].Status, tokens[i].PoolStatus = normalizeTokenState(&tokens[i], now)
		if tokens[i].Weight <= 0 {
			tokens[i].Weight = 1
		}
		if err := s.db.Create(&tokens[i]).Error; err != nil {
			logger.Warnf("Failed to import token %s: %v", tokens[i].Name, err)
			failed++
		} else {
			success++
		}
	}

	return success, failed, nil
}

func (s *TokenService) GetStats() (map[string]interface{}, error) {
	if err := s.RefreshTokenStates(); err != nil {
		return nil, err
	}

	var total, active, expired, disabled, available, cooldown int64
	var quotaSynced, lowDailyQuota, lowWeeklyQuota int64
	var totalActiveRequests int64

	s.db.Model(&database.Token{}).Count(&total)
	s.db.Model(&database.Token{}).Where("status = ?", TokenStatusActive).Count(&active)
	s.db.Model(&database.Token{}).Where("status = ?", TokenStatusExpired).Count(&expired)
	s.db.Model(&database.Token{}).Where("status = ?", TokenStatusDisabled).Count(&disabled)
	s.db.Model(&database.Token{}).Where("pool_status = ?", TokenPoolAvailable).Count(&available)
	s.db.Model(&database.Token{}).Where("pool_status = ?", TokenPoolCooldown).Count(&cooldown)
	s.db.Model(&database.Token{}).Where("quota_updated_at IS NOT NULL").Count(&quotaSynced)
	s.db.Model(&database.Token{}).
		Where("quota_updated_at IS NOT NULL AND hide_daily_quota = ? AND daily_quota_remaining_percent <= ?", false, 20).
		Count(&lowDailyQuota)
	s.db.Model(&database.Token{}).
		Where("quota_updated_at IS NOT NULL AND hide_weekly_quota = ? AND weekly_quota_remaining_percent <= ?", false, 20).
		Count(&lowWeeklyQuota)
	s.db.Model(&database.Token{}).Select("COALESCE(SUM(active_requests), 0)").Scan(&totalActiveRequests)

	return map[string]interface{}{
		"total":                 total,
		"active":                active,
		"expired":               expired,
		"disabled":              disabled,
		"available":             available,
		"cooldown":              cooldown,
		"exhausted":             int64(0),
		"quota_synced":          quotaSynced,
		"low_daily_quota":       lowDailyQuota,
		"low_weekly_quota":      lowWeeklyQuota,
		"total_active_requests": totalActiveRequests,
	}, nil
}

func (s *TokenService) AllocateToken(userID uint) (*database.Token, error) {
	tokens, err := s.GetAvailablePoolTokens()
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no available tokens in pool")
	}

	token := &tokens[0]
	now := time.Now()
	token.AllocatedToID = &userID
	token.AllocatedAt = &now
	if err := s.db.Save(token).Error; err != nil {
		return nil, err
	}
	return token, nil
}

func (s *TokenService) GetUserAllocatedToken(userID uint) (*database.Token, error) {
	var token database.Token
	if err := s.db.Where("allocated_to_id = ?", userID).First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (s *TokenService) RefreshTokenStates() error {
	var tokens []database.Token
	if err := s.db.Find(&tokens).Error; err != nil {
		return err
	}

	now := time.Now()
	for i := range tokens {
		status, poolStatus := normalizeTokenState(&tokens[i], now)
		updates := map[string]interface{}{}
		if tokens[i].Status != status {
			updates["status"] = status
		}
		if tokens[i].PoolStatus != poolStatus {
			updates["pool_status"] = poolStatus
		}
		if tokens[i].MaxRequests != 0 {
			updates["max_requests"] = 0
		}
		if isEmptyQuotaSnapshot(&tokens[i]) {
			for key, value := range emptyQuotaSnapshotUpdates() {
				updates[key] = value
			}
		}
		if len(updates) == 0 {
			continue
		}
		if err := s.db.Model(&database.Token{}).Where("id = ?", tokens[i].ID).Updates(updates).Error; err != nil {
			return err
		}
	}

	return nil
}

func (s *TokenService) RefreshTokenStateByID(id string) error {
	var token database.Token
	if err := s.db.First(&token, "id = ?", id).Error; err != nil {
		return err
	}

	status, poolStatus := normalizeTokenState(&token, time.Now())
	updates := map[string]interface{}{}
	if token.Status != status {
		updates["status"] = status
	}
	if token.PoolStatus != poolStatus {
		updates["pool_status"] = poolStatus
	}
	if token.MaxRequests != 0 {
		updates["max_requests"] = 0
	}
	if isEmptyQuotaSnapshot(&token) {
		for key, value := range emptyQuotaSnapshotUpdates() {
			updates[key] = value
		}
	}
	if len(updates) == 0 {
		return nil
	}
	return s.db.Model(&database.Token{}).Where("id = ?", id).Updates(updates).Error
}

type TokenCreateRequest struct {
	Token         string     `json:"token"`
	Name          string     `json:"name"`
	Description   string     `json:"description"`
	TenantAddress string     `json:"tenant_address"`
	ProxyURL      string     `json:"proxy_url"`
	MaxRequests   int        `json:"max_requests"`
	Weight        int        `json:"weight"`
	IsShared      bool       `json:"is_shared"`
	Email         string     `json:"email"`
	ExpiresAt     *time.Time `json:"expires_at"`
}

func (s *TokenService) CreateToken(req *TokenCreateRequest) (*database.Token, error) {
	isShared := req.IsShared
	tenantAddress := strings.TrimSpace(req.TenantAddress)
	if tenantAddress == "" {
		tenantAddress = DefaultWindsurfAPIURL
	}
	var email *string
	if req.Email != "" {
		email = &req.Email
	}
	var proxyURL *string
	if req.ProxyURL != "" {
		proxyURL = &req.ProxyURL
	}
	weight := req.Weight
	if weight <= 0 {
		weight = 1
	}

	token := &database.Token{
		Token:         req.Token,
		Name:          req.Name,
		Description:   req.Description,
		TenantAddress: tenantAddress,
		ProxyURL:      proxyURL,
		MaxRequests:   0,
		Weight:        weight,
		IsShared:      &isShared,
		Email:         email,
		ExpiresAt:     req.ExpiresAt,
	}

	if err := s.Create(token); err != nil {
		return nil, err
	}
	return token, nil
}

func (s *TokenService) BatchImportTokens(reqs []TokenCreateRequest) (int, int, error) {
	success := 0
	failed := 0

	for _, req := range reqs {
		if _, err := s.CreateToken(&req); err != nil {
			logger.Warnf("Failed to import token %s: %v", req.Name, err)
			failed++
		} else {
			success++
		}
	}

	return success, failed, nil
}

func normalizeTokenState(token *database.Token, now time.Time) (string, string) {
	status := token.Status
	if status == "" {
		status = TokenStatusActive
	}

	switch {
	case status == TokenStatusDisabled:
		return TokenStatusDisabled, TokenPoolDisabled
	case token.ExpiresAt != nil && token.ExpiresAt.Before(now):
		return TokenStatusExpired, TokenPoolExpired
	case token.CooldownUntil != nil && token.CooldownUntil.After(now):
		return TokenStatusActive, TokenPoolCooldown
	default:
		return TokenStatusActive, TokenPoolAvailable
	}
}

func normalizeTokenUpdates(updates map[string]interface{}) (map[string]interface{}, error) {
	normalized := make(map[string]interface{}, len(updates))

	for key, value := range updates {
		switch key {
		case "expires_at":
			expiresAt, err := normalizeOptionalTime(value)
			if err != nil {
				return nil, err
			}
			normalized[key] = expiresAt
		case "weight", "used_requests", "active_requests", "consecutive_failures", "last_status_code":
			normalized[key] = normalizeInt(value)
		case "max_requests":
			normalized[key] = 0
		case "status":
			status := strings.TrimSpace(fmt.Sprintf("%v", value))
			if status != "" && status != TokenStatusActive && status != TokenStatusDisabled && status != TokenStatusExpired {
				return nil, fmt.Errorf("invalid token status: %s", status)
			}
			normalized[key] = status
		case "pool_status":
			poolStatus := strings.TrimSpace(fmt.Sprintf("%v", value))
			if poolStatus != "" && poolStatus != TokenPoolAvailable && poolStatus != TokenPoolCooldown && poolStatus != TokenPoolDisabled && poolStatus != TokenPoolExpired && poolStatus != TokenPoolExhausted {
				return nil, fmt.Errorf("invalid pool status: %s", poolStatus)
			}
			normalized[key] = poolStatus
		case "proxy_url":
			str := strings.TrimSpace(fmt.Sprintf("%v", value))
			if str == "" {
				normalized[key] = nil
			} else {
				normalized[key] = str
			}
		case "cooldown_until", "last_used_at", "last_error_at":
			tm, err := normalizeOptionalTime(value)
			if err != nil {
				return nil, err
			}
			normalized[key] = tm
		default:
			normalized[key] = value
		}
	}

	return normalized, nil
}

func normalizeOptionalTime(value interface{}) (*time.Time, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case time.Time:
		return &v, nil
	case *time.Time:
		return v, nil
	case string:
		if strings.TrimSpace(v) == "" {
			return nil, nil
		}
		for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04"} {
			parsed, err := time.ParseInLocation(layout, v, time.Local)
			if err == nil {
				return &parsed, nil
			}
		}
		return nil, fmt.Errorf("invalid time value: %s", v)
	default:
		return nil, fmt.Errorf("invalid time value type: %T", value)
	}
}

func normalizeInt(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float32:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}
