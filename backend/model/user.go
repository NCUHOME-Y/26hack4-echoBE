package model

import "time"

// 用户的基本信息
type User struct {
	ID            int64     `gorm:"primaryKey" json:"id"`
	Account       string    `json:"account"`
	Nickname      string    `json:"nickname"`
	PasswordHash  string    `json:"-"` //当以json格式输出时，忽略该字段，保证密码哈希值不会被泄露
	Avatar        string    `json:"avatar"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	Exp           int       `json:"exp"`
	Level         int       `gorm:"-" json:"level"`
	LevelProgress int       `gorm:"-" json:"level_progress"`
}

// 注册的请求体，只声明客户端该传的字段
type RegisterRequest struct {
	Account  string `json:"account" binding:"required,min=3,max=64,alphanum"`  //账号 3-64 位，alphanum 拦掉字母数字以外的字符
	Nickname string `json:"nickname" binding:"required,max=7"`                 //PRD 规定昵称最多 7 个字符，中英文都按一个字符算
	Password string `json:"password" binding:"required,min=8,max=64,alphanum"` //alphanum默认正则表达式，只允许大小写字母和数字，保证64字节，不会撞上bcrypt的72字节上限
}

//登录的请求体
type LoginRequest struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}

//对用户的等级做出计算
func (u *User) FillLevel() {
	u.Level = u.Exp / LevelExpUnit
	if u.Level > MaxLevel {
		u.Level = MaxLevel //等级封顶 9999，与前端进度条上限一致
	}
	u.LevelProgress = u.Exp % LevelExpUnit
}

// 经验值流水
type ExpEntry struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	UserID     int64     `json:"user_id"`
	Delta      int       `json:"delta"`
	Reason     string    `json:"reason"`
	SourceType string    `json:"source_type"`
	SourceID   int64     `json:"source_id"`
	CreatedAt  time.Time `json:"created_at"`
}

func (ExpEntry) TableName() string { return "exp_entries" }
