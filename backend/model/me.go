package model

// UpdateAvatarReq换头像请求体，字段名与 User.avatar 和数据库列 avatar 保持一致
// binding 只负责必填和长度上限（32 与列宽一致），avatar-N 的格式规则在 service 层
type UpdateAvatarReq struct {
	Avatar string `json:"avatar" binding:"required,max=32"`
}
