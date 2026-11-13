package model

import "time"

// Option 选项表，对应数据库的 options 表
type Option struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	QuestionID int64     `json:"question_id"`
	Label      string    `json:"label"` //上限 18 字符，由列宽拦
	SortNo     int16     `json:"sort_no"` //展示顺序
	CreatedAt  time.Time `json:"created_at"`
	//以下两个不属于数据库里面的字段，是统计结果的时候动态计算的
	Votes   int64   `gorm:"-" json:"votes"`
	Percent float64 `gorm:"-" json:"percent"` //0 到 100，保留一位小数
}

func (Option) TableName() string { return "options" } //告诉gorm使用options表
