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

type SkipHandler struct {
	skipSvc *service.SkipService
}

//把 Service 注入 Handler
func NewSkipHandler(skipSvc *service.SkipService) *SkipHandler {
	return &SkipHandler{skipSvc: skipSvc}
}

// Create 跳过这题
func (h *SkipHandler) Create(c *gin.Context) {
	questionID, err := strconv.ParseInt(c.Param("id"), 10, 64) //从路径中读取问题ID
	if err != nil || questionID <= 0 {
		Error(c, http.StatusNotFound, model.ErrQuestionNotFound.Error()) //与投票、详情同一口径
		return
	}

	userID := c.GetInt64("userID")

	result, first, err := h.skipSvc.Skip(c.Request.Context(), questionID, userID) //调用service层的 Skip 方法
	if err != nil {
		switch {
		case errors.Is(err, model.ErrQuestionNotFound):
			Error(c, http.StatusNotFound, err.Error())
		case errors.Is(err, model.ErrOwnQuestionSkip), errors.Is(err, model.ErrAlreadyVotedSkip):
			Error(c, http.StatusConflict, err.Error())
		default: //剩下的都是查库写库失败之类的真故障
			middleware.L().Error("跳过失败", zap.Int64("question_id", questionID), zap.Int64("user_id", userID), zap.Error(err))
			Error(c, http.StatusInternalServerError, "操作失败，请稍后重试")
		}
		return
	}

	//首次 201，重复跳过幂等 200，两者响应体相同
	if first {
		Created(c, result)
		return
	} //防止产生重复记录
	Success(c, result)
}
