package repository

import (
	"context"
	"fmt"

	"echo/model"

	"gorm.io/gorm"
)

//保存数据库链接（该目录下的这种代码都是这个意思）
type OptionRepository struct {
	db *gorm.DB
}

// NewOptionRepository 创建 OptionRepository（该目录下的这种代码都是这个意思）	
func NewOptionRepository(db *gorm.DB) *OptionRepository {
	return &OptionRepository{db: db}
}

// ListByQuestion 根据问题ID查询选项列表
func (r *OptionRepository) ListByQuestion(ctx context.Context, questionID int64) ([]model.Option, error) {
	var opts []model.Option
	err := r.db.WithContext(ctx).Where("question_id = ?", questionID).Order("sort_no").Find(&opts).Error
	if err != nil {
		return nil, fmt.Errorf("查询选项失败: %w", err)
	}
	return opts, nil
}

// ListByQuestions 一次查询多道题的选项，提高查询效率
func (r *OptionRepository) ListByQuestions(ctx context.Context, questionIDs []int64) ([]model.Option, error) {
	//空的问题ID直接返回空列表，避免 SQL 语法错误
	if len(questionIDs) == 0 {
		return []model.Option{}, nil
	}
	var opts []model.Option
	err := r.db.WithContext(ctx).Where("question_id IN ?", questionIDs). //批量查询
		Order("question_id, sort_no").Find(&opts).Error
	if err != nil {
		return nil, fmt.Errorf("批量查询选项失败: %w", err)
	}
	return opts, nil
}
