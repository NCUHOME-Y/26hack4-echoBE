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

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*model.User, error) {
	var u model.User
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &u, nil
}

// ListByIDs 根据一组用户ID批量查询用户，增加查询的效率
func (r *UserRepository) ListByIDs(ctx context.Context, ids []int64) (map[int64]*model.User, error) {
	if len(ids) == 0 {
		return map[int64]*model.User{}, nil
	} //若没有用户ID直接返回空map，避免SQL报错
	var list []model.User
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&list).Error //一次性查询出所有用户，IN语句可以传入切片
	if err != nil {
		return nil, fmt.Errorf("批量查询用户失败: %w", err)
	}
	out := make(map[int64]*model.User, len(list))
	for i := range list {
		out[list[i].ID] = &list[i] //取下标而不是范围变量，避免拿到循环副本的地址
	} //将切片转换为map
	return out, nil
}

func (r *UserRepository) UpdateAvatarReq(ctx context.Context, id int64, fields map[string]interface{}) error {
	res := r.db.WithContext(ctx).Model(&model.User{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return fmt.Errorf("更新用户头像失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return model.ErrUserNotFound //没有影响任何 说明这个人不存在
	}
	return nil
}
