package service

import (
	"context"
	"fmt"
	"time"

	"echo/middleware"
	"echo/model"
	"echo/repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
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
	voteRepo     *repository.VoteRepository     //数每个选项的票、取投票明细
	optionRepo   *repository.OptionRepository   //判提交的选项属不属于这道题
	expRepo      *repository.ExpRepository      //结算这一刻发积分
}

// 注入这个业务所需要的repository
func NewSettleService(questionRepo *repository.QuestionRepository, voteRepo *repository.VoteRepository,
	optionRepo *repository.OptionRepository, expRepo *repository.ExpRepository) *SettleService {
	return &SettleService{
		questionRepo: questionRepo,
		voteRepo:     voteRepo,
		optionRepo:   optionRepo,
		expRepo:      expRepo,
	}
}

// SettleOutcome 一次结算的结果
type SettleOutcome struct {
	QuestionID    int64
	FinalOptionID *int64 //自动结算出来恒为空，只有提问者公布之后才有值
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

	//一次遍历求出票数分布，只为争议判定服务；自动结算不替提问者做选择
	t := tallyVotes(counts)
	var finalOptionID *int64 //最终选择一律留空，等提问者用 #13 公布
	contested := t.contested()

	//写入结算结果，并在同一个事务里发积分
	won, err := s.questionRepo.SettleWith(ctx, questionID, finalOptionID, contested, func(tx *gorm.DB) error {
		return s.grantSettleExp(tx, ctx, q.AskerID, questionID, finalOptionID)
	})
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

// settleReward 结算这一刻，某个投票者该拿哪一档：方案里的 25 / 5 / 15
func settleReward(votedOptionID int64, finalOptionID *int64) (reason string, delta int) {
	//平票或没人投票，还没定下来
	if finalOptionID == nil {
		return model.ExpReasonSettleFailed, model.ExpSettleFailed
	}

	//投中了最终答案
	if votedOptionID == *finalOptionID {
		return model.ExpReasonSettleMatch, model.ExpSettleMatch
	}

	//没有投中最终答案
	return model.ExpReasonSettleMiss, model.ExpSettleMiss
}

// grantSettleExp 在结算事务内发积分：提问者一份 settle_ask（只在他本人定出最终选择时），每个投票者按选项归属一份。
// 投票明细要在事务里读，而不是进事务前读——UPDATE 已经拿到这道题的行锁，
// 此时读到的名单和写进去的结算结果才是同一份事实。
// 重复发放由 exp_entries 的唯一约束 (user_id, source_type, source_id, reason) 拦在库层，
// 所以这里不需要先查后插。
func (s *SettleService) grantSettleExp(tx *gorm.DB, ctx context.Context, askerID int64, questionID int64, finalOptionID *int64) error {
	//提问者那份 40 只在最终选择是他定出来的时候才发；到期时没定，这题就没有结算奖励
	if finalOptionID != nil {
		if err := s.expRepo.GrantTx(tx, &model.ExpEntry{
			UserID:     askerID,
			Delta:      model.ExpSettleAsk,
			Reason:     model.ExpReasonSettleAsk,
			SourceType: model.ExpSourceQuestion,
			SourceID:   questionID,
		}); err != nil {
			return err
		}
	}

	//查询所有的投票
	votes, err := s.voteRepo.WithTx(tx).ListByQuestion(ctx, questionID)
	if err != nil {
		return err
	}

	//给所有投票者发经验
	for _, v := range votes {
		reason, delta := settleReward(v.OptionID, finalOptionID)
		if err := s.expRepo.GrantTx(tx, &model.ExpEntry{
			UserID:     v.UserID,
			Delta:      delta,
			Reason:     reason,
			SourceType: model.ExpSourceQuestion,
			SourceID:   questionID,
		}); err != nil {
			return err
		}
	}
	return nil
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
			middleware.L().Error("结算任务：单题结算失败", zap.Int64("question_id", list[i].ID), zap.Error(err))
			continue
		} //单题数据异常不会阻塞后面的题目
		if !outcome.Won {
			continue //被别人抢先结算过了，不算本轮的产出
		}
		won++
		middleware.L().Info("结算任务：已结算",
			zap.Int64("question_id", list[i].ID),
			zap.Int64p("final_option_id", outcome.FinalOptionID), //指针字段：没定成就是 null，不用手工拼 "NULL"
			zap.Bool("contested", outcome.Contested),
		)
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
			middleware.L().Info("结算任务：收到退出信号，停止扫描")
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
		middleware.L().Error("结算任务：一轮扫描失败", zap.String("trigger", why), zap.Error(err))
		return
	}
	if won > 0 {
		middleware.L().Info("结算任务：完成一轮结算", zap.String("trigger", why), zap.Int("settled", won))
	}
}

// tally 一道题的票数分布结论
type tally struct {
	total  int64
	top    int64
	second int64
}

// tallyVotes 一次求出总量、第一名票数、第二名票数
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
	return t
}

// contested 是否进争议海螺
func (t tally) contested() bool {
	return t.total >= contestedMinVotes && gapScore(t.top, t.second, t.total) <= contestedMaxGapScore
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

	//最终选择只由提问者本人在题目还开着的时候定
	if q.FinalOptionID != nil {
		return nil, model.ErrFinalAlreadySet
	}
	if !q.IsLive() { //到期或被结算过就不再开放，这题永久停在"没有最终选择"
		return nil, model.ErrFinalWindowClosed
	}

	//公布即结算：写入他指定的选项，同时打 settled_at、关闭题目、同一个事务里发积分
	published := optionID
	counts, err := s.voteRepo.CountByOption(ctx, questionID)
	if err != nil {
		return nil, err
	}
	won, err := s.questionRepo.SettleWith(ctx, questionID, &published, tallyVotes(counts).contested(), func(tx *gorm.DB) error {
		return s.grantSettleExp(tx, ctx, q.AskerID, questionID, &published)
	})
	if err != nil {
		return nil, err
	}
	if !won {
		return nil, s.recheckFinal(ctx, questionID)
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
	if q.IsSettled() { //被后台结算抢先关掉了，窗口已经关了
		return model.ErrFinalWindowClosed
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
