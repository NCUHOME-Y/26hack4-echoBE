package service

import (
	"context"

	"echo/model"
	"echo/repository"
)

// SkipService 放弃作答权的业务规则层
type SkipService struct {
	questionRepo *repository.QuestionRepository //判题在不在
	voteRepo     *repository.VoteRepository     //判这题我投过没有
	skipRepo     *repository.SkipRepository     //落跳过记录
}

//组装上述三个依赖
func NewSkipService(questionRepo *repository.QuestionRepository, voteRepo *repository.VoteRepository,
	skipRepo *repository.SkipRepository) *SkipService {
	return &SkipService{
		questionRepo: questionRepo,
		voteRepo:     voteRepo,
		skipRepo:     skipRepo,
	}
}

// SkipResult 跳过的响应体
type SkipResult struct {
	QuestionID int64 `json:"question_id"`
	Skipped    bool  `json:"skipped"`
}

// Skip 校验后落下跳过记录。first 为 false 表示之前就跳过过，不是错误
func (s *SkipService) Skip(ctx context.Context, questionID, userID int64) (*SkipResult, bool, error) {
	//查询问题，不存在返回 ErrQuestionNotFound
	q, err := s.questionRepo.GetByID(ctx, questionID)
	if err != nil {
		return nil, false, err
	}

	//自己提的题本来就不在作答池里
	if q.AskerID == userID {
		return nil, false, model.ErrOwnQuestionSkip
	}

	//已经投过这题就不能再放弃作答权
	v, err := s.voteRepo.GetByQuestionAndUser(ctx, questionID, userID)
	if err != nil {
		return nil, false, err
	}
	if v != nil {
		return nil, false, model.ErrAlreadyVotedSkip
	}

	//创建或重复确认跳过记录
	first, err := s.skipRepo.CreateOrTouch(ctx, questionID, userID)
	if err != nil {
		return nil, false, err
	}

	return &SkipResult{QuestionID: questionID, Skipped: true}, first, nil
}
