package service

import (
	"windsurf-gateway/internal/config"
	"windsurf-gateway/internal/database"

	"gorm.io/gorm"
)

type Services struct {
	Auth            *AuthService
	UserAuth        *UserAuthService
	Token           *TokenService
	Cache           *CacheService
	Stats           *StatsService
	LoadBalancer    *LoadBalancerService
	InvitationCode  *InvitationCodeService
	RequestRecord   *RequestRecordService
	Plugin          *PluginService
	SystemConfig    *SystemConfigService
}

func NewServices(db *gorm.DB, redis *database.RedisClient, cfg *config.Config) *Services {
	cacheService := NewCacheService(redis)
	authService := NewAuthService(db, cfg.Security.JWTSecret)

	userAuthCfg := &UserAuthConfig{
		PasswordMinLength: cfg.UserAuth.PasswordMinLength,
		PasswordMaxLength: cfg.UserAuth.PasswordMaxLength,
	}
	userAuthService := NewUserAuthService(db, cfg.Security.JWTSecret, userAuthCfg)

	tokenService := NewTokenService(db, cacheService)
	statsService := NewStatsService(db, cacheService)
	loadBalancerService := NewLoadBalancerService(db, cacheService, tokenService)
	invitationCodeService := NewInvitationCodeService(db)
	requestRecordService := NewRequestRecordService(db)
	pluginService := NewPluginService(db)
	systemConfigService := NewSystemConfigService(db)

	userAuthService.SetInvitationCodeService(invitationCodeService)

	return &Services{
		Auth:           authService,
		UserAuth:       userAuthService,
		Token:          tokenService,
		Cache:          cacheService,
		Stats:          statsService,
		LoadBalancer:   loadBalancerService,
		InvitationCode: invitationCodeService,
		RequestRecord:  requestRecordService,
		Plugin:         pluginService,
		SystemConfig:   systemConfigService,
	}
}
