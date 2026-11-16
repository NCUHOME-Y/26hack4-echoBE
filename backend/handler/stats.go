package handler

import (
	"net/http"

	"echo/middleware"
	"echo/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type StatsHandler struct {
	statsSvc *service.StatsService
}

//把 Service 注入 Handler
func NewStatsHandler(statsSvc *service.StatsService) *StatsHandler {
	return &StatsHandler{statsSvc: statsSvc}
}

// Daily 今日统计 
func (h *StatsHandler) Daily(c *gin.Context) {
	result, err := h.statsSvc.Daily(c.Request.Context())
	if err != nil {
		middleware.L().Error("今日统计失败", zap.Error(err))
		Error(c, http.StatusInternalServerError, "统计获取失败")
		return
	}

	Success(c, result)
}
