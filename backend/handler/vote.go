package handler

import (
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

	//TODO 步 2 起在这里接 h.voteSvc.CastVote，并把下面这行占位删掉
	log.Printf("投票：骨架占位 questionID=%d userID=%d optionID=%d", questionID, userID, req.OptionID)
	Error(c, http.StatusInternalServerError, "投票失败，请稍后重试")
}
