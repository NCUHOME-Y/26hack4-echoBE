package model

import "time"

// 用户的基本信息
type User struct {
	ID           int64     `gorm:"primaryKey" json:"id"`
	Account      string    `json:"account"`
	Nickname     string    `json:"nickname"`
	PasswordHash string    `json:"-"` //当以json格式输出时，忽略该字段，保证密码哈希值不会被泄露
	Avatar       string    `json:"avatar"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// 注册的请求体，只声明客户端该传的字段
type RegisterRequest struct {
	Account  string `json:"account" binding:"required,min=3,max=64"`//binding:"required"表示该字段是必须的
	Nickname string `json:"nickname" binding:"required,max=64"`
	Password string `json:"password" binding:"required,min=8,max=64,alphanum"` //alphanum默认正则表达式，只允许大小写字母和数字，保证64字节，不会撞上bcrypt的72字节上限
}

//登录的请求体
type LoginRequest struct {
	Account  string `json:"account" binding:"required"`
	Password string `json:"password" binding:"required"`
}
