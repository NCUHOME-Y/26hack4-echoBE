package model

import "time"

// Vote 投票表，对应数据库的 votes 表
type Vote struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	QuestionID int64     `json:"question_id"`
	UserID     int64     `json:"user_id"`
	OptionID   int64     `json:"option_id"` //复合外键保证它属于同一道题
	CreatedAt  time.Time `json:"created_at"`
}

func (Vote) TableName() string { return "votes" }

// 提交投票的请求体，question_id 在URL路径上
type CastVoteReq struct {
	OptionID int64 `json:"option_id" binding:"required"`
}
