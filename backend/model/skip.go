package model

import "time"

// QuestionSkip 用户跳过题目表，对应数据库的 question_skips 表
type QuestionSkip struct {
	QuestionID int64     `gorm:"primaryKey" json:"question_id"`
	UserID     int64     `gorm:"primaryKey" json:"user_id"`
	SkippedAt  time.Time `json:"skipped_at"`
} //一个用户对一个问题只能有一条跳过记录，保证跳过后不会再次遇到该题目

func (QuestionSkip) TableName() string { return "question_skips" }
