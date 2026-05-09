package handler

import (
	"io"
	"net/http"
	"strconv"
	"time"

	"windsurf-gateway/internal/database"
	"windsurf-gateway/internal/logger"
	"windsurf-gateway/internal/proxy"
	"windsurf-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type ProxyHandler struct {
	proxy    *proxy.ProxyService
	services *service.Services
}

func NewProxyHandler(proxySvc *proxy.ProxyService, services *service.Services) *ProxyHandler {
	return &ProxyHandler{proxy: proxySvc, services: services}
}

func (h *ProxyHandler) ForwardWithUserToken(c *gin.Context) {
	startTime := time.Now()

	authHeader := c.GetHeader("Authorization")
	var userToken string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		userToken = authHeader[7:]
	} else {
		userToken = authHeader
	}

	var user *database.User
	var token *database.Token
	var err error
	if userToken != "" {
		user, err = h.services.UserAuth.GetUserByToken(userToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user token"})
			return
		}
		if !user.CanMakeRequest() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "quota exceeded or account disabled"})
			return
		}
		rateLimitKey := "ratelimit:user:" + strconv.FormatUint(uint64(user.ID), 10)
		allowed, remaining, err := h.services.Cache.GetRateLimit(rateLimitKey, user.RateLimitPerMinute, time.Minute)
		if err != nil || !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded", "remaining": remaining})
			return
		}
		token, err = h.services.LoadBalancer.SelectToken(user.ID)
	} else {
		token, err = h.services.LoadBalancer.SelectAnyToken()
	}
	if err != nil {
		logger.Errorf("Failed to select backend token: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no available backend tokens"})
		return
	}

	body, _ := io.ReadAll(c.Request.Body)
	requestPath := c.Param("path")
	if requestPath == "" {
		requestPath = c.Request.URL.RequestURI()
	}

	proxyReq := &proxy.ProxyRequest{
		Token:         token,
		Method:        c.Request.Method,
		Path:          requestPath,
		Headers:       c.Request.Header,
		Body:          body,
		ClientIP:      proxy.GetClientIP(c.Request),
		UserAgent:     c.Request.UserAgent(),
		TenantAddress: token.TenantAddress,
	}

	captured, err := h.proxy.ForwardStream(c.Request.Context(), proxyReq, c.Writer)
	latency := time.Since(startTime)

	if user != nil {
		user.IncrementUsage()
		h.services.UserAuth.UpdateUser(user.ID, map[string]interface{}{
			"used_requests": user.UsedRequests,
		})
	}
	h.services.Token.IncrementUsage(token.ID)

	statusCode := http.StatusOK
	if err != nil {
		statusCode = http.StatusBadGateway
	}

	record := &database.RequestLog{
		TokenID:       token.ID,
		Method:        c.Request.Method,
		Path:          requestPath,
		UserAgent:     c.Request.UserAgent(),
		ClientIP:      proxy.GetClientIP(c.Request),
		TenantAddress: token.TenantAddress,
		StatusCode:    statusCode,
		RequestSize:   int64(len(body)),
		ResponseSize:  int64(len(captured)),
		Latency:       latency.Microseconds(),
	}
	if user != nil {
		record.UserID = &user.ID
	}
	h.services.RequestRecord.Create(record)

	if err != nil {
		logger.Errorf("Proxy error: %v", err)
	}
}

func (h *ProxyHandler) ForwardWithSystemToken(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
		return
	}

	var tokenStr string
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		tokenStr = authHeader[7:]
	} else {
		tokenStr = authHeader
	}

	token, err := h.services.Token.ValidateToken(tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid system token"})
		return
	}

	body, _ := io.ReadAll(c.Request.Body)

	proxyReq := &proxy.ProxyRequest{
		Token:         token,
		Method:        c.Request.Method,
		Path:          c.Param("path"),
		Headers:       c.Request.Header,
		Body:          body,
		ClientIP:      proxy.GetClientIP(c.Request),
		UserAgent:     c.Request.UserAgent(),
		TenantAddress: token.TenantAddress,
	}

	_, err = h.proxy.ForwardStream(c.Request.Context(), proxyReq, c.Writer)
	if err != nil {
		logger.Errorf("System proxy error: %v", err)
	}
}
