package handler

import (
	"windsurf-gateway/internal/config"
	"windsurf-gateway/internal/proxy"
	"windsurf-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	Auth           *AuthHandler
	UserAuth       *UserAuthHandler
	Token          *TokenHandler
	Stats          *StatsHandler
	Proxy          *ProxyHandler
	InvitationCode *InvitationCodeHandler
	RequestRecord  *RequestRecordHandler
	Plugin         *PluginHandler
	SystemConfig   *SystemConfigHandler
	System         *SystemHandler
}

func NewHandlers(services *service.Services, cfg *config.Config) *Handlers {
	proxyService := proxy.NewProxyService(cfg)

	return &Handlers{
		Auth:           NewAuthHandler(services.Auth),
		UserAuth:       NewUserAuthHandler(services.UserAuth),
		Token:          NewTokenHandler(services.Token),
		Stats:          NewStatsHandler(services.Stats),
		Proxy:          NewProxyHandler(proxyService, services),
		InvitationCode: NewInvitationCodeHandler(services.InvitationCode),
		RequestRecord:  NewRequestRecordHandler(services.RequestRecord),
		Plugin:         NewPluginHandler(services.Plugin),
		SystemConfig:   NewSystemConfigHandler(services.SystemConfig),
		System:         NewSystemHandler(),
	}
}

func getPageParams(c *gin.Context) (int, int) {
	page := 1
	pageSize := 20
	if p, ok := c.GetQuery("page"); ok {
		if v, err := parseInt(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps, ok := c.GetQuery("page_size"); ok {
		if v, err := parseInt(ps); err == nil && v > 0 && v <= 100 {
			pageSize = v
		}
	}
	return page, pageSize
}

func parseInt(s string) (int, error) {
	var n int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
