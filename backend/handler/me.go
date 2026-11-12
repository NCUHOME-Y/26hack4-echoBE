package handler

import (
	"errors"
	"log"
	"net/http"

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
	if errors.Is(err, model.ErrUserNotFound) { //token 有效但账号已不存在，让前端跳登录
		log.Printf("资料查询：账号不存在 userID=%d err=%v", userID, err)
		Error(c, http.StatusUnauthorized, "账号不存在或已注销，请重新登录")
		return
	}
	if err != nil {
		log.Printf("资料查询失败 userID=%d err=%v", userID, err)
		Error(c, http.StatusInternalServerError, "获取资料失败")
		return
	}
	Success(c, profile)
}

// UpdateAvatar换头像
func (h *MeHandler) UpdateAvatar(c *gin.Context) {
	var req model.UpdateAvatarReq
	if err := c.ShouldBindJSON(&req); err != nil { //binding 只管必填和长度，格式规则在 service
		log.Printf("换头像参数校验失败: %v", err)
		Error(c, http.StatusBadRequest, "参数不合法：avatar 必填且不超过 32 字符")
		return
	}
	userID := c.GetInt64("userID")
	if err := h.meSvc.UpdateAvatar(c.Request.Context(), userID, req.Avatar); err != nil {
		if errors.Is(err, model.ErrInvalidAvatar) { //格式不对是调用方的错
			log.Printf("换头像格式非法 userID=%d avatar=%q", userID, req.Avatar)
			Error(c, http.StatusBadRequest, "参数不合法：avatar 格式应为 avatar-N")
			return
		}
		if errors.Is(err, model.ErrUserNotFound) {
			log.Printf("换头像：账号不存在 userID=%d err=%v", userID, err)
			Error(c, http.StatusUnauthorized, "账号不存在或已注销，请重新登录")
			return
		}
		log.Printf("换头像失败 userID=%d err=%v", userID, err)
		Error(c, http.StatusInternalServerError, "更新头像失败")
		return
	}
	Success(c, gin.H{"avatar": req.Avatar})
}
