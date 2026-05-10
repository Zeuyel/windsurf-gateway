package handler

import (
	"windsurf-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type UserAuthHandler struct {
	svc *service.UserAuthService
}

func NewUserAuthHandler(svc *service.UserAuthService) *UserAuthHandler {
	return &UserAuthHandler{svc: svc}
}

func (h *UserAuthHandler) Register(c *gin.Context) {
	var req struct {
		Username       string `json:"username" binding:"required"`
		Password       string `json:"password" binding:"required"`
		Email          string `json:"email"`
		InvitationCode string `json:"invitation_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request: "+err.Error())
		return
	}

	user, token, err := h.svc.Register(req.Username, req.Password, req.Email, req.InvitationCode)
	if err != nil {
		Error(c, 400, err.Error())
		return
	}

	Success(c, gin.H{"user": user, "token": token})
}

func (h *UserAuthHandler) Login(c *gin.Context) {
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

	Success(c, gin.H{"user": user, "token": token})
}

func (h *UserAuthHandler) Logout(c *gin.Context) {
	Success(c, nil)
}

func (h *UserAuthHandler) Refresh(c *gin.Context) {
	Success(c, nil)
}

func (h *UserAuthHandler) Me(c *gin.Context) {
	userID := c.GetUint("user_id")
	user, err := h.svc.GetUserByID(userID)
	if err != nil {
		Error(c, 404, "user not found")
		return
	}
	Success(c, user)
}

func (h *UserAuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request")
		return
	}
	h.svc.UpdateUser(userID, map[string]interface{}{"email": req.Email})
	Success(c, nil)
}

func (h *UserAuthHandler) ChangePassword(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		OldPassword string `json:"old_password" binding:"required"`
		NewPassword string `json:"new_password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request")
		return
	}
	if err := h.svc.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, nil)
}

func (h *UserAuthHandler) RegenerateAPIToken(c *gin.Context) {
	userID := c.GetUint("user_id")
	token, err := h.svc.RegenerateAPIToken(userID)
	if err != nil {
		Error(c, 500, "failed to regenerate api token")
		return
	}
	Success(c, gin.H{"api_token": token})
}

func (h *UserAuthHandler) ListUsers(c *gin.Context) {
	page, pageSize := getPageParams(c)
	users, total, err := h.svc.ListUsers(page, pageSize, c.Query("username"), c.Query("status"))
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, gin.H{"list": users, "total": total, "page": page, "page_size": pageSize})
}

func (h *UserAuthHandler) AdminCreateUser(c *gin.Context) {
	var req struct {
		Username           string `json:"username" binding:"required"`
		Email              string `json:"email"`
		Password           string `json:"password" binding:"required"`
		Role               string `json:"role"`
		Status             string `json:"status"`
		UnlimitedAccess    bool   `json:"unlimited_access"`
		RateLimitPerMinute int    `json:"rate_limit_per_minute"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request: "+err.Error())
		return
	}

	user, err := h.svc.CreateManagedUser(req.Username, req.Password, req.Email, req.Role, req.Status, req.UnlimitedAccess, req.RateLimitPerMinute)
	if err != nil {
		Error(c, 400, err.Error())
		return
	}
	Success(c, user)
}

func (h *UserAuthHandler) AdminUpdateUser(c *gin.Context) {
	id := c.Param("id")
	var req struct {
		Email              string `json:"email"`
		Role               string `json:"role"`
		Status             string `json:"status"`
		UnlimitedAccess    *bool  `json:"unlimited_access"`
		RateLimitPerMinute *int   `json:"rate_limit_per_minute"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request")
		return
	}
	uid := parseUint(id)
	updates := map[string]interface{}{}
	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Role != "" {
		updates["role"] = req.Role
	}
	if req.Status != "" {
		updates["status"] = req.Status
	}
	if req.UnlimitedAccess != nil {
		updates["unlimited_access"] = *req.UnlimitedAccess
	}
	if req.RateLimitPerMinute != nil && *req.RateLimitPerMinute > 0 {
		updates["rate_limit_per_minute"] = *req.RateLimitPerMinute
	}
	if len(updates) == 0 {
		Error(c, 400, "no changes submitted")
		return
	}
	if err := h.svc.UpdateUser(uid, updates); err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, nil)
}

func (h *UserAuthHandler) BanUser(c *gin.Context) {
	id := parseUint(c.Param("id"))
	if err := h.svc.BanUser(id); err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, nil)
}

func (h *UserAuthHandler) UnbanUser(c *gin.Context) {
	id := parseUint(c.Param("id"))
	if err := h.svc.UnbanUser(id); err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, nil)
}

func (h *UserAuthHandler) DeleteUser(c *gin.Context) {
	id := parseUint(c.Param("id"))
	if err := h.svc.DeleteUser(id); err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, nil)
}

func (h *UserAuthHandler) ToggleSharedPermission(c *gin.Context) {
	Success(c, nil)
}

func (h *UserAuthHandler) UserAuthMiddleware() gin.HandlerFunc {
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

func (h *UserAuthHandler) GetUserSettings(c *gin.Context) {
	userID := c.GetUint("user_id")
	user, err := h.svc.GetUserByID(userID)
	if err != nil {
		Error(c, 404, "user not found")
		return
	}

	accessMode := "windsurf_credit_limited"
	if user.UnlimitedAccess {
		accessMode = "unlimited"
	}

	Success(c, gin.H{
		"user": user,
		"access": gin.H{
			"mode":                  accessMode,
			"unlimited_access":      user.UnlimitedAccess,
			"rate_limit_per_minute": user.RateLimitPerMinute,
			"used_requests":         user.UsedRequests,
			"token_status":          user.TokenStatus,
		},
	})
}

func (h *UserAuthHandler) UpdateUserSettings(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request")
		return
	}
	if err := h.svc.UpdateUser(userID, map[string]interface{}{"email": req.Email}); err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, nil)
}

func parseUint(s string) uint {
	var n uint
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + uint(c-'0')
	}
	return n
}
