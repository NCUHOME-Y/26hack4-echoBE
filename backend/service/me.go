package service

import (
	"context"
	"regexp"

	"echo/model"
	"echo/repository"
)

type MeService struct {
	userRepo *repository.UserRepository
	meRepo   *repository.MeRepository
}

func NewMeService(
	userRepo *repository.UserRepository,
	meRepo *repository.MeRepository,
) *MeService {
	return &MeService{
		userRepo: userRepo,
		meRepo:   meRepo,
	}
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

// avatarPattern 必须和数据库的 users_avatar_check 逐字符一致，
// 否则非法值会穿过应用层、被数据库拦成 500 而不是 400
var avatarPattern = regexp.MustCompile(`^avatar-[0-9]{1,3}$`)

// UpdateAvatar换新头像 直接生效的
func (s *MeService) UpdateAvatar(ctx context.Context, userID int64, avatar string) error {
	if !avatarPattern.MatchString(avatar) {
		return model.ErrInvalidAvatar
	}
	return s.userRepo.UpdateAvatarReq(ctx, userID, map[string]interface{}{"avatar": avatar})
}

// ListMyQuestions 查询我发布的问题，page/pageSize 已由 handler 层的 parsePage 统一校验
func (s *MeService) ListMyQuestions(
	ctx context.Context,
	userID int64,
	page, pageSize int,
) (*model.MyQuestionsData, error) {

	offset := (page - 1) * pageSize
	list, total, err := s.meRepo.ListMyQuestions(
		ctx,
		userID,
		pageSize,
		offset,
	)
	if err != nil {
		return nil, err
	}

	return &model.MyQuestionsData{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// ListMyAnswers 查询我投过票的问题 page/pageSize 已由 handler 层的 parsePage 统一校验
func (s *MeService) ListMyAnswers(
	ctx context.Context,
	userID int64,
	page, pageSize int,
	keyword string,
) (*model.MyAnswersData, error) {

	offset := (page - 1) * pageSize
	list, total, err := s.meRepo.ListMyAnswers(
		ctx,
		userID,
		keyword,
		pageSize,
		offset,
	)
	if err != nil {
		return nil, err
	}

	return &model.MyAnswersData{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}
