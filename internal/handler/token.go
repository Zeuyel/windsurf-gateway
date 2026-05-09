package handler

import (
	"windsurf-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type TokenHandler struct {
	svc *service.TokenService
}

func NewTokenHandler(svc *service.TokenService) *TokenHandler {
	return &TokenHandler{svc: svc}
}

func (h *TokenHandler) List(c *gin.Context) {
	page, pageSize := getPageParams(c)
	status := c.Query("status")
	tokens, total, err := h.svc.List(page, pageSize, status)
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
		IsShared      bool   `json:"is_shared"`
		Email         string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request: "+err.Error())
		return
	}

	token := &service.TokenCreateRequest{
		Token:         req.Token,
		Name:          req.Name,
		Description:   req.Description,
		TenantAddress: req.TenantAddress,
		ProxyURL:      req.ProxyURL,
		MaxRequests:   req.MaxRequests,
		IsShared:      req.IsShared,
		Email:         req.Email,
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

func (h *TokenHandler) GetTokenUsers(c *gin.Context) {
	Success(c, []interface{}{})
}

func (h *TokenHandler) GetBanReason(c *gin.Context) {
	Success(c, gin.H{"reason": ""})
}

func (h *TokenHandler) BatchRefreshAuthSession(c *gin.Context) {
	Success(c, nil)
}
