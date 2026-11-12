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
