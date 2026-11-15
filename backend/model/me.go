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
