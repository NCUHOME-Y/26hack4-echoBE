package repository

import (
	"context"
	"errors"
	"fmt"

	"echo/model"

	"gorm.io/gorm"
)

type QuestionRepository struct {
	db *gorm.DB
}

func NewQuestionRepository(db *gorm.DB) *QuestionRepository {
	return &QuestionRepository{db: db}
}

// GetByID 根据问题 ID 查询问题
func (r *QuestionRepository) GetByID(ctx context.Context, id int64) (*model.Question, error) {
	var q model.Question
	err := r.db.WithContext(ctx).First(&q, id).Error //主键查询走 First 的字符串/整数简写形式
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, model.ErrQuestionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查询问题失败: %w", err)
	}
	return &q, nil
}
