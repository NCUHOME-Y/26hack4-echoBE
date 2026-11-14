package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"echo/model"
	"echo/repository"
)

// replyListLimit 契约规定评论列表最多返回 50 条，库里不删只截断
const replyListLimit = 50

// ReplyService 评论区的业务规则层
type ReplyService struct {
	questionRepo *repository.QuestionRepository //判题在不在，不存在要给 404 而不是空列表
	replyRepo    *repository.ReplyRepository    //捞列表、数总数
	userRepo     *repository.UserRepository     //批量取作者资料
}

//把三个 Repository 注入到 ReplyService
func NewReplyService(questionRepo *repository.QuestionRepository, replyRepo *repository.ReplyRepository,
	userRepo *repository.UserRepository) *ReplyService {
	return &ReplyService{
		questionRepo: questionRepo,
		replyRepo:    replyRepo,
		userRepo:     userRepo,
	}
}

// ReplyView 一条对外的评论，作者资料只做展示，不能点进用户页
type ReplyView struct {
	ID        int64           `json:"id"`
	Content   string          `json:"content"`
	CreatedAt time.Time       `json:"created_at"`
	IsMine    bool            `json:"is_mine"`
	Author    model.UserBrief `json:"author"`
}

// ReplyList 评论列表的响应体
type ReplyList struct {
	List     []ReplyView `json:"list"`
	Returned int64       `json:"returned"` //返回数
	Total    int64       `json:"total"` //实际库里面的总数
}

// List 取某题的评论列表
func (s *ReplyService) List(ctx context.Context, questionID, userID int64) (*ReplyList, error) {
	//确认题是否存在
	if _, err := s.questionRepo.GetByID(ctx, questionID); err != nil {
		return nil, err
	}

	//查询最新的评论
	replies, err := s.replyRepo.ListByQuestion(ctx, questionID, replyListLimit)
	if err != nil {
		return nil, err
	}
	//查询评论总数
	total, err := s.replyRepo.CountByQuestion(ctx, questionID)
	if err != nil {
		return nil, err
	}

	//作者ID去重，避免一个用户多次传给前端
	ids := make([]int64, 0, len(replies))
	seen := make(map[int64]bool, len(replies))
	for _, rp := range replies {
		if !seen[rp.UserID] {
			seen[rp.UserID] = true
			ids = append(ids, rp.UserID)
		}
	}

	//一次性查询所有评论作者
	users, err := s.userRepo.ListByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	//空切片而不是 nil
	list := make([]ReplyView, 0, len(replies))
	//组装对外的评论列表
	for _, rp := range replies {
		u, ok := users[rp.UserID]
		if !ok {
			//作者不存在时返回错误，防止程序崩溃
			return nil, fmt.Errorf("评论 %d 的作者 %d 查不到", rp.ID, rp.UserID)
		}

		//组装对外的评论结构体
		list = append(list, ReplyView{
			ID:        rp.ID,
			Content:   rp.Content,
			CreatedAt: rp.CreatedAt,
			IsMine:    rp.UserID == userID, //是不是我的，在 Go 里判，不进 SQL
			Author:    model.NewUserBrief(u),
		})
	}

	return &ReplyList{List: list, Returned: int64(len(list)), Total: total}, nil
}

// Create 发表一条评论
func (s *ReplyService) Create(ctx context.Context, questionID, userID int64, content string) (*ReplyView, error) {
	//题不存在不允许保存
	if _, err := s.questionRepo.GetByID(ctx, questionID); err != nil {
		return nil, err
	}

	//去除首尾的空格，避免把无意义的东西保存到数据库里面
	content = strings.TrimSpace(content)
	//规定评论内容长度 1-32 字符，超出范围直接返回错误
	if n := utf8.RuneCountInString(content); n < 1 || n > 32 {
		return nil, model.ErrInvalidReplyContent
	}

	//查询当前的用户信息
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	//创建评论模型
	rp := &model.Reply{QuestionID: questionID, UserID: userID, Content: content}
	//写入数据库
	if err := s.replyRepo.Create(ctx, rp); err != nil {
		return nil, err
	}

	//组装对外的评论结构体
	return &ReplyView{
		ID:        rp.ID,
		Content:   rp.Content,
		CreatedAt: rp.CreatedAt,
		IsMine:    true, //刚发的必然是自己的
		Author:    model.NewUserBrief(u),
	}, nil
}
