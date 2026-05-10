package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"windsurf-gateway/internal/database"
	"windsurf-gateway/internal/logger"
	"windsurf-gateway/internal/proxy"
	"windsurf-gateway/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProxyHandler struct {
	proxy    *proxy.ProxyService
	services *service.Services
}

const (
	gatewayClientIDHeader         = "X-Windsurf-Gateway-Client-Id"
	legacyGatewayPlaceholderToken = "sk-ws-01-gateway-placeholder"
)

func NewProxyHandler(proxySvc *proxy.ProxyService, services *service.Services) *ProxyHandler {
	return &ProxyHandler{proxy: proxySvc, services: services}
}

func (h *ProxyHandler) ForwardWithUserToken(c *gin.Context) {
	requestID := uuid.NewString()
	c.Writer.Header().Set("X-Request-ID", requestID)

	authRequired := h.services.SystemConfig.GetBool("require_user_auth_proxy", false)
	userToken := extractGatewayUserToken(c.GetHeader("Authorization"))
	if authRequired && userToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "gateway user token required"})
		return
	}

	var user *database.User
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
		allowed, remaining, rateErr := h.services.Cache.GetRateLimit(rateLimitKey, user.RateLimitPerMinute, time.Minute)
		if rateErr != nil || !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded", "remaining": remaining})
			return
		}
	}

	assignmentKey := buildAssignmentKey(c, user)
	selectionPolicy := service.TokenSelectionPolicy{
		RequireWindsurfQuota: user == nil || !user.UnlimitedAccess,
	}
	backendToken, err := h.services.LoadBalancer.SelectTokenForAssignmentWithPolicy(c.Request.Context(), assignmentKey, selectionPolicy)
	if err != nil {
		logger.Errorf("Failed to select backend token: %v", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no available backend tokens"})
		return
	}

	h.forwardRequest(c, requestID, user, backendToken)
}

func (h *ProxyHandler) ForwardWithSystemToken(c *gin.Context) {
	requestID := uuid.NewString()
	c.Writer.Header().Set("X-Request-ID", requestID)

	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
		return
	}

	tokenStr := extractBearerToken(authHeader)
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
		return
	}

	validatedToken, err := h.services.Token.ValidateToken(tokenStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid system token"})
		return
	}

	backendToken, err := h.services.LoadBalancer.AcquireSpecificToken(c.Request.Context(), validatedToken.ID)
	if err != nil {
		logger.Errorf("Failed to acquire specific backend token %s: %v", validatedToken.ID, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "backend token unavailable"})
		return
	}

	h.forwardRequest(c, requestID, nil, backendToken)
}

func (h *ProxyHandler) forwardRequest(c *gin.Context, requestID string, user *database.User, backendToken *database.Token) {
	body, readErr := io.ReadAll(c.Request.Body)
	if readErr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	requestPath := buildRequestPath(c)
	proxyReq := &proxy.ProxyRequest{
		Token:               backendToken,
		Method:              c.Request.Method,
		Path:                requestPath,
		Headers:             c.Request.Header,
		Body:                body,
		CaptureResponseBody: shouldCaptureUpstreamResponseBody(requestPath),
		ContentType:         c.ContentType(),
		ClientIP:            proxy.GetClientIP(c.Request),
		UserAgent:           c.Request.UserAgent(),
		TenantAddress:       backendToken.TenantAddress,
	}

	proxyResp, proxyErr := h.proxy.ForwardStream(c.Request.Context(), proxyReq, c.Writer)
	outcome := classifyProxyOutcome(requestPath, proxyResp, proxyErr)
	if err := h.services.LoadBalancer.CompleteRequest(backendToken.ID, outcome); err != nil {
		logger.Warnf("Failed to update backend token %s state: %v", backendToken.ID, err)
	}
	if shouldSyncTokenQuota(requestPath, proxyResp, proxyErr) {
		tokenID := backendToken.ID
		headers := proxyResp.Headers.Clone()
		bodyCopy := append([]byte(nil), proxyResp.CapturedBody...)
		go func() {
			if err := h.services.Token.UpdateQuotaFromGetUserStatusResponse(tokenID, headers, bodyCopy); err != nil {
				logger.Warnf("Failed to update token %s quota snapshot: %v", tokenID, err)
			}
		}()
	}

	if user != nil {
		user.IncrementUsage()
		if err := h.services.UserAuth.UpdateUser(user.ID, map[string]interface{}{"used_requests": user.UsedRequests}); err != nil {
			logger.Warnf("Failed to update user %d usage: %v", user.ID, err)
		}
	}

	logRecord := buildRequestLog(requestID, user, backendToken, proxyReq, proxyResp, outcome)
	if err := h.services.RequestRecord.Create(logRecord); err != nil {
		logger.Warnf("Failed to persist request log %s: %v", requestID, err)
	}

	if proxyErr != nil {
		logger.Errorf("Proxy error for request %s: %v", requestID, proxyErr)
		if !c.Writer.Written() {
			c.JSON(http.StatusBadGateway, gin.H{"error": "proxy request failed", "request_id": requestID})
		}
	}
}

