package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"echo/middleware"
	"echo/model"
	"echo/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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
		middleware.L().Warn("资料查询：账号不存在", zap.Int64("user_id", userID), zap.Error(err))
		Error(c, http.StatusUnauthorized, "账号不存在或已注销，请重新登录")
		return
	}
	if err != nil {
		middleware.L().Error("资料查询失败", zap.Int64("user_id", userID), zap.Error(err))
		Error(c, http.StatusInternalServerError, "获取资料失败")
		return
	}
	Success(c, profile)
}

// UpdateAvatar换头像
func (h *MeHandler) UpdateAvatar(c *gin.Context) {
	var req model.UpdateAvatarReq
	if err := c.ShouldBindJSON(&req); err != nil { //binding 只管必填和长度，格式规则在 service
		middleware.L().Warn("换头像参数校验失败", zap.Error(err))
		Error(c, http.StatusBadRequest, "参数不合法：avatar 必填且不超过 32 字符")
		return
	}
	userID := c.GetInt64("userID")
	if err := h.meSvc.UpdateAvatar(c.Request.Context(), userID, req.Avatar); err != nil {
		if errors.Is(err, model.ErrInvalidAvatar) { //格式不对是调用方的错
			middleware.L().Warn("换头像格式非法", zap.Int64("user_id", userID), zap.String("avatar", req.Avatar))
			Error(c, http.StatusBadRequest, "参数不合法：avatar 格式应为 avatar-N")
			return
		}
		if errors.Is(err, model.ErrUserNotFound) {
			middleware.L().Warn("换头像：账号不存在", zap.Int64("user_id", userID), zap.Error(err))
			Error(c, http.StatusUnauthorized, "账号不存在或已注销，请重新登录")
			return
		}
		middleware.L().Error("换头像失败", zap.Int64("user_id", userID), zap.Error(err))
		Error(c, http.StatusInternalServerError, "更新头像失败")
		return
	}
	Success(c, gin.H{"avatar": req.Avatar})
}

// ListMyQuestions 获取当前用户发布的问题列表
func (h *MeHandler) ListMyQuestions(c *gin.Context) {
	page, pageSize := parsePage(c)

	userID := c.GetInt64("userID")
	data, err := h.meSvc.ListMyQuestions(
		c.Request.Context(),
		userID,
		page,
		pageSize,
	)
	if err != nil {
		middleware.L().Error("获取我发布的问题列表失败", zap.Int64("user_id", userID), zap.Error(err))
		Error(c, http.StatusInternalServerError, "获取失败")
		return
	}

	Success(c, data)
}

// ListMyAnswers 获取当前用户投过票的问题历史
func (h *MeHandler) ListMyAnswers(c *gin.Context) {
	page, pageSize := parsePage(c)

	keyword := strings.TrimSpace(c.Query("keyword"))
	if utf8.RuneCountInString(keyword) > 100 {
		Error(c, http.StatusBadRequest, "参数不合法：关键词最多 100 字符")
		return
	}

	userID := c.GetInt64("userID")
	data, err := h.meSvc.ListMyAnswers(
		c.Request.Context(),
		userID,
		page,
		pageSize,
		keyword,
	)
	if err != nil {
		middleware.L().Error("获取我参与的问题失败", zap.Int64("user_id", userID), zap.Error(err))
		Error(c, http.StatusInternalServerError, "获取失败")
		return
	}

	Success(c, data)
}

// parsePage 统一解析分页参数：解析失败回退默认，越界夹紧到契约允许的区间
func parsePage(c *gin.Context) (int, int) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil || pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100 //与 openapi 的 maximum:100 对齐，避免一次拉爆
	}

	return page, pageSize
}
