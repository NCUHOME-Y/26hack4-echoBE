package repository

import (
	"context"
	"errors"
	"fmt"

	"echo/model"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create 创建新的用户
func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	err := r.db.WithContext(ctx).Create(u).Error //数据库会把iD和CreatedAt、UpdatedAt字段自动填充到User结构体中
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return model.ErrAccountTaken //调用哨兵错误，返回给上层，方便上层进行错误判断
	}
	if err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}
	return nil
}
// GetByAccount 根据账号查询用户
func (r *UserRepository) GetByAccount(ctx context.Context, account string) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).Where("account = ?", account).First(&u).Error //?作为占位符，防止SQL注入
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &u, nil
}