package handler

import (
	"errors"
	"net/http"

	"echo/middleware"
	"echo/model"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *UserHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	//读取 JSON并绑定到req中
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.L().Warn("登录参数校验失败", zap.Error(err))
		Error(c, http.StatusBadRequest, "参数不合法：账号和密码必填")
		return
	}

	u, err := h.svc.Login(c.Request.Context(), req)
	//处理登录结果
	if errors.Is(err, model.ErrInvalidCredentials) {
		Error(c, http.StatusUnauthorized, err.Error())
		return
	}
	if err != nil {
		middleware.L().Error("登录失败", zap.Error(err))
		Error(c, http.StatusInternalServerError, "登录失败，请稍后重试")
		return
	}

	token, err := middleware.GenerateToken(u.ID) //暂时token的有效期是24小时，后续可以改时间
	if err != nil {
		middleware.L().Error("签发 token 失败", zap.Error(err))
		Error(c, http.StatusInternalServerError, "登录失败，请稍后重试")
		return
	}

	Success(c, gin.H{"token": token, "user": u})
}
