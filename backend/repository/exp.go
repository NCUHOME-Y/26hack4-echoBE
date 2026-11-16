package repository

import (
	"context"
	"fmt"

	"echo/model"

	"gorm.io/gorm"
)

type ExpRepository struct {
	db *gorm.DB
}

func NewExpRepository(db *gorm.DB) *ExpRepository {
	return &ExpRepository{db: db}
}

// GrantTx在调用方的事务内记一条流水并累加余额
func (r *ExpRepository) GrantTx(tx *gorm.DB, e *model.ExpEntry) error {
	//exp_entries 的唯一约束 (user_id, source_type, source_id, reason) 拦重复发放
	if err := tx.Create(e).Error; err != nil {
		return fmt.Errorf("记录经验流水失败: %w", err)
	}
	if err := tx.Model(&model.User{}).Where("id = ?", e.UserID).
		UpdateColumn("exp", gorm.Expr("exp + ?", e.Delta)).Error; err != nil {
		return fmt.Errorf("累加用户经验失败: %w", err)
	}
	return nil
}

// Grant给没有现成事务的调用方用（投票、结算会用到）
func (r *ExpRepository) Grant(ctx context.Context, e *model.ExpEntry) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return r.GrantTx(tx, e)
	})
}
