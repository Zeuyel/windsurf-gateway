package handler

import (
	"windsurf-gateway/internal/service"

	"github.com/gin-gonic/gin"
)

type RequestRecordHandler struct {
	svc *service.RequestRecordService
}

func NewRequestRecordHandler(svc *service.RequestRecordService) *RequestRecordHandler {
	return &RequestRecordHandler{svc: svc}
}

func (h *RequestRecordHandler) List(c *gin.Context) {
	page, pageSize := getPageParams(c)
	filter := service.RequestLogFilter{
		Query:           c.Query("q"),
		TokenID:         c.Query("token_id"),
		Username:        c.Query("username"),
		FailureCategory: c.Query("failure_category"),
		StatusCode:      service.ParseStatusCodeFilter(c.Query("status_code")),
	}
	logs, total, err := h.svc.List(page, pageSize, filter)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, gin.H{"list": logs, "total": total, "page": page, "page_size": pageSize})
}

func (h *RequestRecordHandler) Search(c *gin.Context) {
	query := c.Query("q")
	page, pageSize := getPageParams(c)
	logs, total, err := h.svc.Search(query, page, pageSize)
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, gin.H{"list": logs, "total": total, "page": page, "page_size": pageSize})
}

type PluginHandler struct {
	svc *service.PluginService
}

func NewPluginHandler(svc *service.PluginService) *PluginHandler {
	return &PluginHandler{svc: svc}
}

func (h *PluginHandler) GetList(c *gin.Context) {
	plugins, err := h.svc.GetList()
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, plugins)
}

func (h *PluginHandler) Download(c *gin.Context) {
	id := parseUint(c.Param("id"))
	plugin, err := h.svc.GetByID(id)
	if err != nil {
		Error(c, 404, "plugin not found")
		return
	}
	h.svc.IncrementDownload(id)
	c.File(plugin.FilePath)
}

type SystemConfigHandler struct {
	svc *service.SystemConfigService
}

func NewSystemConfigHandler(svc *service.SystemConfigService) *SystemConfigHandler {
	return &SystemConfigHandler{svc: svc}
}

func (h *SystemConfigHandler) GetSystemConfig(c *gin.Context) {
	configs, err := h.svc.GetAll()
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, configs)
}

func (h *SystemConfigHandler) UpdateSystemConfig(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "invalid request")
		return
	}
	for k, v := range req {
		if err := h.svc.Set(k, v); err != nil {
			Error(c, 500, err.Error())
			return
		}
	}
	Success(c, nil)
}

func (h *SystemConfigHandler) GetSystemStats(c *gin.Context) {
	stats, err := h.svc.GetSystemStats()
	if err != nil {
		Error(c, 500, err.Error())
		return
	}
	Success(c, stats)
}

type SystemHandler struct{}

func NewSystemHandler() *SystemHandler {
	return &SystemHandler{}
}

func (h *SystemHandler) GetVersion(c *gin.Context) {
	Success(c, gin.H{"version": "1.0.0", "name": "Windsurf Gateway"})
}
