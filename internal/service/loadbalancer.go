package service

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"

	"windsurf-gateway/internal/database"

	"gorm.io/gorm"
)

type LoadBalancerStrategy string

const (
	StrategyRoundRobin   LoadBalancerStrategy = "round_robin"
	StrategyRandom       LoadBalancerStrategy = "random"
	StrategyWeighted     LoadBalancerStrategy = "weighted"
	StrategyLeastConn    LoadBalancerStrategy = "least_connections"
	defaultAuthCooldown  time.Duration        = 30 * time.Minute
	defaultQuotaCooldown time.Duration        = 10 * time.Minute
	defaultErrorCooldown time.Duration        = 2 * time.Minute
)

type LoadBalancerService struct {
	db           *gorm.DB
	cache        *CacheService
	token        *TokenService
	systemConfig *SystemConfigService
	mu           sync.Mutex
	rrIndex      int
	rng          *rand.Rand
}

type TokenRequestOutcome struct {
	StatusCode      int
	FailureCategory string
	ErrorMessage    string
	Success         bool
	Penalize        bool
	Cooldown        time.Duration
}

func NewLoadBalancerService(db *gorm.DB, cache *CacheService, token *TokenService, systemConfig *SystemConfigService) *LoadBalancerService {
	return &LoadBalancerService{
		db:           db,
		cache:        cache,
		token:        token,
		systemConfig: systemConfig,
		rng:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *LoadBalancerService) SelectToken(ctx context.Context) (*database.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.token.RefreshTokenStates(); err != nil {
		return nil, err
	}

	tokens, err := s.token.GetActiveTokens()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	candidates := make([]*database.Token, 0, len(tokens))
	for i := range tokens {
		if tokens[i].IsReadyForScheduling(now) {
			candidates = append(candidates, &tokens[i])
		}
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available backend tokens")
	}

	strategy := s.GetStrategy()
	selected, err := s.pickToken(strategy, candidates)
	if err != nil {
		return nil, err
	}

	if err := s.acquireToken(selected.ID, now); err != nil {
		return nil, err
	}
	selected.ActiveRequests++
	selected.LastUsedAt = &now
	return selected, nil
}

func (s *LoadBalancerService) SelectAnyToken(ctx context.Context) (*database.Token, error) {
	return s.SelectToken(ctx)
}

func (s *LoadBalancerService) AcquireSpecificToken(ctx context.Context, tokenID string) (*database.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_ = ctx
	if err := s.token.RefreshTokenStateByID(tokenID); err != nil {
		return nil, err
	}

	token, err := s.token.GetByID(tokenID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if !token.IsReadyForScheduling(now) {
		return nil, fmt.Errorf("backend token %s is unavailable", tokenID)
	}
	if err := s.acquireToken(token.ID, now); err != nil {
		return nil, err
	}
	token.ActiveRequests++
	token.LastUsedAt = &now
	return token, nil
}

func (s *LoadBalancerService) CompleteRequest(tokenID string, outcome TokenRequestOutcome) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	updates := map[string]interface{}{
		"active_requests":  gorm.Expr("CASE WHEN active_requests > 0 THEN active_requests - 1 ELSE 0 END"),
		"used_requests":    gorm.Expr("used_requests + 1"),
		"last_used_at":     now,
		"last_status_code": outcome.StatusCode,
	}

	switch {
	case outcome.Penalize:
		updates["consecutive_failures"] = gorm.Expr("consecutive_failures + 1")
		updates["total_failures"] = gorm.Expr("total_failures + 1")
		updates["last_error"] = truncateString(outcome.ErrorMessage, 1000)
		updates["last_error_at"] = now
		if outcome.Cooldown <= 0 {
			outcome.Cooldown = defaultErrorCooldown
		}
		cooldownUntil := now.Add(outcome.Cooldown)
		updates["cooldown_until"] = cooldownUntil
		updates["pool_status"] = TokenPoolCooldown
	case outcome.Success:
		updates["consecutive_failures"] = 0
		updates["total_successes"] = gorm.Expr("total_successes + 1")
		updates["last_error"] = ""
		updates["last_error_at"] = nil
		updates["cooldown_until"] = nil
		updates["pool_status"] = TokenPoolAvailable
	default:
		updates["consecutive_failures"] = 0
		updates["last_error"] = ""
		updates["last_error_at"] = nil
		updates["cooldown_until"] = nil
		updates["pool_status"] = TokenPoolAvailable
	}

	if err := s.db.Model(&database.Token{}).Where("id = ?", tokenID).Updates(updates).Error; err != nil {
		return err
	}
	return s.token.RefreshTokenStateByID(tokenID)
}

func (s *LoadBalancerService) GetStrategy() LoadBalancerStrategy {
	if s.systemConfig == nil {
		return StrategyRoundRobin
	}
	value, err := s.systemConfig.Get("load_balancer_strategy")
	if err != nil {
		return StrategyRoundRobin
	}
	switch LoadBalancerStrategy(value) {
	case StrategyRoundRobin, StrategyRandom, StrategyWeighted, StrategyLeastConn:
		return LoadBalancerStrategy(value)
	default:
		return StrategyRoundRobin
	}
}

func (s *LoadBalancerService) SetStrategy(strategy LoadBalancerStrategy) error {
	switch strategy {
	case StrategyRoundRobin, StrategyRandom, StrategyWeighted, StrategyLeastConn:
		if s.systemConfig == nil {
			return fmt.Errorf("system config service unavailable")
		}
		return s.systemConfig.Set("load_balancer_strategy", string(strategy))
	default:
		return fmt.Errorf("unsupported load balancer strategy: %s", strategy)
	}
}

func (s *LoadBalancerService) GetStats() (map[string]interface{}, error) {
	if err := s.token.RefreshTokenStates(); err != nil {
		return nil, err
	}

	stats, err := s.token.GetStats()
	if err != nil {
		return nil, err
	}
	stats["strategy"] = string(s.GetStrategy())
	stats["round_robin_index"] = s.rrIndex
	return stats, nil
}

func (s *LoadBalancerService) acquireToken(tokenID string, now time.Time) error {
	return s.db.Model(&database.Token{}).Where("id = ?", tokenID).Updates(map[string]interface{}{
		"active_requests": gorm.Expr("active_requests + 1"),
		"last_used_at":    now,
	}).Error
}

func (s *LoadBalancerService) pickToken(strategy LoadBalancerStrategy, candidates []*database.Token) (*database.Token, error) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].ActiveRequests == candidates[j].ActiveRequests {
			return candidates[i].ID < candidates[j].ID
		}
		return candidates[i].ActiveRequests < candidates[j].ActiveRequests
	})

	switch strategy {
	case StrategyRandom:
		return s.pickRandom(candidates)
	case StrategyWeighted:
		return s.pickWeighted(candidates)
	case StrategyLeastConn:
		return candidates[0], nil
	default:
		return s.pickRoundRobin(candidates), nil
	}
}

func (s *LoadBalancerService) pickRoundRobin(candidates []*database.Token) *database.Token {
	if len(candidates) == 0 {
		return nil
	}
	selected := candidates[s.rrIndex%len(candidates)]
	s.rrIndex = (s.rrIndex + 1) % len(candidates)
	return selected
}

func (s *LoadBalancerService) pickRandom(candidates []*database.Token) (*database.Token, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available backend tokens")
	}
	return candidates[s.rng.Intn(len(candidates))], nil
}

func (s *LoadBalancerService) pickWeighted(candidates []*database.Token) (*database.Token, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no available backend tokens")
	}

	totalWeight := 0
	for _, candidate := range candidates {
		weight := candidate.Weight
		if weight <= 0 {
			weight = 1
		}
		totalWeight += weight
	}
	if totalWeight <= 0 {
		return s.pickRandom(candidates)
	}

	roll := s.rng.Intn(totalWeight)
	current := 0
	for _, candidate := range candidates {
		weight := candidate.Weight
		if weight <= 0 {
			weight = 1
		}
		current += weight
		if roll < current {
			return candidate, nil
		}
	}
	return candidates[len(candidates)-1], nil
}

func truncateString(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}
