package handler

import (
	"windsurf-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type InvitationCodeHandler struct {
	svc *service.InvitationCodeService
}

func NewInvitationCodeHandler(svc *service.InvitationCodeService) *InvitationCodeHandler {
	return &InvitationCodeHandler{svc: svc}
}

func (h *InvitationCodeHandler) Validate(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		Error(c, 400, "code required")
		return
	}
	valid, err := h.svc.Validate(code)
	if err != nil || !valid {
		Error(c, 400, "invalid invitation code")
		return
	}
	Success(c, gin.H{"valid": true})
}

func (h *InvitationCodeHandler) List(c *gin.Context) {
	codes, err := h.svc.List()
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, codes)
}

func (h *InvitationCodeHandler) Generate(c *gin.Context) {
	var req struct {
		MaxUses int `json:"max_uses"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.MaxUses = 1
	}
	if req.MaxUses <= 0 {
		req.MaxUses = 1
	}
	code, err := h.svc.Generate(req.MaxUses)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, code)
}

func (h *InvitationCodeHandler) Delete(c *gin.Context) {
	id := parseUint(c.Param("id"))
	if err := h.svc.Delete(id); err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, nil)
}
