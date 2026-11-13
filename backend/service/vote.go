package service

import "echo/repository"

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
