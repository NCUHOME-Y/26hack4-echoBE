package service

import (
	"context"
	"errors"

	"echo/model"

	"golang.org/x/crypto/bcrypt"
)

func (s *UserService) Login(ctx context.Context, req model.LoginRequest) (*model.User, error) {
	u, err := s.repo.GetByAccount(ctx, req.Account)
	if errors.Is(err, model.ErrUserNotFound) {
		return nil, model.ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil { //CompareHashAndPassword会自动比较
		return nil, model.ErrInvalidCredentials
	}

	u.FillLevel()

	return u, nil
}
