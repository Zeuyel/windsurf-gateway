package service

import (
	"math/rand"
	"sync"
	"time"

	"windsurf-gateway/internal/database"
	"windsurf-gateway/internal/logger"

	"gorm.io/gorm"
)

type LoadBalancerService struct {
	db    *gorm.DB
	cache *CacheService
	token *TokenService
	mu    sync.RWMutex
}

func NewLoadBalancerService(db *gorm.DB, cache *CacheService, token *TokenService) *LoadBalancerService {
	return &LoadBalancerService{
		db:    db,
		cache: cache,
		token: token,
	}
}

func (s *LoadBalancerService) SelectToken(userID uint) (*database.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	allocated, err := s.token.GetUserAllocatedToken(userID)
	if err == nil && allocated != nil {
		if s.isTokenValid(allocated) {
			return allocated, nil
		}
		s.releaseToken(allocated)
	}

	token, err := s.token.AllocateToken(userID)
	if err != nil {
		logger.Warnf("No available tokens for user %d, trying round-robin", userID)
		return s.selectRoundRobin()
	}

	logger.Infof("Allocated token %s to user %d", token.ID, userID)
	return token, nil
}

func (s *LoadBalancerService) isTokenValid(token *database.Token) bool {
	if token.Status != "active" {
		return false
	}
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return false
	}
	if token.MaxRequests > 0 && token.UsedRequests >= token.MaxRequests {
		return false
	}
	return true
}

func (s *LoadBalancerService) releaseToken(token *database.Token) {
	token.PoolStatus = "available"
	token.AllocatedToID = nil
	token.AllocatedAt = nil
	s.db.Save(token)
}

func (s *LoadBalancerService) selectRoundRobin() (*database.Token, error) {
	tokens, err := s.token.GetActiveTokens()
	if err != nil {
		return nil, err
	}

	validTokens := make([]*database.Token, 0)
	for i := range tokens {
		if s.isTokenValid(&tokens[i]) {
			validTokens = append(validTokens, &tokens[i])
		}
	}

	if len(validTokens) == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	idx := rand.Intn(len(validTokens))
	return validTokens[idx], nil
}

func (s *LoadBalancerService) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	total, _ := s.token.GetStats()
	available, _ := s.token.GetAvailablePoolTokens()

	return map[string]interface{}{
		"total":     total,
		"available": len(available),
	}
}
