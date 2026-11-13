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

type VoteHandler struct {
	voteSvc *service.VoteService
}

//把 Service 注入 Handler
func NewVoteHandler(voteSvc *service.VoteService) *VoteHandler {
	return &VoteHandler{voteSvc: voteSvc}
}

// Create 提交投票 POST /api/questions/:id/votes
func (h *VoteHandler) Create(c *gin.Context) {
	questionID, err := strconv.ParseInt(c.Param("id"), 10, 64) //从路径中读取问题ID
	if err != nil || questionID <= 0 {
		Error(c, http.StatusNotFound, model.ErrQuestionNotFound.Error()) //统一按404处理
		return
	}

	//解析请求体
	var req model.CastVoteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("投票：请求体解析失败 questionID=%d err=%v", questionID, err)
		Error(c, http.StatusBadRequest, "参数不合法：option_id 必填")
		return
	}

	//从上下文中获取 userID（JWTAuth 中间件已经放进去了）
	userID := c.GetInt64("userID")

	//调 Service 做校验、插票和聚合
	result, err := h.voteSvc.CastVote(c.Request.Context(), questionID, userID, req.OptionID)
	if err != nil {
		//按状态码分组
		switch {
		case errors.Is(err, model.ErrQuestionNotFound), errors.Is(err, model.ErrOptionNotInQuestion):
			Error(c, http.StatusNotFound, err.Error())
		case errors.Is(err, model.ErrQuestionClosed), errors.Is(err, model.ErrOwnQuestionVote),
			errors.Is(err, model.ErrAlreadyVoted), errors.Is(err, model.ErrQuestionSkipped):
			Error(c, http.StatusConflict, err.Error())
		default: //剩下的都是查库失败之类的真故障，不能当成校验不通过
			log.Printf("投票校验失败 questionID=%d userID=%d err=%v", questionID, userID, err)
			Error(c, http.StatusInternalServerError, "投票失败，请稍后重试")
		}
		return
	}

	Created(c, result) //201，响应里直接带最新占比
}
