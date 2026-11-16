package service

import (
	"context"
	"time"

	"echo/repository"
)

// cstZone 是用来固定时区的，东八区
var cstZone = time.FixedZone("CST", 8*3600)

// StatsService 首页看板的今日统计
type StatsService struct {
	questionRepo *repository.QuestionRepository //今日发布量、今日结算量
	voteRepo     *repository.VoteRepository     //今日投票量
}
//注入这个业务所需要的repository
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

// Daily 统计今日发布量、投票量、结算量
func (s *StatsService) Daily(ctx context.Context) (*DailyStats, error) {
	//计算今日的时间范围
	start, end := todayRange(time.Now())

	//统计今日发布量
	questions, err := s.questionRepo.CountCreatedBetween(ctx, start, end)
	if err != nil {
		return nil, err
	}

	//统计今日投票量
	votes, err := s.voteRepo.CountBetween(ctx, start, end)
	if err != nil {
		return nil, err
	}

	//统计今日结算量
	settled, err := s.questionRepo.CountSettledBetween(ctx, start, end)
	if err != nil {
		return nil, err
	}

	return &DailyStats{QuestionsToday: questions, VotesToday: votes, SettledToday: settled}, nil
}

// todayRange 计算传入时间在北京时间的范围
func todayRange(t time.Time) (time.Time, time.Time) {
	local := t.In(cstZone) //转换到东八区
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, cstZone) //构造当天的零点
	return start, start.AddDate(0, 0, 1) //生成第二天的零点
}
