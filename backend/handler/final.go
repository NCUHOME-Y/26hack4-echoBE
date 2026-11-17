package handler

import (
	"errors"
	"net/http"
	"strconv"

	"echo/middleware"
	"echo/model"
	"echo/service"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type FinalHandler struct {
	settleSvc *service.SettleService
}

//把 Service 注入 Handler
func NewFinalHandler(settleSvc *service.SettleService) *FinalHandler {
	return &FinalHandler{settleSvc: settleSvc}
}

// Publish 公布最终选择 PUT /api/questions/:id/final
func (h *FinalHandler) Publish(c *gin.Context) {
	questionID, err := strconv.ParseInt(c.Param("id"), 10, 64) //从路径中读取问题ID
	if err != nil || questionID <= 0 {
		Error(c, http.StatusNotFound, model.ErrQuestionNotFound.Error()) //与投票、详情、抽题、跳过同一口径
		return
	}

	//解析请求体
	var req model.SetFinalReq
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.L().Warn("公布最终选择：请求体解析失败", zap.Int64("question_id", questionID), zap.Error(err))
		Error(c, http.StatusBadRequest, "参数不合法：option_id 必填")
		return
	}

	userID := c.GetInt64("userID")

	//调用service层的 PublishFinal 方法
	result, err := h.settleSvc.PublishFinal(c.Request.Context(), questionID, userID, req.OptionID)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrQuestionNotFound), errors.Is(err, model.ErrOptionNotInQuestion):
			Error(c, http.StatusNotFound, err.Error())
		case errors.Is(err, model.ErrNotAsker):
			Error(c, http.StatusForbidden, err.Error())
		case errors.Is(err, model.ErrFinalAlreadySet):
			Error(c, http.StatusConflict, err.Error())
		default: //剩下的才是真故障
			middleware.L().Error("公布最终选择失败", zap.Int64("question_id", questionID), zap.Int64("user_id", userID), zap.Error(err))
			Error(c, http.StatusInternalServerError, "服务异常，请稍后重试")
		}
		return
	}

	Success(c, result) //200，公布即结束，响应体是 question_id 与 final_option_id
}
