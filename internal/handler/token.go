package handler

import (
	"time"

	"windsurf-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type TokenHandler struct {
	svc         *service.TokenService
	smartImport *service.SmartTokenImportService
}

func NewTokenHandler(svc *service.TokenService, smartImport *service.SmartTokenImportService) *TokenHandler {
	return &TokenHandler{svc: svc, smartImport: smartImport}
}

func (h *TokenHandler) List(c *gin.Context) {
	page, pageSize := getPageParams(c)
	status := c.Query("status")
	poolStatus := c.Query("pool_status")
	tokens, total, err := h.svc.List(page, pageSize, status, poolStatus)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, gin.H{"list": tokens, "total": total, "page": page, "page_size": pageSize})
}

func (h *TokenHandler) Create(c *gin.Context) {
	var req struct {
		Token         string `json:"token" binding:"required"`
		Name          string `json:"name" binding:"required"`
		Description   string `json:"description"`
		TenantAddress string `json:"tenant_address" binding:"required"`
		ProxyURL      string `json:"proxy_url"`
		MaxRequests   int    `json:"max_requests"`
		Weight        int    `json:"weight"`
		IsShared      bool   `json:"is_shared"`
		Email         string `json:"email"`
		ExpiresAt     string `json:"expires_at"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request: "+err.Error())
		return
	}

	var expiresAt *time.Time
	if req.ExpiresAt != "" {
		parsed, err := time.ParseInLocation("2006-01-02 15:04:05", req.ExpiresAt, time.Local)
		if err != nil {
			parsed, err = time.Parse(time.RFC3339, req.ExpiresAt)
			if err != nil {
				Error(c, 400, "invalid expires_at format")
				return
			}
		}
		expiresAt = &parsed
	}

	token := &service.TokenCreateRequest{
		Token:         req.Token,
		Name:          req.Name,
		Description:   req.Description,
		TenantAddress: req.TenantAddress,
		ProxyURL:      req.ProxyURL,
		MaxRequests:   req.MaxRequests,
		Weight:        req.Weight,
		IsShared:      req.IsShared,
		Email:         req.Email,
		ExpiresAt:     expiresAt,
	}

	created, err := h.svc.CreateToken(token)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, created)
}

func (h *TokenHandler) Get(c *gin.Context) {
	token, err := h.svc.GetByID(c.Param("id"))
	if err != nil {
		Error(c, 404, "token not found")
		return
	}
	Success(c, token)
}

func (h *TokenHandler) Update(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request")
		return
	}
	if err := h.svc.Update(c.Param("id"), req); err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, nil)
}

func (h *TokenHandler) Delete(c *gin.Context) {
	if err := h.svc.Delete(c.Param("id")); err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, nil)
}

func (h *TokenHandler) UnlockCooldown(c *gin.Context) {
	if err := h.svc.UnlockCooldown(c.Param("id")); err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, nil)
}

func (h *TokenHandler) Validate(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		Error(c, 400, "token required")
		return
	}
	token, err := h.svc.ValidateToken(tokenStr)
	if err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, gin.H{"valid": true, "token": token})
}

func (h *TokenHandler) Stats(c *gin.Context) {
	stats, err := h.svc.GetStats()
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, stats)
}

func (h *TokenHandler) BatchImport(c *gin.Context) {
	var req struct {
		Tokens []service.TokenCreateRequest `json:"tokens" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request")
		return
	}

	success, failed, err := h.svc.BatchImportTokens(req.Tokens)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, gin.H{"success": success, "failed": failed})
}

func (h *TokenHandler) SmartLoginImport(c *gin.Context) {
	if h.smartImport == nil {
		Error(c, 500, "smart login import service unavailable")
		return
	}

	var req service.SmartTokenImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request: "+err.Error())
		return
	}

	result, err := h.smartImport.Import(&req)
	if err != nil {
		switch typed := err.(type) {
		case *service.SmartTokenImportSelectionError:
			ErrorWithData(c, 409, typed.Error(), typed.Result)
		case *service.SmartTokenImportUnsupportedError:
			ErrorWithData(c, 400, typed.Error(), typed.Result)
		default:
			Error(c, 500, err.Error())
		}
		return
	}

	Success(c, result)
}

func (h *TokenHandler) GetTokenUsers(c *gin.Context) {
	Success(c, []interface{}{})
}

func (h *TokenHandler) GetBanReason(c *gin.Context) {
	Success(c, gin.H{"reason": ""})
}

func (h *TokenHandler) BatchRefreshAuthSession(c *gin.Context) {
	Success(c, nil)
}
