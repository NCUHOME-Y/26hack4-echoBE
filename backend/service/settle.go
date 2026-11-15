package service

import (
	"context"

	"echo/repository"
)

// 争议判定的两个门槛
const (
	contestedMinVotes    = 30 //总票数大于30
	contestedMaxGapScore = 15 //前两名百分比相差小于15
)

// SettleService 到期自动结算的业务规则层
type SettleService struct {
	questionRepo *repository.QuestionRepository //取题、落结算状态
	voteRepo     *repository.VoteRepository     //数每个选项的票
}
 
//注入这个业务所需要的repository
func NewSettleService(questionRepo *repository.QuestionRepository, voteRepo *repository.VoteRepository) *SettleService {
	return &SettleService{
		questionRepo: questionRepo,
		voteRepo:     voteRepo,
	}
}

// SettleOutcome 一次结算的结果
type SettleOutcome struct {
	QuestionID    int64
	FinalOptionID *int64 //空表示这题还没定：无人投票或最高票数平票，留给提问者补公布
	Contested     bool
	Already       bool //这题之前是否就被结算过
	Won           bool //本次 UPDATE 是否真正生效
}

// SettleQuestion 结算一道到期的题
func (s *SettleService) SettleQuestion(ctx context.Context, questionID int64) (*SettleOutcome, error) {
	//查询问题
	q, err := s.questionRepo.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}

	//快路径，已结算直接返回
	if q.IsSettled() {
		return &SettleOutcome{QuestionID: questionID, Already: true}, nil
	}

	//统计票数
	counts, err := s.voteRepo.CountByOption(ctx, questionID)
	if err != nil {
		return nil, err
	}

	//一次遍历同时求出总票数、第一名和第二名。
	var total, top, second int64
	for _, n := range counts {
		total += n
		switch {
		case n > top:
			second = top
			top = n
		case n > second:
			second = n
		}
	}

	//第一名唯一才定得出来；平票或者根本没人投，就留空给提问者
	var finalOptionID *int64
	if total > 0 && top > second {
		//找出最终选项ID
		for id, n := range counts {
			if n == top {
				v := id
				finalOptionID = &v
				break
			}
		}
	}

	//判断是否进入争议
	contested := total > contestedMinVotes && gapScore(top, second, total) <= contestedMaxGapScore

	//写入结算结果
	won, err := s.questionRepo.MarkSettled(ctx, questionID, finalOptionID, contested)
	if err != nil {
		return nil, err
	}

	//返回结果
	return &SettleOutcome{
		QuestionID:    questionID,
		FinalOptionID: finalOptionID,
		Contested:     contested,
		Won:           won,
	}, nil
}

// gapScore 第一名和第二名差多少个百分点
func gapScore(top, second, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(top-second) * 100 / float64(total)
}
