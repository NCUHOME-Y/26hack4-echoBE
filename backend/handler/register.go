package handler

import (
	"errors"
	"log"
	"net/http"

	"echo/model"

	"github.com/gin-gonic/gin"
)

func (h *UserHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("注册参数校验失败: %v", err)
		Error(c, http.StatusBadRequest, "参数不合法：账号和昵称必填，密码 8-64 位")
		return
	}

	u, err := h.svc.Register(c.Request.Context(), req)
	if errors.Is(err, model.ErrAccountTaken) {
		Error(c, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		log.Printf("注册失败: %v", err)
		Error(c, http.StatusInternalServerError, "注册失败，请稍后重试")
		return
	}

	Success(c, u)
}