func buildRequestPath(c *gin.Context) string {
	path := c.Param("path")
	if path == "" {
		path = c.Request.URL.Path
	}
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if rawQuery := c.Request.URL.RawQuery; rawQuery != "" {
		path += "?" + rawQuery
	}
	return path
}

func extractBearerToken(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return strings.TrimSpace(value[7:])
	}
	return value
}

func extractGatewayUserToken(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	if !strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return ""
	}

	token := strings.TrimSpace(value[7:])
	if !strings.HasPrefix(strings.ToLower(token), "ws-") {
		return ""
	}
	return token
}

func buildAssignmentKey(c *gin.Context, user *database.User) string {
	if user != nil {
		return fmt.Sprintf("user:%d", user.ID)
	}
	if gatewayClientID := extractGatewayClientID(c); gatewayClientID != "" {
		return buildHashedAnonymousKey("client:" + gatewayClientID)
	}
	return buildAnonymousAssignmentKey(c.GetHeader("Authorization"), proxy.GetClientIP(c.Request))
}

func extractGatewayClientID(c *gin.Context) string {
	for _, header := range []string{gatewayClientIDHeader, "X-Request-Session-Id"} {
		if value := strings.TrimSpace(c.GetHeader(header)); value != "" {
			return value
		}
	}
	return ""
}

func buildAnonymousAssignmentKey(authHeader, clientIP string) string {
	authHeader = strings.TrimSpace(authHeader)
	clientIP = strings.TrimSpace(clientIP)
	if clientIP == "" {
		clientIP = "unknown"
	}

	normalizedAuth := strings.ToLower(authHeader)
	switch {
	case normalizedAuth == "":
		return buildHashedAnonymousKey("ip:" + clientIP)
	case strings.Contains(normalizedAuth, legacyGatewayPlaceholderToken):
		return buildHashedAnonymousKey("legacy-auth:" + normalizedAuth + "|ip:" + clientIP)
	default:
		return buildHashedAnonymousKey("auth:" + normalizedAuth)
	}
}

func buildHashedAnonymousKey(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "anon:" + hex.EncodeToString(sum[:16])
}

func classifyProxyOutcome(requestPath string, resp *proxy.ProxyResponse, err error) service.TokenRequestOutcome {
	outcome := service.TokenRequestOutcome{}
	if resp != nil {
		outcome.StatusCode = resp.StatusCode
	}

	if err != nil {
		message := err.Error()
		lower := strings.ToLower(message)
		outcome.ErrorMessage = message

		switch {
		case strings.Contains(lower, "broken pipe"), strings.Contains(lower, "reset by peer"), strings.Contains(lower, "write response"), strings.Contains(lower, "context canceled"):
			outcome.FailureCategory = "client_disconnected"
			outcome.Success = false
			outcome.Penalize = false
		case strings.Contains(lower, "timeout"):
			outcome.FailureCategory = "transport_timeout"
			outcome.Penalize = true
			outcome.Cooldown = 2 * time.Minute
		default:
			outcome.FailureCategory = "transport_error"
			outcome.Penalize = true
			outcome.Cooldown = 5 * time.Minute
		}
		if outcome.StatusCode == 0 {
			outcome.StatusCode = http.StatusBadGateway
		}
		return outcome
	}

	if resp == nil {
		outcome.StatusCode = http.StatusBadGateway
		outcome.FailureCategory = "empty_response"
		outcome.Penalize = true
		outcome.Cooldown = 2 * time.Minute
		outcome.ErrorMessage = "empty proxy response"
		return outcome
	}

	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		if shouldIgnoreAuthCooldown(requestPath) {
			outcome.FailureCategory = "optional_upstream_401"
			outcome.ErrorMessage = coalesceProxyError(resp.ErrorMessage, "upstream returned 401")
			outcome.Penalize = false
			outcome.Success = false
			break
		}
		outcome.FailureCategory = "upstream_401"
		outcome.ErrorMessage = coalesceProxyError(resp.ErrorMessage, "upstream returned 401")
		outcome.Penalize = true
		outcome.Cooldown = 30 * time.Minute
	case resp.StatusCode == http.StatusForbidden:
		outcome.FailureCategory = "upstream_403"
		outcome.ErrorMessage = coalesceProxyError(resp.ErrorMessage, "upstream returned 403")
		outcome.Penalize = true
		outcome.Cooldown = 30 * time.Minute
	case resp.StatusCode == http.StatusTooManyRequests:
		outcome.FailureCategory = "upstream_429"
		outcome.ErrorMessage = coalesceProxyError(resp.ErrorMessage, "upstream returned 429")
		outcome.Penalize = true
		outcome.Cooldown = 10 * time.Minute
	case resp.StatusCode >= 500:
		outcome.FailureCategory = fmt.Sprintf("upstream_%d", resp.StatusCode)
		outcome.ErrorMessage = coalesceProxyError(resp.ErrorMessage, fmt.Sprintf("upstream returned %d", resp.StatusCode))
		outcome.Penalize = true
		outcome.Cooldown = 2 * time.Minute
	case resp.StatusCode >= 400:
		outcome.FailureCategory = fmt.Sprintf("client_%d", resp.StatusCode)
		outcome.ErrorMessage = truncateErrorMessage(resp.BodySnippet)
		outcome.Success = false
		outcome.Penalize = false
	default:
		outcome.Success = true
	}
	return outcome
}

