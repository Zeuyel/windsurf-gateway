package service

import (
	"errors"
	"fmt"
	"time"

	"windsurf-gateway/internal/database"
	"windsurf-gateway/internal/logger"

	"github.com/google/uuid"
	"gorm.io/gorm"
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
	token.Status = "active"
	token.PoolStatus = "available"
	return s.db.Create(token).Error
}

func (s *TokenService) GetByID(id string) (*database.Token, error) {
	var token database.Token
	if err := s.db.First(&token, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

func (s *TokenService) List(page, pageSize int, status string) ([]database.Token, int64, error) {
	var tokens []database.Token
	var total int64

	query := s.db.Model(&database.Token{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	query.Count(&total)

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&tokens).Error; err != nil {
		return nil, 0, err
	}

	return tokens, total, nil
}

func (s *TokenService) Update(id string, updates map[string]interface{}) error {
	return s.db.Model(&database.Token{}).Where("id = ?", id).Updates(updates).Error
}

func (s *TokenService) Delete(id string) error {
	return s.db.Where("id = ?", id).Delete(&database.Token{}).Error
}

func (s *TokenService) GetActiveTokens() ([]database.Token, error) {
	var tokens []database.Token
	if err := s.db.Where("status = ?", "active").Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func (s *TokenService) GetAvailablePoolTokens() ([]database.Token, error) {
	var tokens []database.Token
	if err := s.db.Where("status = ? AND pool_status = ?", "active", "available").Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func (s *TokenService) IncrementUsage(id string) error {
	return s.db.Model(&database.Token{}).Where("id = ?", id).
		UpdateColumn("used_requests", gorm.Expr("used_requests + 1")).Error
}

func (s *TokenService) ValidateToken(tokenStr string) (*database.Token, error) {
	var token database.Token
	if err := s.db.Where("token = ?", tokenStr).First(&token).Error; err != nil {
		return nil, errors.New("token not found")
	}
	if token.Status != "active" {
		return nil, errors.New("token is not active")
	}
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return nil, errors.New("token has expired")
	}
	return &token, nil
}

func (s *TokenService) BatchImport(tokens []database.Token) (int, int, error) {
	success := 0
	failed := 0

	for _, token := range tokens {
		token.ID = uuid.New().String()[:20]
		token.Status = "active"
		token.PoolStatus = "available"
		if err := s.db.Create(&token).Error; err != nil {
			logger.Warnf("Failed to import token %s: %v", token.Name, err)
			failed++
		} else {
			success++
		}
	}

	return success, failed, nil
}

func (s *TokenService) GetStats() (map[string]interface{}, error) {
	var total, active, expired, disabled int64

	s.db.Model(&database.Token{}).Count(&total)
	s.db.Model(&database.Token{}).Where("status = ?", "active").Count(&active)
	s.db.Model(&database.Token{}).Where("status = ?", "expired").Count(&expired)
	s.db.Model(&database.Token{}).Where("status = ?", "disabled").Count(&disabled)

	return map[string]interface{}{
		"total":    total,
		"active":   active,
		"expired":  expired,
		"disabled": disabled,
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
	token.PoolStatus = "allocated"
	token.AllocatedToID = &userID
	token.AllocatedAt = &now

	if err := s.db.Save(token).Error; err != nil {
		return nil, err
	}

	return token, nil
}

func (s *TokenService) GetUserAllocatedToken(userID uint) (*database.Token, error) {
	var token database.Token
	if err := s.db.Where("allocated_to_id = ? AND pool_status = ?", userID, "allocated").
		First(&token).Error; err != nil {
		return nil, err
	}
	return &token, nil
}

type TokenCreateRequest struct {
	Token         string `json:"token"`
	Name          string `json:"name"`
	Description   string `json:"description"`
	TenantAddress string `json:"tenant_address"`
	ProxyURL      string `json:"proxy_url"`
	MaxRequests   int    `json:"max_requests"`
	IsShared      bool   `json:"is_shared"`
	Email         string `json:"email"`
}

func (s *TokenService) CreateToken(req *TokenCreateRequest) (*database.Token, error) {
	isShared := req.IsShared
	var email *string
	if req.Email != "" {
		email = &req.Email
	}
	var proxyURL *string
	if req.ProxyURL != "" {
		proxyURL = &req.ProxyURL
	}

	token := &database.Token{
		Token:         req.Token,
		Name:          req.Name,
		Description:   req.Description,
		TenantAddress: req.TenantAddress,
		ProxyURL:      proxyURL,
		MaxRequests:   req.MaxRequests,
		IsShared:      &isShared,
		Email:         email,
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
