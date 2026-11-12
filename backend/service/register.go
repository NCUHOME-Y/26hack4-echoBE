package service

import (
	"context"
	"fmt"

	"echo/model"

	"golang.org/x/crypto/bcrypt"
)

func (s *UserService) Register(ctx context.Context, req model.RegisterRequest) (*model.User, error) {
	//对密码进行bcrypt哈希处理
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码哈希失败: %w", err)
	}

	//创建一个新的用户对象
	u := &model.User{
		Account:      req.Account,
		Nickname:     req.Nickname,
		PasswordHash: string(hash),
	}

	//保存到数据库
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	u.FillLevel()

	return u, nil
}
