package model

import "time"

// Reply 评论区表，对应数据库的 replies 表
type Reply struct {
	ID         int64      `gorm:"primaryKey" json:"id"`
	QuestionID int64      `json:"question_id"`
	UserID     int64      `json:"user_id"`
	Content    string     `json:"content"` //上限 32 字符，CHECK 和列宽一起拦
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	DeletedAt  *time.Time `json:"-"` //对外不返回，只用来过滤
}

func (Reply) TableName() string { return "replies" }

// CreateReplyReq 创建评论请求体，question_id放在URL路径中
type CreateReplyReq struct {
	Content string `json:"content" binding:"required,min=1,max=32"` //validator 按字符数计，和 char_length 一致
}
