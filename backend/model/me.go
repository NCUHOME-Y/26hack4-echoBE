package model

type UpdateAvatarReq struct {
	AvatarID string `json:"avatar_id" binding:"required,max=32"`
}
