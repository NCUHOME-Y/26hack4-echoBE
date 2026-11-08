package model

// 用户的基本信息
type User struct {
	Nickname string `json:"nickname"`
	Account  string `json:"account"`
	Password string `json:"password"`
	Avatar   string `json:"avatar"`
}
