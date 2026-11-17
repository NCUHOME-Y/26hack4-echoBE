package model

// UserBrief 定义简要的对外用户资料
type UserBrief struct {
	ID       int64  `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Level    int    `json:"level"`
}

// 从 User 生成一份对外资料，等级由 exp 算出
func NewUserBrief(u *User) UserBrief {
	u.FillLevel()
	return UserBrief{ID: u.ID, Nickname: u.Nickname, Avatar: u.Avatar, Level: u.Level}
}
