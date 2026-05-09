package handler

import (
	"windsurf-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request")
		return
	}

	user, token, err := h.svc.Login(req.Username, req.Password)
	if err != nil {
		Error(c, 401, err.Error())
		return
	}

	Success(c, gin.H{
		"user":  user,
		"token": token,
	})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	Success(c, nil)
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID := c.GetUint("user_id")
	user, err := h.svc.GetUserByID(userID)
	if err != nil {
		Error(c, 404, "user not found")
		return
	}
	Success(c, user)
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request")
		return
	}

	user, err := h.svc.GetUserByID(userID)
	if err != nil {
		Error(c, 404, "user not found")
		return
	}

	user.Email = req.Email
	Success(c, user)
}

func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			Error(c, 401, "authorization required")
			c.Abort()
			return
		}

		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		claims, err := h.svc.ValidateToken(token)
		if err != nil {
			Error(c, 401, "invalid token")
			c.Abort()
			return
		}

		if userID, ok := (*claims)["user_id"].(float64); ok {
			c.Set("user_id", uint(userID))
		}
		if username, ok := (*claims)["username"].(string); ok {
			c.Set("username", username)
		}
		if role, ok := (*claims)["role"].(string); ok {
			c.Set("role", role)
		}

		c.Next()
	}
}

func (h *AuthHandler) AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		if role != "admin" {
			Error(c, 403, "admin access required")
			c.Abort()
			return
		}
		c.Next()
	}
}
