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

// Create 发表回复 POST /api/questions/:id/replies
func (h *ReplyHandler) Create(c *gin.Context) {
	questionID, err := strconv.ParseInt(c.Param("id"), 10, 64) //先判路径，与列表接口同一个顺序
	if err != nil || questionID <= 0 {
		Error(c, http.StatusNotFound, model.ErrQuestionNotFound.Error())
		return
	}

	//解析请求体，获取评论内容
	var req model.CreateReplyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("发表评论：请求体解析失败 questionID=%d err=%v", questionID, err)
		Error(c, http.StatusBadRequest, "参数不合法：评论内容 1-32 字符")
		return
	}

	userID := c.GetInt64("userID")

	//调用业务层，创建评论
	view, err := h.replySvc.Create(c.Request.Context(), questionID, userID, req.Content)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrQuestionNotFound):
			Error(c, http.StatusNotFound, err.Error())
		case errors.Is(err, model.ErrInvalidReplyContent):
			Error(c, http.StatusBadRequest, "参数不合法：评论内容 1-32 字符")
		case errors.Is(err, model.ErrUserNotFound):
			Error(c, http.StatusUnauthorized, "账号不存在或已注销，请重新登录") //token有效但账号被删，与发布问题同一口径
		default:
			log.Printf("发表评论失败 questionID=%d userID=%d err=%v", questionID, userID, err)
			Error(c, http.StatusInternalServerError, "发送失败，请稍后重试")
		}
		return
	}

	Created(c, view)
}

// Delete 删除回复
func (h *ReplyHandler) Delete(c *gin.Context) {
	replyID, err := strconv.ParseInt(c.Param("id"), 10, 64) //从路径中读取回复ID
	if err != nil || replyID <= 0 {
		Error(c, http.StatusNotFound, model.ErrReplyNotFound.Error()) //非法ID与不存在同一句，与其他接口口径一致
		return
	}

	userID := c.GetInt64("userID")

	result, err := h.replySvc.Delete(c.Request.Context(), replyID, userID)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrReplyNotFound):
			Error(c, http.StatusNotFound, err.Error())
		case errors.Is(err, model.ErrNotReplyOwner):
			Error(c, http.StatusForbidden, err.Error())
		default: //剩下的才是真故障，比如评论在但所属的题取不到
			log.Printf("删除评论失败 replyID=%d userID=%d err=%v", replyID, userID, err)
			Error(c, http.StatusInternalServerError, "服务异常，请稍后重试")
		}
		return
	}

	Success(c, result) //200，软删除不返回 204，契约要 {id, deleted: true}
}
