package repository

import (
	"context"
	"fmt"

	"echo/model"

	"gorm.io/gorm"
)

type MeRepository struct {
	db *gorm.DB
}

func NewMeRepository(db *gorm.DB) *MeRepository {
	return &MeRepository{db: db}
}

// ListMyQuestions 查询我发布的问题，并通过标量子查询统计每题总票数
func (r *MeRepository) ListMyQuestions(
	ctx context.Context,
	userID int64,
	limit, offset int,
) ([]model.MyQuestionItem, int64, error) {
	list := make([]model.MyQuestionItem, 0)
	var total int64

	if err := r.db.WithContext(ctx).
		Model(&model.Question{}).
		Where("asker_id = ?", userID).
		Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计我发布的问题失败: %w", err)
	}

	err := r.db.WithContext(ctx).
		Table("questions AS q").
		Select(`
			q.id,
			q.description,
			q.status,
			q.expires_at,
			q.created_at,
			q.settled_at,
			q.contested_at,
			(SELECT COUNT(v.id)
			 FROM votes AS v
			 WHERE v.question_id = q.id) AS total_votes
		`).
		Where("q.asker_id = ?", userID).
		Order("q.created_at DESC, q.id DESC").
		Limit(limit).
		Offset(offset).
		Scan(&list).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询我发布的问题失败: %w", err)
	}

	return list, total, nil
}
