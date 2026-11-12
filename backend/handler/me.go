package handler

import (
	"echo/model"
	"echo/service"

	"github.com/gin-gonic/gin"
)

type MeHandler struct {
	meSvc *service.MeService
}

func NewMeHandler(meSvc *service.MeService) *MeHandler {
	return &MeHandler{meSvc: meSvc}
}

// profile返回当前登录的用户资料（“我的”页面数据源
func (h *MeHandler) Profile(c *gin.Context) {
	userID := c.GetInt64("userID")
	profile, err := h.meSvc.GetProfile(c.Request.Context(), userID)
	if err != nil {
		Error(c, 500, "获取资料失败")
		return
	}
	Success(c, profile)
}

// UpdateAvatar换头像
func (h *MeHandler) UpdateAvatar(c *gin.Context) {
	var req model.UpdateAvatarReq
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, 400, "参数不合法")
		return
	}
	userID := c.GetInt64("userID")
	if err := h.meSvc.UpdateAvatar(c.Request.Context(), userID, req.AvatarID); err != nil {
		Error(c, 500, "更新头像失败")
		return
	}
	Success(c, gin.H{"avatar_id": req.AvatarID})
}
