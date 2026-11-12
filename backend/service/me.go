package service

import (
	"context"
	"echo/model"
	"echo/repository"
)

type MeService struct {
	userRepo *repository.UserRepository
}

func NewMeService(userRepo *repository.UserRepository) *MeService {
	return &MeService{userRepo: userRepo}
}

// 查资料+算等级
func (s *MeService) GetProfile(ctx context.Context, userID int64) (*model.User, error) {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	u.FillLevel() //exp->level/level_progress
	return u, nil
}

// UpdateAvatar换新头像 直接生效的
func (s *MeService) UpdateAvatar(ctx context.Context, userID int64, avatar string) error {
	//只接受avatar-N格式 防止随便塞个URL进来
	if len(avatar) < 8 || avatar[:7] != "avatar-" {
		return model.ErrInvalidAvatar
	}
	return s.userRepo.UpdateAvatarReq(ctx, userID, map[string]interface{}{"avatar": avatar})
}
