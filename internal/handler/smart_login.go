package handler

import (
	"windsurf-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type SmartLoginHandler struct {
	smartLoginService   *service.SmartLoginService
	firebaseAuthService *service.FirebaseAuthService
	devinAuthService    *service.DevinAuthService
}

func NewSmartLoginHandler(
	smartLoginService *service.SmartLoginService,
	firebaseAuthService *service.FirebaseAuthService,
	devinAuthService *service.DevinAuthService,
) *SmartLoginHandler {
	return &SmartLoginHandler{
		smartLoginService:   smartLoginService,
		firebaseAuthService: firebaseAuthService,
		devinAuthService:    devinAuthService,
	}
}

// SniffLoginMethod 智能识别登录方式
// POST /api/v1/smart-login/sniff
func (h *SmartLoginHandler) SniffLoginMethod(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.smartLoginService.SniffLoginMethod(req.Email)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

// FirebaseLogin Firebase 邮箱密码登录
// POST /api/v1/smart-login/firebase
func (h *SmartLoginHandler) FirebaseLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.firebaseAuthService.SignIn(req.Email, req.Password)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

// FirebaseRefreshToken Firebase 刷新 Token
// POST /api/v1/smart-login/firebase/refresh
func (h *SmartLoginHandler) FirebaseRefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.firebaseAuthService.RefreshToken(req.RefreshToken)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}

// DevinLogin Devin 邮箱密码登录
// POST /api/v1/smart-login/devin
func (h *SmartLoginHandler) DevinLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		OrgID    string `json:"org_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "invalid request"})
		return
	}

	result, err := h.devinAuthService.LoginWithPassword(req.Email, req.Password, req.OrgID)
	if err != nil {
		c.JSON(401, gin.H{"error": err.Error()})
		return
	}

	c.JSON(200, result)
}