func shouldCaptureUpstreamResponseBody(requestPath string) bool {
	return stripQuery(requestPath) == "/exa.seat_management_pb.SeatManagementService/GetUserStatus"
}

func shouldSyncTokenQuota(requestPath string, resp *proxy.ProxyResponse, err error) bool {
	if err != nil || resp == nil || resp.StatusCode != http.StatusOK || len(resp.CapturedBody) == 0 {
		return false
	}
	return shouldCaptureUpstreamResponseBody(requestPath)
}

func shouldIgnoreAuthCooldown(requestPath string) bool {
	switch stripQuery(requestPath) {
	case "/exa.api_server_pb.ApiServerService/CheckChatCapacity",
		"/exa.api_server_pb.ApiServerService/CheckUserMessageRateLimit",
		"/exa.api_server_pb.ApiServerService/GetDefaultWorkflowTemplates",
		"/exa.seat_management_pb.SeatManagementService/GetProfileData":
		return true
	default:
		return false
	}
}

func stripQuery(path string) string {
	if idx := strings.Index(path, "?"); idx >= 0 {
		return path[:idx]
	}
	return path
}

func buildRequestLog(requestID string, user *database.User, backendToken *database.Token, req *proxy.ProxyRequest, resp *proxy.ProxyResponse, outcome service.TokenRequestOutcome) *database.RequestLog {
	statusCode := http.StatusBadGateway
	upstreamStatusCode := 0
	if resp != nil && resp.StatusCode > 0 {
		statusCode = resp.StatusCode
		upstreamStatusCode = resp.StatusCode
	} else if outcome.StatusCode > 0 {
		statusCode = outcome.StatusCode
	}

	logRecord := &database.RequestLog{
		TokenID:            backendToken.ID,
		TokenName:          backendToken.Name,
		RequestID:          requestID,
		Method:             req.Method,
		Path:               req.Path,
		UserAgent:          req.UserAgent,
		ClientIP:           req.ClientIP,
		TenantAddress:      req.TenantAddress,
		StatusCode:         statusCode,
		UpstreamStatusCode: upstreamStatusCode,
		RequestSize:        int64(len(req.Body)),
		FailureCategory:    outcome.FailureCategory,
		ErrorMessage:       truncateErrorMessage(coalesceProxyError(outcome.ErrorMessage, respError(resp))),
	}
	if resp != nil {
		logRecord.ResponseSize = resp.Size
		logRecord.Latency = resp.Latency.Microseconds()
	}
	if user != nil {
		logRecord.UserID = &user.ID
		logRecord.Username = user.Username
	}
	return logRecord
}

func coalesceProxyError(value, fallback string) string {
	if strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func respError(resp *proxy.ProxyResponse) string {
	if resp == nil {
		return ""
	}
	if resp.ErrorMessage != "" {
		return resp.ErrorMessage
	}
	return resp.BodySnippet
}

func truncateErrorMessage(value string) string {
	value = strings.TrimSpace(sanitizeUTF8(value))
	if len(value) <= 1000 {
		return value
	}
	return value[:1000]
}

func sanitizeUTF8(value string) string {
	if value == "" {
		return value
	}
	if utf8.ValidString(value) {
		return value
	}
	return strings.ToValidUTF8(value, "")
}
