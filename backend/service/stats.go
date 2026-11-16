package service

import (
	"context"
	"time"

	"echo/repository"
)

// cstZone 是「今日」的口径：固定 +08:00，不是 time.Local。
// 用 time.Local 的话，进程跑在 UTC 容器里时区就变成 UTC，北京时间 00:00-08:00 之间发布的问题、
// 投的票、结的算全被算成昨天，而且不报错、数字看着也正常。契约写的是 +08:00 的日历日，照它来
var cstZone = time.FixedZone("CST", 8*3600)

// StatsService 首页看板的今日统计
type StatsService struct {
	questionRepo *repository.QuestionRepository //今日发布量、今日结算量
	voteRepo     *repository.VoteRepository     //今日投票量
}

func NewStatsService(questionRepo *repository.QuestionRepository, voteRepo *repository.VoteRepository) *StatsService {
	return &StatsService{
		questionRepo: questionRepo,
		voteRepo:     voteRepo,
	}
}

// DailyStats 今日三项统计，字段名与契约一致
type DailyStats struct {
	QuestionsToday int64 `json:"questions_today"`
	VotesToday     int64 `json:"votes_today"`
	SettledToday   int64 `json:"settled_today"`
}

// Daily 统计今日发布量、投票量、结算量。
// 结算量按 settled_at（结算动作发生的时间）算，不是 expires_at（到期时间）
func (s *StatsService) Daily(ctx context.Context) (*DailyStats, error) {
	start, end := todayRange(time.Now())

	questions, err := s.questionRepo.CountCreatedBetween(ctx, start, end)
	if err != nil {
		return nil, err
	}
	votes, err := s.voteRepo.CountBetween(ctx, start, end)
	if err != nil {
		return nil, err
	}
	settled, err := s.questionRepo.CountSettledBetween(ctx, start, end)
	if err != nil {
		return nil, err
	}

	return &DailyStats{QuestionsToday: questions, VotesToday: votes, SettledToday: settled}, nil
}

// todayRange 返回包含 t 的那个 +08:00 日历日的 [当日零点, 次日零点)，左闭右开。
// t 做成参数是为了能被直接测试：传进任意时刻都能验出它归哪个北京日，不依赖现在几点
func todayRange(t time.Time) (time.Time, time.Time) {
	local := t.In(cstZone)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, cstZone)
	return start, start.AddDate(0, 0, 1)
}
