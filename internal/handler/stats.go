package handler

import (
	"windsurf-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	svc *service.StatsService
}

func NewStatsHandler(svc *service.StatsService) *StatsHandler {
	return &StatsHandler{svc: svc}
}

func (h *StatsHandler) Overview(c *gin.Context) {
	data, err := h.svc.GetOverview()
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, data)
}

func (h *StatsHandler) Trend(c *gin.Context) {
	rangeLabel := c.DefaultQuery("range", "7d")
	data, err := h.svc.GetTrend(rangeLabel)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, data)
}

func (h *StatsHandler) TokenStats(c *gin.Context) {
	data, err := h.svc.GetTokenStats(c.Param("id"))
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, data)
}

func (h *StatsHandler) Usage(c *gin.Context) {
	page, pageSize := getPageParams(c)
	logs, total, err := h.svc.GetUsage(page, pageSize)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, gin.H{"list": logs, "total": total, "page": page, "page_size": pageSize})
}

func (h *StatsHandler) Cleanup(c *gin.Context) {
	var req struct {
		Days int `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Days <= 0 {
		req.Days = 30
	}
	count, err := h.svc.CleanupOldLogs(req.Days)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, gin.H{"deleted": count})
}
