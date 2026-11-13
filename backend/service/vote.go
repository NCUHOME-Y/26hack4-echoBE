package service

import (
	"context"
	"math"

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

// VoteResult 投票成功的响应
type VoteResult struct {
	QuestionID int64        `json:"question_id"`
	OptionID   int64        `json:"option_id"`
	TotalVotes int64        `json:"total_votes"`
	Options    []OptionView `json:"options"`
}

// OptionView 一个选项的票数与占比，0 票的选项也要在数组里
type OptionView struct {
	ID      int64   `json:"id"`
	Label   string  `json:"label"`
	SortNo  int16   `json:"sort_no"`
	Votes   int64   `json:"votes"`
	Percent float64 `json:"percent"`
}

// CastVote 校验、插票、聚合占比
func (s *VoteService) CastVote(ctx context.Context, questionID, userID, optionID int64) (*VoteResult, error) {
	//查询问题
	q, err := s.questionRepo.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}

	//查询这个问题现在能不能投票
	if !q.IsLive() {
		return nil, model.ErrQuestionClosed
	}

	//禁止给自己提的题投票
	if q.AskerID == userID {
		return nil, model.ErrOwnQuestionVote
	}

	//选项必须属于这道题
	opts, err := s.optionRepo.ListByQuestion(ctx, questionID)
	if err != nil {
		return nil, err
	}
	//判断提交的选项是否属于这道题
	matched := false
	for _, o := range opts {
		if o.ID == optionID {
			matched = true
			break
		}
	}
	if !matched {
		return nil, model.ErrOptionNotInQuestion
	}

	//查我是不是已经投过，避免重复投票
	v, err := s.voteRepo.GetByQuestionAndUser(ctx, questionID, userID)
	if err != nil {
		return nil, err
	}
	if v != nil {
		return nil, model.ErrAlreadyVoted
	}

	//查询用户是否跳过
	skipped, err := s.skipRepo.Exists(ctx, questionID, userID)
	if err != nil {
		return nil, err
	}
	if skipped {
		return nil, model.ErrQuestionSkipped
	}

	//真正的投票操作
	if err := s.voteRepo.Cast(ctx, questionID, userID, optionID, s.skipRepo); err != nil {
		return nil, err
	}

	//统计最新的票数情况
	counts, err := s.voteRepo.CountByOption(ctx, questionID)
	if err != nil {
		return nil, err
	}
	var total int64
	for _, n := range counts {
		total += n
	}

	//组装返回数据
	views := make([]OptionView, 0, len(opts))
	for _, o := range opts {
		n := counts[o.ID] //没有票的选项在 map 里不存在，取出来正好是 0
		views = append(views, OptionView{
			ID:      o.ID,
			Label:   o.Label,
			SortNo:  o.SortNo,
			Votes:   n,
			Percent: percent(n, total),
		})
	}

	return &VoteResult{
		QuestionID: questionID,
		OptionID:   optionID,
		TotalVotes: total,
		Options:    views,
	}, nil
}

// percent 保留一位小数。total 为 0 时直接返回 0，不做除法
func percent(votes, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return math.Round(float64(votes)*1000/float64(total)) / 10
}
