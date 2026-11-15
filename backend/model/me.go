package model

import (
	"time"
)

// UpdateAvatarReq换头像请求体，字段名与 User.avatar 和数据库列 avatar 保持一致
// binding 只负责必填和长度上限（32 与列宽一致），avatar-N 的格式规则在 service 层
type UpdateAvatarReq struct {
	Avatar string `json:"avatar" binding:"required,max=32"`
}

// MyQuestionItem 我的问题项列表
type MyQuestionItem struct {
	ID          int64      `json:"id"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	ExpiresAt   time.Time  `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
	SettledAt   *time.Time `json:"settled_at"`
	ContestedAt *time.Time `json:"contested_at"`
	TotalVotes  int64      `json:"total_votes"`
}
type MyQuestionsData struct {
	List     []MyQuestionItem `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// MyAnswerItem 我投过票的问题记录
type MyAnswerItem struct {
	QuestionID    int64      `json:"question_id"`
	Description   string     `json:"description"`
	Status        string     `json:"status"`
	MyOptionID    int64      `json:"my_option_id"`
	MyOptionLabel string     `json:"my_option_label"`
	FinalOptionID *int64     `json:"final_option_id"`
	Matched       *bool      `json:"matched"`
	VotedAt       time.Time  `json:"voted_at"`
	SettledAt     *time.Time `json:"settled_at"`
}

type MyAnswersData struct {
	List     []MyAnswerItem `json:"list"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"page_size"`
}
