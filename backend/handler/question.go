package handler

import (
	"errors"
	"log"
	"net/http"

	"echo/model"
	"echo/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type QuestionHandler struct {
	questionSvc *service.QuestionService
}

func NewQuestionHandler(questionSvc *service.QuestionService) *QuestionHandler {
	return &QuestionHandler{questionSvc: questionSvc}
}

// Create发布问题与选项
func (h *QuestionHandler) Create(c *gin.Context) {
	var req model.CreateQuestionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("发布问题：请求体解析失败 err=%v", err)
		Error(c, http.StatusBadRequest, "请求参数格式错误")
		return
	}

	userID := c.GetInt64("userID")
	detail, err := h.questionSvc.Create(c.Request.Context(), userID, req)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrInvalidDescription):
			Error(c, http.StatusBadRequest, "参数不合法：问题描述 1-100 字符")
		case errors.Is(err, model.ErrInvalidOptionCount):
			Error(c, http.StatusBadRequest, "参数不合法：选项数量只能是 2-4 个")
		case errors.Is(err, model.ErrInvalidOptionLabel):
			Error(c, http.StatusBadRequest, "参数不合法：选项文字 1-18 字符")
		case errors.Is(err, model.ErrInvalidDuration):
			Error(c, http.StatusBadRequest, "参数不合法：结算时间只能是 1/3/6/12/24/48 小时")
		case errors.Is(err, model.ErrUserNotFound):
			Error(c, http.StatusUnauthorized, "账号不存在或已注销，请重新登录")
		default:
			log.Printf("发布问题失败 userID=%d err=%v", userID, err)
			Error(c, http.StatusInternalServerError, "发布失败，请稍后重试")
		}
		return
	}

	Created(c, detail) //201+code:201+msg:"success"
}

// Detail问题详情与实时占比
func (h *QuestionHandler) Detail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		//非正整数的ID就是库里没有这题，按404走
		Error(c, http.StatusNotFound, model.ErrQuestionNotFound.Error())
		return
	}

	userID := c.GetInt64("userID")
	detail, err := h.questionSvc.GetDetail(c.Request.Context(), id, userID)
	if err != nil {
		switch {
		case errors.Is(err, model.ErrQuestionNotFound):
			Error(c, http.StatusNotFound, "问题不存在")
		default:
			log.Printf("获取问题详情失败 questionID=%d err=%v", id, err)
			Error(c, http.StatusInternalServerError, "获取问题详情失败")
		}
		return
	}

	Success(c, detail)
}

// Random 随机抽一题
func (h *QuestionHandler) Random(c *gin.Context) {
	userID := c.GetInt64("userID") //从JWT中间件中解析出用户ID
	detail, err := h.questionSvc.Random(c.Request.Context(), userID) //调用service层完成抽题业务
	if err != nil {
		log.Printf("抽题失败 userID=%d err=%v", userID, err)
		Error(c, http.StatusInternalServerError, "抽题失败，请稍后重试")
		return
	} //处理数据库错误
	if detail == nil {
		c.JSON(http.StatusOK, Response{Code: 200, Msg: "暂时没有可回答的问题", Data: nil})
		return
	} //空池也是正常状态
	Success(c, detail)
}
