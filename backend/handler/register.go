package handler

import (
	"errors"
	"net/http"

	"echo/middleware"
	"echo/model"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *UserHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.L().Warn("注册参数校验失败", zap.Error(err))
		Error(c, http.StatusBadRequest, "参数不合法：账号和昵称必填，账号 3-64 位、昵称最多 7 位、密码 8-64 位，且账号和密码只能包含大小写字母和数字")
		return
	}

	u, err := h.svc.Register(c.Request.Context(), req)
	if errors.Is(err, model.ErrAccountTaken) {
		Error(c, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		middleware.L().Error("注册失败", zap.Error(err))
		Error(c, http.StatusInternalServerError, "注册失败，请稍后重试")
		return
	}

	Created(c, u)
}