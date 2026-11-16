package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"echo/model"
	"echo/repository"
)

// 争议判定的两个门槛
const (
	contestedMinVotes    = 30 //总票数大于30
	contestedMaxGapScore = 15 //前两名百分比相差小于15
)

// 结算任务的节奏：30 秒一轮，一轮最多处理 50 题。
const (
	settleInterval = 30 * time.Second
	settleBatch    = 50
)

// SettleService 到期自动结算与提问者公布的业务规则层
type SettleService struct {
	questionRepo *repository.QuestionRepository //取题、落结算状态、补公布最终选择
	voteRepo     *repository.VoteRepository     //数每个选项的票
	optionRepo   *repository.OptionRepository   //判提交的选项属不属于这道题
}

//注入这个业务所需要的repository
func NewSettleService(questionRepo *repository.QuestionRepository, voteRepo *repository.VoteRepository,
	optionRepo *repository.OptionRepository) *SettleService {
	return &SettleService{
		questionRepo: questionRepo,
		voteRepo:     voteRepo,
		optionRepo:   optionRepo,
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

	//一次遍历求出票数分布。自动结算和提问者公布用的是同一个函数，两套判定不可能分叉
	t := tallyVotes(counts)
	finalOptionID := t.topID //第一名唯一时才有值；平票或者根本没人投就留空给提问者
	contested := t.contested()

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

// SettleDue 扫一轮：捞出到期未结算的题逐题结算，返回真正写入的题数
func (s *SettleService) SettleDue(ctx context.Context, limit int) (int, error) {
	//查询未结算的题目
	list, err := s.questionRepo.ListSettleable(ctx, limit)
	if err != nil {
		return 0, err
	}

	var won int
	for i := range list {
		outcome, err := s.SettleQuestion(ctx, list[i].ID)
		if err != nil {
			log.Printf("结算任务：单题结算失败 questionID=%d err=%v", list[i].ID, err)
			continue
		} //单题数据异常不会阻塞后面的题目
		if !outcome.Won {
			continue //被别人抢先结算过了，不算本轮的产出
		}
		won++
		final := "NULL"
		if outcome.FinalOptionID != nil {
			final = fmt.Sprint(*outcome.FinalOptionID)
		}
		log.Printf("结算任务：已结算 questionID=%d final=%s contested=%v", list[i].ID, final, outcome.Contested)
	}
	return won, nil
}

// Run 后台结算循环：启动先跑一轮补齐欠账
func (s *SettleService) Run(ctx context.Context) {
	//创建定时器，每30秒一次
	ticker := time.NewTicker(settleInterval)
	defer ticker.Stop()

	s.round(ctx, "启动")
	//循环跑一轮，直到收到退出信号
	for {
		select {
		case <-ctx.Done():
			log.Println("结算任务：收到退出信号，停止扫描")
			return
		case <-ticker.C:
			s.round(ctx, "定时")
		}
	}
}

// round 跑一轮
func (s *SettleService) round(ctx context.Context, why string) {
	won, err := s.SettleDue(ctx, settleBatch)
	if err != nil { //查询失败只打日志，不终止循环
		log.Printf("结算任务：%s一轮扫描失败 err=%v", why, err)
		return
	}
	if won > 0 {
		log.Printf("结算任务：%s一轮结算 %d 题", why, won)
	}
}

// tally 一道题的票数分布结论
type tally struct {
	total  int64
	top    int64
	second int64
	topID  *int64 //第一名票数唯一时对应的选项；平票或没人投票时为 nil
}

// tallyVotes 一次求出总量、第一名和第二名
func tallyVotes(counts map[int64]int64) tally {
	var t tally
	for _, n := range counts {
		t.total += n
		switch {
		case n > t.top:
			t.second = t.top
			t.top = n
		case n > t.second:
			t.second = n
		}
	}
	//第一名唯一才定得出来
	if t.total > 0 && t.top > t.second {
		for id, n := range counts {
			if n == t.top {
				v := id
				t.topID = &v
				break
			}
		}
	}
	return t
}

// contested 是否进争议海螺
func (t tally) contested() bool {
	return t.total > contestedMinVotes && gapScore(t.top, t.second, t.total) <= contestedMaxGapScore
}

// FinalResult 公布最终选择的响应体
type FinalResult struct {
	QuestionID    int64 `json:"question_id"`
	FinalOptionID int64 `json:"final_option_id"`
}

// PublishFinal 提问者公布最终选择
func (s *SettleService) PublishFinal(ctx context.Context, questionID, userID, optionID int64) (*FinalResult, error) {
	//查询问题
	q, err := s.questionRepo.GetByID(ctx, questionID) 
	if err != nil {
		return nil, err
	}

	//确认当前用户为提问者
	if q.AskerID != userID { 
		return nil, model.ErrNotAsker
	}

	//确认选项属于当前问题
	opts, err := s.optionRepo.ListByQuestion(ctx, questionID) 
	if err != nil {
		return nil, err
	}
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

	//确认当前问题还没结算
	if q.FinalOptionID != nil {
		return nil, model.ErrFinalAlreadySet
	}

	//只补充最终选项
	published := optionID
	if q.IsSettled() {
		won, err := s.questionRepo.MarkFinalOnly(ctx, questionID, optionID)
		if err != nil {
			return nil, err
		}
		if !won {
			return nil, s.recheckFinal(ctx, questionID)
		}
	} else { //未结算的问题，同时完成公布和结算
		counts, err := s.voteRepo.CountByOption(ctx, questionID)
		if err != nil {
			return nil, err
		}
		won, err := s.questionRepo.MarkSettled(ctx, questionID, &published, tallyVotes(counts).contested())
		if err != nil {
			return nil, err
		}
		if !won {
			return nil, s.recheckFinal(ctx, questionID)
		}
	}

	return &FinalResult{QuestionID: questionID, FinalOptionID: optionID}, nil
}

// recheckFinal 用于处理并发更新失败（这一块是AI加进去的没太搞懂原因）
func (s *SettleService) recheckFinal(ctx context.Context, questionID int64) error {
	q, err := s.questionRepo.GetByID(ctx, questionID)
	if err != nil {
		return err
	}
	if q.FinalOptionID != nil {
		return model.ErrFinalAlreadySet
	}
	return fmt.Errorf("公布最终选择未生效 questionID=%d", questionID)
}

// gapScore 第一名和第二名差多少个百分点
func gapScore(top, second, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(top-second) * 100 / float64(total)
}
