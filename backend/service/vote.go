package service

import (
	"context"

	"echo/model"
	"echo/repository"
)

// VoteService 投票与实时占比的业务规则层
type VoteService struct {
	questionRepo *repository.QuestionRepository //判题在不在池子里
	optionRepo   *repository.OptionRepository   //判选项归属 + 组装占比
	voteRepo     *repository.VoteRepository     //查我的票、插票、聚合票数
	skipRepo     *repository.SkipRepository     //查是否已放弃作答权
}

func NewVoteService(questionRepo *repository.QuestionRepository, optionRepo *repository.OptionRepository,
	voteRepo *repository.VoteRepository, skipRepo *repository.SkipRepository) *VoteService {
	return &VoteService{
		questionRepo: questionRepo,
		optionRepo:   optionRepo,
		voteRepo:     voteRepo,
		skipRepo:     skipRepo,
	}
}

// CastVote 提交投票。步 2 只做"这道题现在能不能被我投"，还不查选项归属、不插票、不算占比
func (s *VoteService) CastVote(ctx context.Context, questionID, userID, optionID int64) error {
	//查询问题
	q, err := s.questionRepo.GetByID(ctx, questionID)
	if err != nil {
		return err
	}

	//查询这个问题现在能不能投票
	if !q.IsLive() {
		return model.ErrQuestionClosed
	}

	//禁止给自己提的题投票
	if q.AskerID == userID {
		return model.ErrOwnQuestionVote
	}

	return nil
}
