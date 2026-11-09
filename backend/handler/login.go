package handler

import (
	"errors"
	"log"
	"net/http"

	"echo/model"

	"github.com/gin-gonic/gin"
)

func (h *UserHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	//读取 JSON并绑定到req中
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("登录参数校验失败: %v", err)
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
		log.Printf("登录失败: %v", err)
		Error(c, http.StatusInternalServerError, "登录失败，请稍后重试")
		return
	}

	Success(c, u)
}
