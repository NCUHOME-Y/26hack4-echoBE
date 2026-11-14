package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"echo/model"
	"echo/service"

	"github.com/gin-gonic/gin"
)

type ReplyHandler struct {
	replySvc *service.ReplyService
}

//把 Service 注入 Handler
func NewReplyHandler(replySvc *service.ReplyService) *ReplyHandler {
	return &ReplyHandler{replySvc: replySvc}
}

// List 评论列表 GET /api/questions/:id/replies
func (h *ReplyHandler) List(c *gin.Context) {
	questionID, err := strconv.ParseInt(c.Param("id"), 10, 64) //从路径中读取问题ID
	if err != nil || questionID <= 0 {
		Error(c, http.StatusNotFound, model.ErrQuestionNotFound.Error()) //与投票、详情、跳过同一口径
		return
	}

	userID := c.GetInt64("userID")

	result, err := h.replySvc.List(c.Request.Context(), questionID, userID)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrQuestionNotFound):
			Error(c, http.StatusNotFound, err.Error())
		default: //剩下的都是查库失败之类的真故障，不能当成题不存在
			log.Printf("评论列表失败 questionID=%d userID=%d err=%v", questionID, userID, err)
			Error(c, http.StatusInternalServerError, "服务异常，请稍后重试")
		}
		return
	}

	Success(c, result)
}
