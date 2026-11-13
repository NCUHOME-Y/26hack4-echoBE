package repository

import (
	"context"
	"fmt"

	"echo/model"

	"gorm.io/gorm"
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
