package repository

import (
	"context"
	"fmt"
	"time"

	"echo/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SkipRepository struct {
	db *gorm.DB
}

func NewSkipRepository(db *gorm.DB) *SkipRepository {
	return &SkipRepository{db: db}
}

// Exists 检查用户是否已经跳过该题
func (r *SkipRepository) Exists(ctx context.Context, questionID, userID int64) (bool, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&model.QuestionSkip{}).
		Where("question_id = ? AND user_id = ?", questionID, userID).
		Count(&n).Error
	if err != nil {
		return false, fmt.Errorf("查询跳过记录失败: %w", err)
	}
	return n > 0, nil
}

// WithTx 返回一个绑到事务的副本
func (r *SkipRepository) WithTx(tx *gorm.DB) *SkipRepository {
	return &SkipRepository{db: tx}
}

// CreateOrTouch 落下跳过记录，返回是否首次插入。
func (r *SkipRepository) CreateOrTouch(ctx context.Context, questionID, userID int64) (bool, error) {
	res := r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}). //冲突不会报错，因为冲突代表已经跳过过了
		Create(&model.QuestionSkip{QuestionID: questionID, UserID: userID, SkippedAt: time.Now()})
	if res.Error != nil {
		return false, fmt.Errorf("写入跳过记录失败: %w", res.Error)
	}
	return res.RowsAffected > 0, nil //判断是否为第一次跳过
}
