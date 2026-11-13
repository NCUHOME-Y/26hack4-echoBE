package model

import "time"

// Question问题主表，对应 questions
type Question struct {
	ID            int64      `gorm:"primaryKey" json:"id"`
	AskerID       int64      `json:"asker_id"`
	Description   string     `json:"description"`
	Status        string     `json:"status"` //只取 open，closed，contested
	ExpiresAt     time.Time  `json:"expires_at"`
	SettledAt     *time.Time `json:"settled_at"`      //空表示尚未结算，结算函数靠它做幂等
	ContestedAt   *time.Time `json:"contested_at"`    //空表示不是争议题，判断争议一律看这个字段
	FinalOptionID *int64     `json:"final_option_id"` //空表示没有最终选择
	OptionCount   int16      `json:"option_count"`    //冗余列，CHECK 只允许 2/3/4
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (Question) TableName() string { return "questions" }

//定义状态常量，和数据库 status 的 CHECK 保持一致
const (
	StatusOpen   = "open"
	StatusClosed = "closed"
)

// 是否已经结算过
func (q *Question) IsSettled() bool { return q.SettledAt != nil }

// 截止时间是否已过
func (q *Question) IsExpired() bool { return q.ExpiresAt.Before(time.Now()) }

// 还在作答窗口内：未关闭且未过期（已结算的题 status 一定是 closed，不用再判一次）
func (q *Question) IsLive() bool { return q.Status == StatusOpen && !q.IsExpired() }
